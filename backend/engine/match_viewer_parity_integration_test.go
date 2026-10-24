package engine

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/resources"
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

	// friendly_lineup_substitutions is only created by resources/sql/history/init59_friendly_lineup_substitutions.sql,
	// which neither init.sql nor the auto-applied migrations carry, so a brand-new database lacks it and
	// simulateFriendly fails on it. It is idempotent (CREATE TABLE IF NOT EXISTS), so apply it here to
	// give the friendly test the table a long-running database already has.
	history, err := resources.SQLFiles.ReadFile("sql/history/init59_friendly_lineup_substitutions.sql")
	if err != nil {
		t.Fatalf("reading init59_friendly_lineup_substitutions.sql: %s", err)
	}
	if _, err := db.Exec(string(history)); err != nil {
		t.Fatalf("applying init59_friendly_lineup_substitutions.sql: %s", err)
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
	for i, instant := range instants {
		if instant.Minute != i+1 {
			t.Fatalf("instant %d has minute %d, want one instant per consecutive minute starting at 1", i, instant.Minute)
		}
	}
	if len(instants) != 90 {
		t.Errorf("expected 90 instants for a full-length tie, got %d", len(instants))
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
