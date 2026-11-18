package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// matchViewerParityEnv opens the integration database (STRIKER_TEST_DB_DSN; the test is skipped
// when unset) and returns helpers to seed two playable teams: a country, a home team with a
// stadium, an away team, and 16 outfield-capable players each.
func matchViewerParityEnv(t *testing.T) (*common.DatabaseContext, *sql.DB, func(country string) (home int64, away int64)) {
	t.Helper()
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	dbCtx := &common.DatabaseContext{Db: db, LiveHub: common.NewLiveMatchHub(), ObjectCache: &memoryObjectCache{values: map[string]string{}}}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	leagueSeq := 0
	seedTeam := func(country string, name string, stadium string) int64 {
		t.Helper()
		leagueSeq++
		leagueId := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, $2, 'L', 1) RETURNING league_id`, country, leagueSeq)
		teamId := scan(`INSERT INTO teams (name, country, league_id, stadium_name) VALUES ($1, $2, $3, NULLIF($4, '')) RETURNING team_id`,
			name, country, leagueId, stadium)
		for i := 0; i < 16; i++ {
			keeper := 10
			if i == 0 {
				keeper = 70
			}
			if _, err := db.Exec(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield, dribbling, shot,
				header, quick, creative, technical, avatar, age, shirt_number, stamina, form)
				VALUES ($1, 'P', $2, 0, $3, 60, 60, 60, 60, 60, 60, false, false, false, false, '{}', 25, $4, 80, 50)`,
				teamId, name+strconv.Itoa(i), keeper, i+1); err != nil {
				t.Fatalf("seeding player: %s", err)
			}
		}
		lineup, err := common.BestLineupForTeam(dbCtx, teamId)
		if err != nil {
			t.Fatalf("BestLineupForTeam: %s", err)
		}
		if err := common.SetDefaultLineup(dbCtx, teamId, lineup.Positions); err != nil {
			t.Fatalf("SetDefaultLineup: %s", err)
		}
		return teamId
	}
	seed := func(country string) (int64, int64) {
		t.Helper()
		if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country); err != nil {
			t.Fatalf("seeding country: %s", err)
		}
		return seedTeam(country, country+" Home", "Estadio Norte"), seedTeam(country, country+" Away", "")
	}
	return dbCtx, db, seed
}

// memoryObjectCache is an in-memory common.ObjectCache for tests; locked because the engine
// simulates matches from several goroutines at once.
type memoryObjectCache struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memoryObjectCache) GetValue(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.values[key], nil
}

func (m *memoryObjectCache) SetValue(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
}

func (m *memoryObjectCache) SetValueLongDuration(key string, value string) {
	m.SetValue(key, value)
}

func (m *memoryObjectCache) SetValueNoExpiry(key string, value string) {
	m.SetValue(key, value)
}

func (m *memoryObjectCache) SetValueFor(key string, value string, _ time.Duration) {
	m.SetValue(key, value)
}

func (m *memoryObjectCache) DeleteValue(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
}

// TestSimulateCupTie_StoresInstantsAndFlavor pins that a cup tie persists what a league match
// does: one zone instant per simulated minute (so the Tácticas tab can show the zone duel) and its
// report flavor, with the home stadium name read back for the viewer.
func TestSimulateCupTie_StoresInstantsAndFlavor(t *testing.T) {
	dbCtx, db, seed := matchViewerParityEnv(t)
	home, away := seed("zv1")

	var cupId, tieId int64
	if err := db.QueryRow(common.InsertCupStmtSql, "zv1", 1, "Test Cup").Scan(&cupId); err != nil {
		t.Fatalf("inserting cup: %s", err)
	}
	matchDate := time.Now().Add(-time.Hour)
	if err := db.QueryRow(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date)
		VALUES ($1, 1, 1, $2, $3, $4) RETURNING tie_id`, cupId, home, away, matchDate).Scan(&tieId); err != nil {
		t.Fatalf("inserting tie: %s", err)
	}
	tie, err := common.GetCupTieById(dbCtx, tieId)
	if err != nil {
		t.Fatalf("GetCupTieById: %s", err)
	}

	if err := simulateCupTie(dbCtx, tie); err != nil {
		t.Fatalf("simulateCupTie: %s", err)
	}

	details, err := common.GetCupTieDetails(dbCtx, tieId)
	if err != nil {
		t.Fatalf("GetCupTieDetails: %s", err)
	}
	if details.MatchInstants == nil || len(*details.MatchInstants) == 0 {
		t.Fatalf("expected the tie to store its per-minute instants")
	}
	instants := *details.MatchInstants
	assertStoredAddedTime(t, db, "cup_ties", "tie_id", tieId, instants, details.Events)
	// "Cup tie level after added time" (match-simulation): the shoot-out is decided on the score
	// after the last added minute.
	var homeGoals, awayGoals int
	var wentToPenalties bool
	if err := db.QueryRow(`SELECT home_goals, away_goals, went_to_penalties FROM cup_ties WHERE tie_id = $1`, tieId).
		Scan(&homeGoals, &awayGoals, &wentToPenalties); err != nil {
		t.Fatalf("reading the tie result: %s", err)
	}
	if wentToPenalties != (homeGoals == awayGoals) {
		t.Errorf("went_to_penalties = %v for a %d-%d tie", wentToPenalties, homeGoals, awayGoals)
	}
	if details.Weather == "" || details.RefereeName == "" || details.Assistant1Name == "" || details.Assistant2Name == "" {
		t.Errorf("expected stored weather and referees, got %+v", details)
	}
	if details.Attendance <= 0 {
		t.Errorf("expected a stored attendance, got %d", details.Attendance)
	}
	if details.HomeStadiumName != "Estadio Norte" {
		t.Errorf("expected the home stadium name, got %q", details.HomeStadiumName)
	}
	assertStoredRatingsRebuildFromInstants(t, details.PlayerRatings, *details.HomeTeamId, *details.AwayTeamId, instants, details.Events)
}

// assertStoredAddedTime pins a just-simulated fixture's added time (openspec/specs/stoppage-time)
// against what it stored: both halves' added minutes on the fixture row (1-5 each), one instant per
// simulated minute in match order (1..45, 45+1..45+A1, 46..90, 90+1..90+A2), and the two ADDED_TIME
// boards announcing those same minutes. A match abandoned for lack of players is left out.
func assertStoredAddedTime(t *testing.T, db *sql.DB, table string, idColumn string, id int64, instants []common.MatchInstant, events []common.Event) {
	t.Helper()
	for _, e := range events {
		if e.EventType == common.MATCH_ABANDONED {
			return
		}
	}
	var first, second int
	if err := db.QueryRow("SELECT first_half_added_time, second_half_added_time FROM "+table+" WHERE "+idColumn+" = $1", id).Scan(&first, &second); err != nil {
		t.Fatalf("reading %s added time: %s", table, err)
	}
	if first < MinAddedMinutes || first > MaxAddedFirstHalf || second < MinAddedMinutes || second > MaxAddedSecondHalf {
		t.Fatalf("%s added time %d and %d, want 1-5 each", table, first, second)
	}
	want := make([][2]int, 0, 90+first+second)
	for m := 1; m <= 45; m++ {
		want = append(want, [2]int{m, 0})
	}
	for a := 1; a <= first; a++ {
		want = append(want, [2]int{45, a})
	}
	for m := 46; m <= 90; m++ {
		want = append(want, [2]int{m, 0})
	}
	for a := 1; a <= second; a++ {
		want = append(want, [2]int{90, a})
	}
	if len(instants) != len(want) {
		t.Fatalf("%s: %d instants, want %d (90 + %d + %d)", table, len(instants), len(want), first, second)
	}
	for i, w := range want {
		if instants[i].Minute != w[0] || instants[i].AddedMinute != w[1] {
			t.Fatalf("%s: instant %d is %d+%d, want %d+%d", table, i, instants[i].Minute, instants[i].AddedMinute, w[0], w[1])
		}
	}
	boards := make([][2]int, 0, 2)
	for _, e := range events {
		if e.EventType == common.ADDED_TIME && e.AddedTime != nil {
			boards = append(boards, [2]int{e.Minute, *e.AddedTime})
		}
	}
	if len(boards) != 2 || boards[0] != [2]int{45, first} || boards[1] != [2]int{90, second} {
		t.Errorf("%s: boards %v, want +%d at 45 and +%d at 90", table, boards, first, second)
	}
}

// assertStoredRatingsRebuildFromInstants pins the stored ratings of a just-simulated fixture
// against its stored instants and events: every instant carries contributions, every player any
// of them lists (or any rating event names) has exactly one rating, and recomputing them from the
// stored data - what the frontend does when it scrubs back to the final minute - gives the very
// same records.
func assertStoredRatingsRebuildFromInstants(t *testing.T, stored []common.PlayerRating, home int64, away int64,
	instants []common.MatchInstant, events []common.Event) {
	t.Helper()
	if len(stored) == 0 {
		t.Fatal("expected stored player ratings")
	}
	listed := make(map[int64]bool)
	for _, instant := range instants {
		if instant.PlayerContributions == nil {
			t.Fatalf("instant at minute %d has no player contributions", instant.Minute)
		}
		for id := range instant.PlayerContributions.Home {
			listed[id] = true
		}
		for id := range instant.PlayerContributions.Away {
			listed[id] = true
		}
	}
	rated := make(map[int64]bool)
	for _, r := range stored {
		if rated[r.PlayerId] {
			t.Fatalf("player %d rated twice", r.PlayerId)
		}
		rated[r.PlayerId] = true
	}
	for id := range listed {
		if !rated[id] {
			t.Errorf("player %d was on the pitch but has no rating", id)
		}
	}
	recomputed := common.ComputePlayerRatings(home, away, nil, nil, instants, events, common.MatchFinalMinute(instants, events))
	gotJson, _ := json.Marshal(stored)
	wantJson, _ := json.Marshal(recomputed)
	if string(gotJson) != string(wantJson) {
		t.Errorf("stored ratings differ from a recomputation:\n stored     %s\n recomputed %s", gotJson, wantJson)
	}
}

// TestSimulateFriendly_StoredAttendanceMatchesTicketRevenue pins that a friendly's stored crowd is
// the same figure its ticket revenue was computed from, and that its weather and referees are
// stored too (the ordinary friendly has no World Cup group label).
func TestSimulateFriendly_StoredAttendanceMatchesTicketRevenue(t *testing.T) {
	dbCtx, db, seed := matchViewerParityEnv(t)
	home, away := seed("zv2")

	var matchId int64
	if err := db.QueryRow(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, $3) RETURNING match_id`,
		home, away, time.Now().Add(-time.Hour)).Scan(&matchId); err != nil {
		t.Fatalf("inserting friendly: %s", err)
	}
	friendly, err := common.GetFriendlyMatchById(dbCtx, matchId)
	if err != nil {
		t.Fatalf("GetFriendlyMatchById: %s", err)
	}

	if err := simulateFriendly(dbCtx, friendly); err != nil {
		t.Fatalf("simulateFriendly: %s", err)
	}

	played, err := common.GetFriendlyMatchById(dbCtx, matchId)
	if err != nil {
		t.Fatalf("GetFriendlyMatchById after play: %s", err)
	}
	if played.Weather == "" || played.RefereeName == "" || played.Assistant1Name == "" || played.Assistant2Name == "" {
		t.Errorf("expected stored weather and referees, got %+v", played)
	}
	if played.HomeStadiumName != "Estadio Norte" {
		t.Errorf("expected the home stadium name, got %q", played.HomeStadiumName)
	}
	if played.WorldCupGroupName != "" || played.WorldCupRound != 0 {
		t.Errorf("an ordinary friendly has no World Cup label, got %q / %d", played.WorldCupGroupName, played.WorldCupRound)
	}
	if err := common.EnrichPlayedFriendly(dbCtx, played); err != nil {
		t.Fatalf("EnrichPlayedFriendly: %s", err)
	}
	if played.MatchInstants == nil || played.Events == nil {
		t.Fatal("expected the played friendly's instants and events")
	}
	assertStoredRatingsRebuildFromInstants(t, played.PlayerRatings, home, away, *played.MatchInstants, *played.Events)
	assertStoredAddedTime(t, db, "friendly_matches", "match_id", matchId, *played.MatchInstants, *played.Events)

	var description string
	if err := db.QueryRow(`SELECT description FROM team_transactions WHERE team_id = $1 AND reason = $2 ORDER BY 1 LIMIT 1`,
		home, common.TRANSACTION_TICKET_REVENUE).Scan(&description); err != nil {
		t.Fatalf("reading the ticket revenue transaction: %s", err)
	}
	want := fmt.Sprintf("(%d attendance", played.Attendance)
	if played.Attendance <= 0 || !strings.Contains(description, want) {
		t.Errorf("stored attendance %d must be the one in the ticket revenue description %q", played.Attendance, description)
	}
}

// TestSimulateMatch_StoresPlayerRatings pins that a league match stores its per-player ratings
// with the result and that WeekMatchById returns them as stored, with the id -> rating map
// holding only the rated players.
func TestSimulateMatch_StoresPlayerRatings(t *testing.T) {
	e := newSimEnv(t, "zv3")
	matchId := e.leagueMatch(0)
	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}

	match, err := common.WeekMatchById(e.dbCtx, matchId)
	if err != nil || match == nil {
		t.Fatalf("WeekMatchById: %v, %v", match, err)
	}
	if match.MatchInstants == nil || match.Events == nil {
		t.Fatal("expected the match's instants and events")
	}
	assertStoredRatingsRebuildFromInstants(t, match.PlayerRatings, e.home, e.away, *match.MatchInstants, *match.Events)
	assertStoredAddedTime(t, e.db, "matches", "match_id", matchId, *match.MatchInstants, *match.Events)
	if match.Ratings == nil {
		t.Fatal("expected the id -> rating map")
	}
	for _, r := range match.PlayerRatings {
		rating, inMap := (*match.Ratings)[r.PlayerId]
		if inMap != r.Rated || (inMap && rating != r.Rating) {
			t.Errorf("player %d: map has %v (%v), record is rated=%v %v", r.PlayerId, rating, inMap, r.Rated, r.Rating)
		}
		if _, named := (*match.HomePlayers)[r.PlayerId]; !named {
			if _, named := (*match.AwayPlayers)[r.PlayerId]; !named {
				t.Errorf("rated player %d is missing from both players maps", r.PlayerId)
			}
		}
	}
}

// TestWeekMatchById_LegacyMatchRatesSubstitutedOffScorer pins the read-time fallback for a match
// simulated before stored ratings (no rating rows, no contributions): the starter who scored and
// was then replaced keeps his goal and his 60 minutes, and the substitute is rated separately with
// the remaining 30 and his name resolved.
func TestWeekMatchById_LegacyMatchRatesSubstitutedOffScorer(t *testing.T) {
	e := newSimEnv(t, "zv4")
	matchId := e.leagueMatch(0)
	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}
	lineups, err := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatalf("GetLineupsForMatch: %s", err)
	}
	var starter int64
	for _, pos := range (*lineups)[e.home].Positions {
		if pos.Position != common.GOALKEEPER {
			starter = pos.PlayerId
			break
		}
	}
	bench := e.scan(`SELECT player_id FROM players WHERE team_id = $1 AND player_id <> ALL($2) ORDER BY player_id LIMIT 1`,
		e.home, common.LineupPlayerIds((*lineups)[e.home]))

	// Turn it into a pre-init89 match: no stored ratings, no contributions, and known events.
	e.exec(`DELETE FROM match_player_ratings WHERE kind = $1 AND match_id = $2`, common.SimulationKindLeague, matchId)
	e.exec(`UPDATE match_instants SET player_contributions = NULL WHERE match_id = $1`, matchId)
	e.exec(`DELETE FROM match_events WHERE match_id = $1`, matchId)
	e.exec(`UPDATE matches SET home_goals = 1, away_goals = 0 WHERE match_id = $1`, matchId)
	e.exec(common.InsertMatchEventStmtSql, matchId, common.GOAL, e.home, e.away, 20, fmt.Sprintf("[%d]", starter), `["S"]`, `[]`, `[]`, 1, 0, common.SET_PIECE_NONE, nil, 0, nil)
	e.exec(common.InsertMatchEventStmtSql, matchId, common.SUBSTITUTION, e.home, e.away, 60, fmt.Sprintf("[%d, %d]", starter, bench), `["S", "B"]`, `[]`, `[]`, 1, 0, common.SET_PIECE_NONE, nil, 0, nil)

	match, err := common.WeekMatchById(e.dbCtx, matchId)
	if err != nil || match == nil {
		t.Fatalf("WeekMatchById: %v, %v", match, err)
	}
	var scorer, sub *common.PlayerRating
	for i := range match.PlayerRatings {
		switch match.PlayerRatings[i].PlayerId {
		case starter:
			scorer = &match.PlayerRatings[i]
		case bench:
			sub = &match.PlayerRatings[i]
		}
	}
	if scorer == nil || scorer.Goals != 1.4 || scorer.Minutes != 60 || scorer.Zone != 0 || scorer.Own != nil {
		t.Fatalf("substituted-off scorer: %+v", scorer)
	}
	if sub == nil || sub.Minutes != 30 || sub.CameOn == nil || *sub.CameOn != 60 || sub.Result != 0.1 {
		t.Fatalf("substitute: %+v", sub)
	}
	if _, ok := (*match.HomePlayers)[bench]; !ok {
		t.Error("the substitute must be in the home players map")
	}
	if _, ok := (*match.Ratings)[starter]; !ok {
		t.Error("the substituted-off scorer must keep his rating in the id -> rating map")
	}
}
