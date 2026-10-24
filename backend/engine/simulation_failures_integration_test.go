package engine

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// These tests run against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset) and cover
// openspec harden-stuck-match-recovery: the atomic result save, the failure record, the default
// 0-0 result, the durable post-match effects and the standings repair.

type simEnv struct {
	t       *testing.T
	dbCtx   *common.DatabaseContext
	db      *sql.DB
	country string
	league  int64
	home    int64
	away    int64
}

// newSimEnv seeds two playable teams (16 players each, a default lineup, wages of 1000) in one
// league, with a classification row each for season 1.
func newSimEnv(t *testing.T, country string) *simEnv {
	t.Helper()
	dbCtx, db, seed := matchViewerParityEnv(t)
	home, away := seed(country)
	e := &simEnv{t: t, dbCtx: dbCtx, db: db, country: country, home: home, away: away}
	e.league = e.scan(`SELECT league_id FROM teams WHERE team_id = $1`, home)
	e.exec(`UPDATE teams SET league_id = $1 WHERE team_id = $2`, e.league, away)
	e.exec(`UPDATE players SET wage = 1000 WHERE team_id IN ($1, $2)`, home, away)
	for _, teamId := range []int64{home, away} {
		e.exec(`INSERT INTO classification (league_id, team_id, season) VALUES ($1, $2, 1)`, e.league, teamId)
	}
	return e
}

func (e *simEnv) exec(query string, args ...any) {
	e.t.Helper()
	if _, err := e.db.Exec(query, args...); err != nil {
		e.t.Fatalf("%s: %s", query, err)
	}
}

func (e *simEnv) scan(query string, args ...any) int64 {
	e.t.Helper()
	var v int64
	if err := e.db.QueryRow(query, args...).Scan(&v); err != nil {
		e.t.Fatalf("%s: %s", query, err)
	}
	return v
}

// leagueMatch inserts an unplayed league match that kicked off three hours ago, so its replay
// (and the effects deferred to its completion) is over as soon as the broadcast catches up.
func (e *simEnv) leagueMatch(attempts int) int64 {
	return e.scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, simulation_attempts)
		VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, $4) RETURNING match_id`, e.league, e.home, e.away, attempts)
}

func (e *simEnv) weekMatch(matchId int64) *common.WeekMatch {
	e.t.Helper()
	var m common.WeekMatch
	if err := e.db.QueryRow(`SELECT match_id, league_id, season, week_number, match_date, home_team_id, away_team_id
		FROM matches WHERE match_id = $1`, matchId).Scan(&m.MatchId, &m.DivisionId, &m.Season, &m.WeekNumber, &m.MatchDate, &m.Home, &m.Away); err != nil {
		e.t.Fatalf("reading match %d: %s", matchId, err)
	}
	return &m
}

func (e *simEnv) transactions(teamId int64, reason int) int64 {
	return e.scan(`SELECT count(*) FROM team_transactions WHERE team_id = $1 AND reason = $2`, teamId, reason)
}

func (e *simEnv) matchesPlayedInStandings(teamId int64) int64 {
	return e.scan(`SELECT matches FROM classification WHERE league_id = $1 AND team_id = $2`, e.league, teamId)
}

func (e *simEnv) cupTie(attempts int) (cupId int64, tieId int64) {
	cupId = e.scan(`INSERT INTO cups (country, season, name) VALUES ($1, 1, 'Cup') RETURNING cup_id`, e.country)
	tieId = e.scan(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date, simulation_attempts)
		VALUES ($1, 2, 1, $2, $3, now() - interval '3 hours', $4) RETURNING tie_id`, cupId, e.home, e.away, attempts)
	return cupId, tieId
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// Two simulations of one match can both reach the result save (a live pass and the recovery job,
// or two replicas): the first save applies, the second is rejected, and standings, wages and gate
// receipts are booked once.
func TestSimulateMatch_ConcurrentSimulationsBookOnce(t *testing.T) {
	e := newSimEnv(t, "zs1")
	matchId := e.leagueMatch(1)
	match := e.weekMatch(matchId)

	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m := *match
			_, errs[i] = simulateMatch(e.dbCtx, &m)
		}(i)
	}
	wg.Wait()

	saved, rejected := 0, 0
	for _, err := range errs {
		switch err {
		case nil:
			saved++
		case common.ErrAlreadyPlayed:
			rejected++
		default:
			t.Fatalf("unexpected simulation error: %s", err)
		}
	}
	if saved != 1 || rejected != 1 {
		t.Fatalf("saved %d and rejected %d simulations, want 1 and 1", saved, rejected)
	}
	for _, teamId := range []int64{e.home, e.away} {
		if played := e.matchesPlayedInStandings(teamId); played != 1 {
			t.Errorf("team %d has %d matches in the standings, want 1", teamId, played)
		}
		if wages := e.transactions(teamId, common.TRANSACTION_WAGES); wages != 1 {
			t.Errorf("team %d was charged wages %d times, want once", teamId, wages)
		}
	}
	if gate := e.transactions(e.home, common.TRANSACTION_TICKET_REVENUE); gate != 1 {
		t.Errorf("gate receipts booked %d times, want once", gate)
	}
}

// A completed replay applies the pending effects once and leaves no pending record behind.
func TestSimulateMatch_CompletedReplayAppliesPendingEffectsOnce(t *testing.T) {
	e := newSimEnv(t, "zs2")
	// An injured player of the home side: the effects count this match against the injury.
	playerId := e.scan(`SELECT player_id FROM players WHERE team_id = $1 ORDER BY player_id LIMIT 1`, e.home)
	e.exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 3)`, playerId, common.UNAVAILABLE_INJURY)
	matchId := e.leagueMatch(1)

	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}
	waitFor(t, "the replay to apply and remove the pending effects", func() bool {
		return e.scan(`SELECT count(*) FROM pending_post_match_effects WHERE kind = 'league' AND match_id = $1`, matchId) == 0
	})
	waitFor(t, "the availability countdown", func() bool {
		return e.scan(`SELECT COALESCE((SELECT matches_remaining FROM player_unavailability WHERE player_id = $1), 0)`, playerId) == 2
	})

	// The recovery job finds nothing left for this match, so the countdown is not applied again.
	ApplyDuePostMatchEffects(e.dbCtx)
	if left := e.scan(`SELECT COALESCE((SELECT matches_remaining FROM player_unavailability WHERE player_id = $1), 0)`, playerId); left != 2 {
		t.Errorf("injury has %d matches remaining after the recovery pass, want 2 (applied exactly once)", left)
	}
}

// A pending record due a minute ago is applied by exactly one of two concurrent recovery drains,
// and one due in the future is left alone.
func TestApplyDuePostMatchEffects_ConcurrentDrainsApplyOnce(t *testing.T) {
	e := newSimEnv(t, "zs3")
	playerId := e.scan(`SELECT player_id FROM players WHERE team_id = $1 ORDER BY player_id LIMIT 1`, e.home)
	e.exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 3)`, playerId, common.UNAVAILABLE_INJURY)
	dueMatch, futureMatch := e.leagueMatch(1), e.leagueMatch(1)
	payload := fmt.Sprintf(`{"home":%d,"away":%d,"skipSkillGrowth":true}`, e.home, e.away)
	e.exec(`INSERT INTO pending_post_match_effects (kind, match_id, due_at, payload) VALUES ('league', $1, now() - interval '1 minute', $2)`, dueMatch, payload)
	e.exec(`INSERT INTO pending_post_match_effects (kind, match_id, due_at, payload) VALUES ('league', $1, now() + interval '1 hour', $2)`, futureMatch, payload)

	runConcurrently(2, func() { ApplyDuePostMatchEffects(e.dbCtx) })

	if left := e.scan(`SELECT COALESCE((SELECT matches_remaining FROM player_unavailability WHERE player_id = $1), 0)`, playerId); left != 2 {
		t.Errorf("injury has %d matches remaining after two concurrent drains, want 2 (one drain applied it)", left)
	}
	if due := e.scan(`SELECT count(*) FROM pending_post_match_effects WHERE match_id = $1`, dueMatch); due != 0 {
		t.Errorf("the due record is still pending")
	}
	if future := e.scan(`SELECT count(*) FROM pending_post_match_effects WHERE match_id = $1`, futureMatch); future != 1 {
		t.Errorf("the future record was taken")
	}
}

// A panicking or failing simulation leaves its claim released with the error stored.
func TestRunClaimedSimulation_PanicAndErrorReleaseTheClaim(t *testing.T) {
	e := newSimEnv(t, "zs4")
	panicked, failed, lost := e.leagueMatch(1), e.leagueMatch(1), e.leagueMatch(1)
	e.exec(`UPDATE matches SET in_simulation = true WHERE match_id IN ($1, $2, $3)`, panicked, failed, lost)

	runClaimedSimulation(e.dbCtx, common.SimulationKindLeague, panicked, func() error { panic("simulation exploded") })
	runClaimedSimulation(e.dbCtx, common.SimulationKindLeague, failed, func() error { return fmt.Errorf("cannot read lineup") })
	// Losing the race to another simulation is not a failure: nothing is recorded.
	runClaimedSimulation(e.dbCtx, common.SimulationKindLeague, lost, func() error { return common.ErrAlreadyPlayed })

	check := func(id int64, wantError string) {
		t.Helper()
		var inSimulation bool
		var stored sql.NullString
		if err := e.db.QueryRow(`SELECT in_simulation, last_simulation_error FROM matches WHERE match_id = $1`, id).Scan(&inSimulation, &stored); err != nil {
			t.Fatalf("reading match %d: %s", id, err)
		}
		if wantError == "" {
			if !inSimulation || stored.Valid {
				t.Errorf("match %d: in_simulation=%v error=%v, want the claim and error untouched", id, inSimulation, stored)
			}
			return
		}
		if inSimulation || stored.String != wantError {
			t.Errorf("match %d: in_simulation=%v error=%q, want released with %q", id, inSimulation, stored.String, wantError)
		}
	}
	check(panicked, "panic: simulation exploded")
	check(failed, "cannot read lineup")
	check(lost, "")
}

// Two replicas claiming one failing fixture count one attempt and only one gets to simulate it.
func TestGetSimulableMatches_ConcurrentClaimsCountOneAttempt(t *testing.T) {
	e := newSimEnv(t, "zs5")
	matchId := e.leagueMatch(0)
	e.exec(`UPDATE matches SET simulation_attempts = 1, last_simulation_attempt_at = now() - interval '11 minutes' WHERE match_id = $1`, matchId)

	var mu sync.Mutex
	claims := 0
	runConcurrently(replicas, func() {
		matches, err := common.GetSimulableMatches(e.dbCtx)
		if err != nil {
			t.Errorf("GetSimulableMatches: %s", err)
			return
		}
		for _, m := range *matches {
			if m.MatchId == matchId {
				mu.Lock()
				claims++
				mu.Unlock()
			}
		}
	})

	if claims != 1 {
		t.Errorf("%d replicas claimed the fixture, want exactly 1", claims)
	}
	if attempts := e.scan(`SELECT simulation_attempts FROM matches WHERE match_id = $1`, matchId); attempts != 2 {
		t.Errorf("the fixture has %d attempts, want 2 (one more for the single claim)", attempts)
	}

	// The stuck re-claim is the same: once per claim, whichever replica gets it.
	e.exec(`UPDATE matches SET in_simulation_since = now() - interval '30 minutes' WHERE match_id = $1`, matchId)
	claims = 0
	runConcurrently(replicas, func() {
		matches, err := common.GetStuckMatches(e.dbCtx)
		if err != nil {
			t.Errorf("GetStuckMatches: %s", err)
			return
		}
		for _, m := range *matches {
			if m.MatchId == matchId {
				mu.Lock()
				claims++
				mu.Unlock()
			}
		}
	})
	if claims != 1 {
		t.Errorf("%d replicas re-claimed the stuck fixture, want exactly 1", claims)
	}
	if attempts := e.scan(`SELECT simulation_attempts FROM matches WHERE match_id = $1`, matchId); attempts != 3 {
		t.Errorf("the fixture has %d attempts, want 3", attempts)
	}
}

// A league match at 3 attempts becomes 0-0 with a point each and wages charged, no gate or
// sponsorship; the match carries only the default-result event; its effects are due at once.
func TestSettleDefaultResults_LeagueMatch(t *testing.T) {
	e := newSimEnv(t, "zs6")
	matchId := e.leagueMatch(3)

	SettleDefaultResults(e.dbCtx)

	if played := e.scan(`SELECT count(*) FROM matches WHERE match_id = $1 AND played AND NOT in_simulation AND home_goals = 0 AND away_goals = 0`, matchId); played != 1 {
		t.Fatalf("match %d was not settled 0-0", matchId)
	}
	for _, teamId := range []int64{e.home, e.away} {
		if points := e.scan(`SELECT points FROM classification WHERE league_id = $1 AND team_id = $2`, e.league, teamId); points != 1 {
			t.Errorf("team %d has %d points, want 1", teamId, points)
		}
		if draws := e.scan(`SELECT draws FROM classification WHERE league_id = $1 AND team_id = $2`, e.league, teamId); draws != 1 {
			t.Errorf("team %d has %d draws, want 1", teamId, draws)
		}
		if wages := e.transactions(teamId, common.TRANSACTION_WAGES); wages != 1 {
			t.Errorf("team %d was charged wages %d times, want once", teamId, wages)
		}
		if income := e.transactions(teamId, common.TRANSACTION_TICKET_REVENUE) + e.transactions(teamId, common.TRANSACTION_SPONSORSHIP); income != 0 {
			t.Errorf("team %d got %d gate or sponsorship payments for a match nobody played", teamId, income)
		}
	}
	if events := e.scan(`SELECT count(*) FROM match_events WHERE match_id = $1 AND event_type = $2`, matchId, common.MATCH_DEFAULT_RESULT); events != 1 {
		t.Errorf("%d default-result events, want 1", events)
	}
	if others := e.scan(`SELECT count(*) FROM match_events WHERE match_id = $1`, matchId); others != 1 {
		t.Errorf("%d events stored, want only the default-result one", others)
	}
	if instants := e.scan(`SELECT count(*) FROM match_instants WHERE match_id = $1`, matchId); instants != 0 {
		t.Errorf("%d instants stored, want none", instants)
	}
	if due := e.scan(`SELECT count(*) FROM pending_post_match_effects WHERE kind = 'league' AND match_id = $1 AND due_at <= now()`, matchId); due != 1 {
		t.Errorf("the default result's post-match effects are not due at once")
	}

	// Settling again changes nothing.
	SettleDefaultResults(e.dbCtx)
	if played := e.matchesPlayedInStandings(e.home); played != 1 {
		t.Errorf("home has %d matches in the standings after a second pass, want 1", played)
	}
}

func TestSettleDefaultResults_CupTieGoesToPenaltiesAndPaysThePrize(t *testing.T) {
	e := newSimEnv(t, "zs7")
	_, tieId := e.cupTie(3)

	SettleDefaultResults(e.dbCtx)

	var played, penalties bool
	var homeGoals, awayGoals int
	var winner sql.NullInt64
	if err := e.db.QueryRow(`SELECT played, went_to_penalties, home_goals, away_goals, winner_team_id FROM cup_ties WHERE tie_id = $1`, tieId).
		Scan(&played, &penalties, &homeGoals, &awayGoals, &winner); err != nil {
		t.Fatalf("reading the tie: %s", err)
	}
	if !played || !penalties || homeGoals != 0 || awayGoals != 0 || !winner.Valid || (winner.Int64 != e.home && winner.Int64 != e.away) {
		t.Fatalf("tie after the default: played=%v penalties=%v %d-%d winner=%v", played, penalties, homeGoals, awayGoals, winner)
	}
	if prizes := e.transactions(winner.Int64, common.TRANSACTION_PRIZE_MONEY); prizes != 1 {
		t.Errorf("the winner was paid the round prize %d times, want once", prizes)
	}
	loser := e.home
	if winner.Int64 == e.home {
		loser = e.away
	}
	if prizes := e.transactions(loser, common.TRANSACTION_PRIZE_MONEY); prizes != 0 {
		t.Errorf("the loser was paid %d prizes", prizes)
	}
	if events := e.scan(`SELECT count(*) FROM cup_tie_events WHERE tie_id = $1 AND event_type = $2`, tieId, common.MATCH_DEFAULT_RESULT); events != 1 {
		t.Errorf("%d default-result events, want 1", events)
	}
}

func TestSettleDefaultResults_WorldCupGroupFriendlyIsADraw(t *testing.T) {
	e := newSimEnv(t, "zs8")
	friendlyId := e.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, simulation_attempts)
		VALUES ($1, $2, now() - interval '3 hours', 3) RETURNING match_id`, e.home, e.away)
	worldCupId := e.scan(`INSERT INTO world_cups (season, name) VALUES (1, 'World Cup') RETURNING world_cup_id`)
	groupId := e.scan(`INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, 'A') RETURNING group_id`, worldCupId)
	e.exec(`INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, 1, $2)`, groupId, friendlyId)
	for _, teamId := range []int64{e.home, e.away} {
		e.exec(`INSERT INTO world_cup_group_teams (group_id, team_id) VALUES ($1, $2)`, groupId, teamId)
	}

	SettleDefaultResults(e.dbCtx)

	if played := e.scan(`SELECT count(*) FROM friendly_matches WHERE match_id = $1 AND played AND home_goals = 0 AND away_goals = 0`, friendlyId); played != 1 {
		t.Fatalf("friendly %d was not settled 0-0", friendlyId)
	}
	for _, teamId := range []int64{e.home, e.away} {
		if drawn := e.scan(`SELECT drawn FROM world_cup_group_teams WHERE group_id = $1 AND team_id = $2`, groupId, teamId); drawn != 1 {
			t.Errorf("team %d has %d group draws, want 1", teamId, drawn)
		}
	}
	if gate := e.transactions(e.home, common.TRANSACTION_TICKET_REVENUE); gate != 0 {
		t.Errorf("a gate was split for a friendly nobody played")
	}
}

// Replicas settling at once settle each fixture once: standings, wages and the prize are booked once.
func TestSettleDefaultResults_ConcurrentReplicasSettleOnce(t *testing.T) {
	e := newSimEnv(t, "zs9")
	matchId := e.leagueMatch(3)
	_, tieId := e.cupTie(3)

	runConcurrently(replicas, func() { SettleDefaultResults(e.dbCtx) })

	if played := e.scan(`SELECT count(*) FROM matches WHERE match_id = $1 AND played`, matchId); played != 1 {
		t.Fatalf("match not settled")
	}
	for _, teamId := range []int64{e.home, e.away} {
		if played := e.matchesPlayedInStandings(teamId); played != 1 {
			t.Errorf("team %d has %d matches in the standings, want 1", teamId, played)
		}
		if wages := e.transactions(teamId, common.TRANSACTION_WAGES); wages != 1 {
			t.Errorf("team %d was charged wages %d times, want once", teamId, wages)
		}
	}
	if events := e.scan(`SELECT count(*) FROM match_events WHERE match_id = $1`, matchId); events != 1 {
		t.Errorf("%d events stored for the match, want 1", events)
	}
	if prizes := e.scan(`SELECT count(*) FROM team_transactions WHERE team_id IN ($1, $2) AND reason = $3`, e.home, e.away, common.TRANSACTION_PRIZE_MONEY); prizes != 1 {
		t.Errorf("the tie %d paid %d round prizes, want 1", tieId, prizes)
	}
}

// After the migration seeding, a stuck league match gets exactly one attempt, then is settled by
// default, and its country's season rollover proceeds.
func TestStuckMatchAfterMigration_OneAttemptThenDefaultThenRollover(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	country := uniqueCountryCode()
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1)`, country)
	leagueId := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, country)
	// Teams without players cannot be simulated, so every attempt at this match fails.
	teamA := scan(`INSERT INTO teams (name, country, league_id) VALUES ('A', $1, $2) RETURNING team_id`, country, leagueId)
	teamB := scan(`INSERT INTO teams (name, country, league_id) VALUES ('B', $1, $2) RETURNING team_id`, country, leagueId)
	exec(`INSERT INTO classification (league_id, team_id, season) VALUES ($1, $2, 1), ($1, $3, 1)`, leagueId, teamA, teamB)
	// Stuck for two hours with the 2 attempts the migration gives such a row.
	matchId := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, in_simulation, in_simulation_since, simulation_attempts)
		VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true, now() - interval '2 hours', 2) RETURNING match_id`, leagueId, teamA, teamB)

	advanceCountrySeasonIfComplete(dbCtx, country)
	if season := scan(`SELECT current_season FROM countries WHERE name = $1`, country); season != 1 {
		t.Fatalf("the season advanced to %d with a match still unplayed", season)
	}

	// The one retry on the fixed code: counted, fails, and the claim is released with the reason.
	FinishStuckSimulations(dbCtx)
	var attempts int
	var inSimulation bool
	var played bool
	var lastError sql.NullString
	if err := db.QueryRow(`SELECT simulation_attempts, in_simulation, played, last_simulation_error FROM matches WHERE match_id = $1`, matchId).
		Scan(&attempts, &inSimulation, &played, &lastError); err != nil {
		t.Fatalf("reading the match: %s", err)
	}
	if attempts != 3 || inSimulation || played || !lastError.Valid || lastError.String == "" {
		t.Fatalf("after the retry: attempts=%d in_simulation=%v played=%v error=%v; want 3, released, unplayed, with an error", attempts, inSimulation, played, lastError)
	}

	// No further attempt is spent on it; the default result settles it instead.
	FinishStuckSimulations(dbCtx)
	if again := scan(`SELECT simulation_attempts FROM matches WHERE match_id = $1`, matchId); again != 3 {
		t.Fatalf("the match got a fourth attempt (%d)", again)
	}
	SettleDefaultResults(dbCtx)
	if settled := scan(`SELECT count(*) FROM matches WHERE match_id = $1 AND played AND home_goals = 0 AND away_goals = 0`, matchId); settled != 1 {
		t.Fatalf("the match was not settled 0-0")
	}

	advanceCountrySeasonIfComplete(dbCtx, country)
	if season := scan(`SELECT current_season FROM countries WHERE name = $1`, country); season != 2 {
		t.Errorf("season is %d after the default result, want the rollover to have advanced it to 2", season)
	}
}

// A team showing 12 matches with 11 played is rebuilt to the 11-match totals, a consistent league
// is untouched, and of several concurrent starts only one runs the repair.
func TestRepairStandings_RebuildsDriftedRowsOnly(t *testing.T) {
	e := newSimEnv(t, "zr1")
	// 11 played matches: home wins 2-0 six times, draws 1-1 twice, loses 0-1 three times.
	results := [][2]int{{2, 0}, {2, 0}, {2, 0}, {2, 0}, {2, 0}, {2, 0}, {1, 1}, {1, 1}, {0, 1}, {0, 1}, {0, 1}}
	for _, r := range results {
		e.exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played, home_goals, away_goals)
			VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true, $4, $5)`, e.league, e.home, e.away, r[0], r[1])
	}
	// Home was double-counted once (a 12th, phantom match); away is consistent.
	e.exec(`UPDATE classification SET matches = 12, points = 23, wins = 7, draws = 2, losses = 3, goals_scored = 14, goals_received = 5
		WHERE league_id = $1 AND team_id = $2`, e.league, e.home)
	e.exec(`UPDATE classification SET matches = 11, points = 11, wins = 3, draws = 2, losses = 6, goals_scored = 5, goals_received = 14
		WHERE league_id = $1 AND team_id = $2`, e.league, e.away)

	e2 := newSimEnv(t, "zr2")
	e2.exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played, home_goals, away_goals)
		VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true, 3, 1)`, e2.league, e2.home, e2.away)
	e2.exec(`UPDATE classification SET matches = 1, points = 3, wins = 1, goals_scored = 3, goals_received = 1 WHERE league_id = $1 AND team_id = $2`, e2.league, e2.home)
	e2.exec(`UPDATE classification SET matches = 1, losses = 1, goals_scored = 1, goals_received = 3 WHERE league_id = $1 AND team_id = $2`, e2.league, e2.away)

	if mismatches, err := common.CountStandingsMismatches(e.dbCtx); err != nil || mismatches < 1 {
		t.Fatalf("CountStandingsMismatches = %d, %v; want at least the drifted row", mismatches, err)
	}

	e.exec(`DELETE FROM engine_job_runs WHERE job_name = $1`, standingsRepairSlot)
	rebuilt := make([]int64, replicas)
	var wg sync.WaitGroup
	for i := range rebuilt {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rebuilt[i] = RepairStandingsOnce(e.dbCtx)
		}(i)
	}
	wg.Wait()
	runners := 0
	for _, n := range rebuilt {
		if n > 0 {
			runners++
		}
	}
	if runners != 1 {
		t.Errorf("%d of %d concurrent starts ran the repair (%v), want exactly 1", runners, replicas, rebuilt)
	}

	type row struct{ matches, points, wins, draws, losses, scored, conceded int64 }
	read := func(leagueId int64, teamId int64) row {
		var r row
		if err := e.db.QueryRow(`SELECT matches, points, wins, draws, losses, goals_scored, goals_received FROM classification WHERE league_id = $1 AND team_id = $2`,
			leagueId, teamId).Scan(&r.matches, &r.points, &r.wins, &r.draws, &r.losses, &r.scored, &r.conceded); err != nil {
			t.Fatalf("reading standings: %s", err)
		}
		return r
	}
	if got, want := read(e.league, e.home), (row{11, 20, 6, 2, 3, 14, 5}); got != want {
		t.Errorf("drifted home row = %+v, want %+v", got, want)
	}
	if got, want := read(e.league, e.away), (row{11, 11, 3, 2, 6, 5, 14}); got != want {
		t.Errorf("consistent away row = %+v, want it untouched %+v", got, want)
	}
	if got, want := read(e2.league, e2.home), (row{1, 3, 1, 0, 0, 3, 1}); got != want {
		t.Errorf("consistent league's home row = %+v, want it untouched %+v", got, want)
	}

	// A second pass over consistent standings changes nothing.
	repairStandings(e.dbCtx)
	if got, want := read(e.league, e.home), (row{11, 20, 6, 2, 3, 14, 5}); got != want {
		t.Errorf("home row after a second pass = %+v, want %+v", got, want)
	}
}
