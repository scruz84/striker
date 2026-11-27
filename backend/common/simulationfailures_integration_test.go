package common

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
)

// These tests run against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset) and cover the
// bookkeeping of openspec harden-stuck-match-recovery: the migration, the atomic result saves, the
// attempt counting and backoff, the failure record and the pending post-match effects.

func openSimulationTestDb(t *testing.T) (*sql.DB, *DatabaseContext) {
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
	dbCtx := &DatabaseContext{Db: db}
	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	return db, dbCtx
}

// simulationFixture is a country with one league of two teams, both with a classification row for
// season 1, plus helpers to add fixtures in any state.
type simulationFixture struct {
	t        *testing.T
	db       *sql.DB
	dbCtx    *DatabaseContext
	country  string
	leagueId int64
	home     int64
	away     int64
}

func newSimulationFixture(t *testing.T) *simulationFixture {
	t.Helper()
	db, dbCtx := openSimulationTestDb(t)
	f := &simulationFixture{t: t, db: db, dbCtx: dbCtx, country: unusedTestCountryCode(t, db, 'y')}
	f.exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1)`, f.country)
	f.leagueId = f.scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L', 1) RETURNING league_id`, f.country)
	f.home = f.scan(`INSERT INTO teams (name, country, league_id) VALUES ('Home', $1, $2) RETURNING team_id`, f.country, f.leagueId)
	f.away = f.scan(`INSERT INTO teams (name, country, league_id) VALUES ('Away', $1, $2) RETURNING team_id`, f.country, f.leagueId)
	for _, teamId := range []int64{f.home, f.away} {
		f.exec(`INSERT INTO classification (league_id, team_id, season) VALUES ($1, $2, 1)`, f.leagueId, teamId)
	}
	return f
}

func (f *simulationFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(query, args...); err != nil {
		f.t.Fatalf("%s: %s", query, err)
	}
}

func (f *simulationFixture) scan(query string, args ...any) int64 {
	f.t.Helper()
	var v int64
	if err := f.db.QueryRow(query, args...).Scan(&v); err != nil {
		f.t.Fatalf("%s: %s", query, err)
	}
	return v
}

// leagueMatch inserts an unplayed league match kicking off an hour ago.
func (f *simulationFixture) leagueMatch() int64 {
	f.t.Helper()
	return f.scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, now() - interval '1 hour', $2, $3) RETURNING match_id`, f.leagueId, f.home, f.away)
}

func (f *simulationFixture) friendly() int64 {
	f.t.Helper()
	return f.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date)
		VALUES ($1, $2, now() - interval '1 hour') RETURNING match_id`, f.home, f.away)
}

func (f *simulationFixture) cupTie() int64 {
	f.t.Helper()
	cupId := f.scan(`INSERT INTO cups (country, season, name) VALUES ($1, 1, 'Cup') RETURNING cup_id`, f.country)
	return f.scan(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date)
		VALUES ($1, 1, 1, $2, $3, now() - interval '1 hour') RETURNING tie_id`, cupId, f.home, f.away)
}

type standingsRow struct{ matches, points, wins, draws, losses, scored, conceded int }

func (f *simulationFixture) standings(teamId int64) standingsRow {
	f.t.Helper()
	var r standingsRow
	if err := f.db.QueryRow(`SELECT matches, points, wins, draws, losses, goals_scored, goals_received
		FROM classification WHERE league_id = $1 AND team_id = $2`, f.leagueId, teamId).
		Scan(&r.matches, &r.points, &r.wins, &r.draws, &r.losses, &r.scored, &r.conceded); err != nil {
		f.t.Fatalf("reading standings: %s", err)
	}
	return r
}

func (f *simulationFixture) count(query string, args ...any) int {
	f.t.Helper()
	return int(f.scan(query, args...))
}

func goalEvent(team int64, other int64, minute int, homeGoals int) Event {
	return Event{EventType: GOAL, Minute: minute, TeamOwner: team, OtherTeam: other, TeamOwnerPlayers: []int64{1},
		TeamOwnerPlayersNames: []string{"Scorer"}, HomeGoals: homeGoals}
}

// TestSimulationFailureColumns_MigrationSeedsStuckRowsOnce runs init86 (archived under
// resources/sql/history/ since init.sql carries its columns) against a scratch database that has
// the schema without its columns: rows stuck for over an hour end with 2 attempts, a
// fresh claim and a played row keep 0, and a second run changes nothing - not even a later stuck
// row, because the seeding is guarded by the columns' absence.
func TestSimulationFailureColumns_MigrationSeedsStuckRowsOnce(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer admin.Close()
	name := fmt.Sprintf("striker_migration_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Skipf("cannot create a scratch database: %s", err)
	}
	defer admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")

	scratchDsn := dsn
	if u, err := url.Parse(dsn); err == nil && strings.HasPrefix(dsn, "postgres") {
		u.Path = "/" + name
		scratchDsn = u.String()
	}
	cfg, err := pgx.ParseConfig(scratchDsn)
	if err != nil {
		t.Fatalf("parsing the scratch DSN: %s", err)
	}
	cfg.Database = name
	db := stdlib.OpenDB(*cfg)
	defer db.Close()
	dbCtx := &DatabaseContext{Db: db}
	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	// Take the schema back to what a database had before this change.
	for _, table := range []string{"matches", "friendly_matches"} {
		for _, column := range []string{"simulation_attempts", "last_simulation_error", "last_simulation_attempt_at"} {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, column)); err != nil {
				t.Fatalf("dropping %s.%s: %s", table, column, err)
			}
		}
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}
	exec(`INSERT INTO countries (name, current_season) VALUES ('zzmg', 1)`)
	leagueId := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('zzmg', 1, 'L', 1) RETURNING league_id`)
	home := scan(`INSERT INTO teams (name, country, league_id) VALUES ('H', 'zzmg', $1) RETURNING team_id`, leagueId)
	away := scan(`INSERT INTO teams (name, country, league_id) VALUES ('A', 'zzmg', $1) RETURNING team_id`, leagueId)
	insertMatch := func(played bool, inSimulation bool, since string) int64 {
		return scan(`INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id, played, in_simulation, in_simulation_since)
			VALUES ($1, 1, now() - interval '3 hours', $2, $3, $4, $5, `+since+`) RETURNING match_id`, leagueId, home, away, played, inSimulation)
	}
	stuckTwoHours := insertMatch(false, true, "now() - interval '2 hours'")
	stuckNeverStamped := insertMatch(false, true, "NULL")
	freshClaim := insertMatch(false, true, "now() - interval '5 minutes'")
	playedRow := insertMatch(true, false, "NULL")
	stuckFriendly := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, in_simulation, in_simulation_since)
		VALUES ($1, $2, now() - interval '3 hours', true, now() - interval '2 hours') RETURNING match_id`, home, away)

	attempts := func(id int64) int64 { return scan(`SELECT simulation_attempts FROM matches WHERE match_id = $1`, id) }
	applyHistoryMigration(t, db, "init86_simulation_failures.sql")
	for id, want := range map[int64]int64{stuckTwoHours: 2, stuckNeverStamped: 2, freshClaim: 0, playedRow: 0} {
		if got := attempts(id); got != want {
			t.Errorf("match %d has %d attempts after the migration, want %d", id, got, want)
		}
	}
	if got := scan(`SELECT simulation_attempts FROM friendly_matches WHERE match_id = $1`, stuckFriendly); got != 2 {
		t.Errorf("stuck friendly has %d attempts after the migration, want 2", got)
	}

	// Run again: a row that became stuck since must not be seeded, and nothing else moves.
	exec(`UPDATE matches SET in_simulation_since = now() - interval '5 hours' WHERE match_id = $1`, freshClaim)
	applyHistoryMigration(t, db, "init86_simulation_failures.sql")
	for id, want := range map[int64]int64{stuckTwoHours: 2, stuckNeverStamped: 2, freshClaim: 0, playedRow: 0} {
		if got := attempts(id); got != want {
			t.Errorf("match %d has %d attempts after the second run, want %d (the migration must be re-runnable)", id, got, want)
		}
	}
}

func TestSaveLeagueMatchResultTx_SecondSaveIsRejectedAndChangesNothing(t *testing.T) {
	f := newSimulationFixture(t)
	matchId := f.leagueMatch()
	result := MatchResult{HomeGoals: 2, AwayGoals: 1, Events: []Event{goalEvent(f.home, f.away, 10, 1)}, Instants: []MatchInstant{{Minute: 1}}}

	if err := SaveLeagueMatchResultTx(f.dbCtx, matchId, f.leagueId, f.home, f.away, result); err != nil {
		t.Fatalf("first save: %s", err)
	}
	other := MatchResult{HomeGoals: 0, AwayGoals: 5, Events: []Event{goalEvent(f.away, f.home, 20, 0)}}
	if err := SaveLeagueMatchResultTx(f.dbCtx, matchId, f.leagueId, f.home, f.away, other); err != ErrAlreadyPlayed {
		t.Fatalf("second save returned %v, want ErrAlreadyPlayed", err)
	}

	if home := f.standings(f.home); home != (standingsRow{matches: 1, points: 3, wins: 1, scored: 2, conceded: 1}) {
		t.Errorf("home standings after two saves: %+v", home)
	}
	if away := f.standings(f.away); away != (standingsRow{matches: 1, losses: 1, scored: 1, conceded: 2}) {
		t.Errorf("away standings after two saves: %+v", away)
	}
	var homeGoals, awayGoals int
	f.db.QueryRow(`SELECT home_goals, away_goals FROM matches WHERE match_id = $1`, matchId).Scan(&homeGoals, &awayGoals)
	if homeGoals != 2 || awayGoals != 1 {
		t.Errorf("stored result %d-%d, want the first save's 2-1", homeGoals, awayGoals)
	}
	if events := f.count(`SELECT count(*) FROM match_events WHERE match_id = $1`, matchId); events != 1 {
		t.Errorf("%d events stored, want the first save's 1", events)
	}
}

func TestSaveLeagueMatchResultTx_FailingInsertStoresNothing(t *testing.T) {
	f := newSimulationFixture(t)
	matchId := f.leagueMatch()
	// An event owned by a team that doesn't exist breaks the foreign key after the result row and
	// the first event have already been written.
	result := MatchResult{HomeGoals: 1, AwayGoals: 0, Events: []Event{goalEvent(f.home, f.away, 10, 1), goalEvent(-1, f.away, 20, 1)},
		Effects: &PostMatchPayload{Home: f.home, Away: f.away}}

	if err := SaveLeagueMatchResultTx(f.dbCtx, matchId, f.leagueId, f.home, f.away, result); err == nil || err == ErrAlreadyPlayed {
		t.Fatalf("save with a broken event returned %v, want an insert error", err)
	}

	if played := f.count(`SELECT count(*) FROM matches WHERE match_id = $1 AND played`, matchId); played != 0 {
		t.Errorf("the match is marked played although its save failed")
	}
	if events := f.count(`SELECT count(*) FROM match_events WHERE match_id = $1`, matchId); events != 0 {
		t.Errorf("%d events stored by a failed save", events)
	}
	if pending := f.count(`SELECT count(*) FROM pending_post_match_effects WHERE kind = 'league' AND match_id = $1`, matchId); pending != 0 {
		t.Errorf("a failed save left a pending post-match record")
	}
	if home := f.standings(f.home); home != (standingsRow{}) {
		t.Errorf("a failed save changed the standings: %+v", home)
	}
	// And the match can still be saved afterwards.
	if err := SaveLeagueMatchResultTx(f.dbCtx, matchId, f.leagueId, f.home, f.away, MatchResult{HomeGoals: 1}); err != nil {
		t.Errorf("save after a failed one: %s", err)
	}
}

func TestSaveLeagueMatchResultTx_ReplacesLeftoverEventsAndInstants(t *testing.T) {
	f := newSimulationFixture(t)
	matchId := f.leagueMatch()
	// What an earlier failed attempt of the previous build left behind.
	f.exec(`INSERT INTO match_events (match_id, event_type, team_owner, other_team, minute) VALUES ($1, $2, $3, $4, 5), ($1, $2, $3, $4, 6)`,
		matchId, GOAL, f.home, f.away)
	f.exec(`INSERT INTO match_instants (match_id, minute) VALUES ($1, 1), ($1, 2), ($1, 3)`, matchId)

	result := MatchResult{HomeGoals: 1, Events: []Event{goalEvent(f.home, f.away, 30, 1)}, Instants: []MatchInstant{{Minute: 1}, {Minute: 2}}}
	if err := SaveLeagueMatchResultTx(f.dbCtx, matchId, f.leagueId, f.home, f.away, result); err != nil {
		t.Fatalf("save: %s", err)
	}

	if events := f.count(`SELECT count(*) FROM match_events WHERE match_id = $1`, matchId); events != 1 {
		t.Errorf("%d events after the save, want exactly the new set of 1", events)
	}
	if minute := f.count(`SELECT minute FROM match_events WHERE match_id = $1`, matchId); minute != 30 {
		t.Errorf("the stored event is minute %d, want the new one at 30", minute)
	}
	if instants := f.count(`SELECT count(*) FROM match_instants WHERE match_id = $1`, matchId); instants != 2 {
		t.Errorf("%d instants after the save, want exactly the new set of 2", instants)
	}
}

func TestSaveLeagueMatchResultTx_ConcurrentSavesBookStandingsOnce(t *testing.T) {
	f := newSimulationFixture(t)
	matchId := f.leagueMatch()

	var wg sync.WaitGroup
	results := make([]error, 4)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = SaveLeagueMatchResultTx(f.dbCtx, matchId, f.leagueId, f.home, f.away, MatchResult{HomeGoals: i + 1})
		}(i)
	}
	wg.Wait()

	saved := 0
	for _, err := range results {
		switch err {
		case nil:
			saved++
		case ErrAlreadyPlayed:
		default:
			t.Errorf("unexpected save error: %s", err)
		}
	}
	if saved != 1 {
		t.Errorf("%d concurrent saves succeeded, want exactly 1", saved)
	}
	if home := f.standings(f.home); home.matches != 1 {
		t.Errorf("home played %d matches after concurrent saves, want 1", home.matches)
	}
}

func TestSaveFriendlyAndCupResultTx_OnlyApplyWhileUnplayed(t *testing.T) {
	f := newSimulationFixture(t)

	friendlyId := f.friendly()
	result := MatchResult{HomeGoals: 1, Events: []Event{goalEvent(f.home, f.away, 10, 1)}, Instants: []MatchInstant{{Minute: 1}}}
	if err := SaveFriendlyResultTx(f.dbCtx, friendlyId, f.home, f.away, result); err != nil {
		t.Fatalf("first friendly save: %s", err)
	}
	if err := SaveFriendlyResultTx(f.dbCtx, friendlyId, f.home, f.away, MatchResult{AwayGoals: 3}); err != ErrAlreadyPlayed {
		t.Errorf("second friendly save returned %v, want ErrAlreadyPlayed", err)
	}
	if goals := f.count(`SELECT home_goals FROM friendly_matches WHERE match_id = $1`, friendlyId); goals != 1 {
		t.Errorf("friendly home goals %d, want the first save's 1", goals)
	}

	tieId := f.cupTie()
	f.exec(`INSERT INTO cup_tie_events (tie_id, event_type, team_owner, other_team, minute) VALUES ($1, $2, $3, $4, 5)`, tieId, GOAL, f.home, f.away)
	if err := SaveCupTieResultTx(f.dbCtx, tieId, false, f.home, MatchResult{HomeGoals: 1, Events: []Event{goalEvent(f.home, f.away, 10, 1)}}); err != nil {
		t.Fatalf("first cup save: %s", err)
	}
	if events := f.count(`SELECT count(*) FROM cup_tie_events WHERE tie_id = $1`, tieId); events != 1 {
		t.Errorf("%d cup events after the save, want the new set of 1", events)
	}
	if err := SaveCupTieResultTx(f.dbCtx, tieId, true, f.away, MatchResult{}); err != ErrAlreadyPlayed {
		t.Errorf("second cup save returned %v, want ErrAlreadyPlayed", err)
	}
	if winner := f.scan(`SELECT winner_team_id FROM cup_ties WHERE tie_id = $1`, tieId); winner != f.home {
		t.Errorf("cup winner %d, want the first save's %d", winner, f.home)
	}
}

func TestSaveFriendlyResultTx_UpdatesWorldCupGroupInTheSameTransaction(t *testing.T) {
	f := newSimulationFixture(t)
	friendlyId := f.friendly()
	worldCupId := f.scan(`INSERT INTO world_cups (season, name) VALUES (1, 'World Cup') RETURNING world_cup_id`)
	groupId := f.scan(`INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, 'A') RETURNING group_id`, worldCupId)
	f.exec(`INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, 1, $2)`, groupId, friendlyId)
	for _, teamId := range []int64{f.home, f.away} {
		f.exec(`INSERT INTO world_cup_group_teams (group_id, team_id) VALUES ($1, $2)`, groupId, teamId)
	}

	if err := SaveFriendlyResultTx(f.dbCtx, friendlyId, f.home, f.away, MatchResult{}); err != nil {
		t.Fatalf("save: %s", err)
	}
	for _, teamId := range []int64{f.home, f.away} {
		if drawn := f.count(`SELECT drawn FROM world_cup_group_teams WHERE group_id = $1 AND team_id = $2`, groupId, teamId); drawn != 1 {
			t.Errorf("team %d has %d draws in the group, want 1", teamId, drawn)
		}
	}
}

// A fixture failed 5 minutes ago is not selected; one failed 11 minutes ago is, with its attempts
// incremented. Covers the three selectors and the stuck re-claim.
func TestSimulationSelectors_HonorBackoffAndCountAttempts(t *testing.T) {
	f := newSimulationFixture(t)

	recent := f.leagueMatch()
	f.exec(`UPDATE matches SET simulation_attempts = 1, last_simulation_attempt_at = now() - interval '5 minutes' WHERE match_id = $1`, recent)
	due := f.leagueMatch()
	f.exec(`UPDATE matches SET simulation_attempts = 1, last_simulation_attempt_at = now() - interval '11 minutes' WHERE match_id = $1`, due)
	exhausted := f.leagueMatch()
	f.exec(`UPDATE matches SET simulation_attempts = 3, last_simulation_attempt_at = now() - interval '1 hour' WHERE match_id = $1`, exhausted)

	// Scope the assertions to this fixture's league: other tests' rows may be selectable too.
	selected := func(matches *[]WeekMatch) map[int64]bool {
		ids := map[int64]bool{}
		for _, m := range *matches {
			if m.DivisionId == f.leagueId {
				ids[m.MatchId] = true
			}
		}
		return ids
	}
	matches, err := GetSimulableMatches(f.dbCtx)
	if err != nil {
		t.Fatalf("GetSimulableMatches: %s", err)
	}
	ids := selected(matches)
	if ids[recent] || ids[exhausted] || !ids[due] {
		t.Errorf("selected %v: want only the fixture past its backoff (%d), not %d (5 min ago) or %d (3 attempts)", ids, due, recent, exhausted)
	}
	if attempts := f.count(`SELECT simulation_attempts FROM matches WHERE match_id = $1`, due); attempts != 2 {
		t.Errorf("the claimed match has %d attempts, want 2", attempts)
	}
	if inSimulation := f.count(`SELECT count(*) FROM matches WHERE match_id = $1 AND in_simulation`, due); inSimulation != 1 {
		t.Errorf("the selected match was not claimed")
	}
	if attempts := f.count(`SELECT simulation_attempts FROM matches WHERE match_id = $1`, recent); attempts != 1 {
		t.Errorf("an unselected match had its attempts changed to %d", attempts)
	}

	// Friendlies.
	recentFriendly, dueFriendly := f.friendly(), f.friendly()
	f.exec(`UPDATE friendly_matches SET simulation_attempts = 1, last_simulation_attempt_at = now() - interval '5 minutes' WHERE match_id = $1`, recentFriendly)
	f.exec(`UPDATE friendly_matches SET simulation_attempts = 1, last_simulation_attempt_at = now() - interval '11 minutes' WHERE match_id = $1`, dueFriendly)
	friendlies, err := GetSimulableFriendlies(f.dbCtx)
	if err != nil {
		t.Fatalf("GetSimulableFriendlies: %s", err)
	}
	friendlyIds := map[int64]bool{}
	for _, m := range friendlies {
		friendlyIds[m.MatchId] = true
	}
	if friendlyIds[recentFriendly] || !friendlyIds[dueFriendly] {
		t.Errorf("selected friendlies %v: want only %d", friendlyIds, dueFriendly)
	}
	if attempts := f.count(`SELECT simulation_attempts FROM friendly_matches WHERE match_id = $1`, dueFriendly); attempts != 2 {
		t.Errorf("the claimed friendly has %d attempts, want 2", attempts)
	}

	// Cup ties: the selector filters, and the claim under the tie's lock counts the attempt.
	tieId := f.cupTie()
	f.exec(`UPDATE cup_ties SET simulation_attempts = 1, last_simulation_attempt_at = now() - interval '5 minutes' WHERE tie_id = $1`, tieId)
	ties, err := GetSimulableCupTies(f.dbCtx)
	if err != nil {
		t.Fatalf("GetSimulableCupTies: %s", err)
	}
	for _, tie := range ties {
		if tie.TieId == tieId {
			t.Errorf("a cup tie failed 5 minutes ago was selected")
		}
	}
	if claimed, err := ClaimCupTieAttempt(f.dbCtx, tieId); err != nil || claimed {
		t.Errorf("ClaimCupTieAttempt inside the backoff = %v, %v; want false", claimed, err)
	}
	f.exec(`UPDATE cup_ties SET last_simulation_attempt_at = now() - interval '11 minutes' WHERE tie_id = $1`, tieId)
	if claimed, err := ClaimCupTieAttempt(f.dbCtx, tieId); err != nil || !claimed {
		t.Errorf("ClaimCupTieAttempt past the backoff = %v, %v; want true", claimed, err)
	}
	if attempts := f.count(`SELECT simulation_attempts FROM cup_ties WHERE tie_id = $1`, tieId); attempts != 2 {
		t.Errorf("the claimed tie has %d attempts, want 2", attempts)
	}
	f.exec(`UPDATE cup_ties SET simulation_attempts = 3, last_simulation_attempt_at = now() - interval '1 hour' WHERE tie_id = $1`, tieId)
	if claimed, err := ClaimCupTieAttempt(f.dbCtx, tieId); err != nil || claimed {
		t.Errorf("ClaimCupTieAttempt with 3 attempts used = %v, %v; want false", claimed, err)
	}
}

func TestGetStuckMatches_CountsTheAttemptAndSkipsExhaustedRows(t *testing.T) {
	f := newSimulationFixture(t)
	stuck := f.leagueMatch()
	f.exec(`UPDATE matches SET in_simulation = true, in_simulation_since = now() - interval '30 minutes', simulation_attempts = 1 WHERE match_id = $1`, stuck)
	exhausted := f.leagueMatch()
	f.exec(`UPDATE matches SET in_simulation = true, in_simulation_since = now() - interval '30 minutes', simulation_attempts = 3 WHERE match_id = $1`, exhausted)

	matches, err := GetStuckMatches(f.dbCtx)
	if err != nil {
		t.Fatalf("GetStuckMatches: %s", err)
	}
	ids := map[int64]bool{}
	for _, m := range *matches {
		ids[m.MatchId] = true
	}
	if !ids[stuck] || ids[exhausted] {
		t.Errorf("re-claimed %v: want %d only (the other has used its 3 attempts)", ids, stuck)
	}
	if attempts := f.count(`SELECT simulation_attempts FROM matches WHERE match_id = $1`, stuck); attempts != 2 {
		t.Errorf("the re-claimed match has %d attempts, want 2", attempts)
	}
}

func TestRecordSimulationFailure_ReleasesClaimAndStoresTruncatedError(t *testing.T) {
	f := newSimulationFixture(t)
	matchId := f.leagueMatch()
	f.exec(`UPDATE matches SET in_simulation = true, simulation_attempts = 2 WHERE match_id = $1`, matchId)

	long := strings.Repeat("é", 800)
	if err := RecordSimulationFailure(f.dbCtx, SimulationKindLeague, matchId, fmt.Errorf("%s", long)); err != nil {
		t.Fatalf("RecordSimulationFailure: %s", err)
	}
	var inSimulation bool
	var attempts int
	var stored string
	f.db.QueryRow(`SELECT in_simulation, simulation_attempts, last_simulation_error FROM matches WHERE match_id = $1`, matchId).Scan(&inSimulation, &attempts, &stored)
	if inSimulation {
		t.Errorf("the claim was not released")
	}
	if attempts != 2 {
		t.Errorf("attempts changed to %d by an ordinary failure, want to stay 2", attempts)
	}
	if got := len([]rune(stored)); got != 500 {
		t.Errorf("stored error has %d characters, want it truncated to 500", got)
	}

	// A connection error takes the attempt back.
	f.exec(`UPDATE matches SET in_simulation = true WHERE match_id = $1`, matchId)
	if err := RecordSimulationFailure(f.dbCtx, SimulationKindLeague, matchId, fmt.Errorf("saving: %w", &pgconn.PgError{Code: "08006"})); err != nil {
		t.Fatalf("RecordSimulationFailure: %s", err)
	}
	if attempts := f.count(`SELECT simulation_attempts FROM matches WHERE match_id = $1`, matchId); attempts != 1 {
		t.Errorf("attempts after a connection error: %d, want 1 (the attempt is not counted)", attempts)
	}

	// The same for the other two kinds, and a played fixture is never touched.
	friendlyId, tieId := f.friendly(), f.cupTie()
	f.exec(`UPDATE friendly_matches SET in_simulation = true, simulation_attempts = 1 WHERE match_id = $1`, friendlyId)
	f.exec(`UPDATE cup_ties SET simulation_attempts = 1 WHERE tie_id = $1`, tieId)
	if err := RecordSimulationFailure(f.dbCtx, SimulationKindFriendly, friendlyId, fmt.Errorf("friendly failed")); err != nil {
		t.Fatalf("RecordSimulationFailure friendly: %s", err)
	}
	if err := RecordSimulationFailure(f.dbCtx, SimulationKindCup, tieId, fmt.Errorf("tie failed")); err != nil {
		t.Fatalf("RecordSimulationFailure cup: %s", err)
	}
	var friendlyError, tieError string
	f.db.QueryRow(`SELECT last_simulation_error FROM friendly_matches WHERE match_id = $1`, friendlyId).Scan(&friendlyError)
	f.db.QueryRow(`SELECT last_simulation_error FROM cup_ties WHERE tie_id = $1`, tieId).Scan(&tieError)
	if friendlyError != "friendly failed" || tieError != "tie failed" {
		t.Errorf("stored errors %q / %q", friendlyError, tieError)
	}
	played := f.leagueMatch()
	f.exec(`UPDATE matches SET played = true WHERE match_id = $1`, played)
	_ = RecordSimulationFailure(f.dbCtx, SimulationKindLeague, played, fmt.Errorf("late failure"))
	if stored := f.count(`SELECT count(*) FROM matches WHERE match_id = $1 AND last_simulation_error IS NOT NULL`, played); stored != 0 {
		t.Errorf("a failure was recorded on a fixture that is already played")
	}
}

func TestPostMatchEffects_ClaimedOnceAndDrainedWhenDue(t *testing.T) {
	f := newSimulationFixture(t)
	payload := &PostMatchPayload{Home: f.home, Away: f.away, HomeGoals: 2, AwayGoals: 1,
		PendingUnavailabilities: []PendingUnavailability{{PlayerId: 7, Reason: UNAVAILABLE_INJURY, Matches: 3}},
		HomeMinutesByPosition:   map[int64]map[int]int{7: {DEFENSE: 90}}}

	// A save stores its pending record, due when the live-broadcast window has passed.
	matchId := f.leagueMatch()
	if err := SaveLeagueMatchResultTx(f.dbCtx, matchId, f.leagueId, f.home, f.away, MatchResult{HomeGoals: 2, AwayGoals: 1, Effects: payload}); err != nil {
		t.Fatalf("save: %s", err)
	}
	var dueAt, matchDate time.Time
	f.db.QueryRow(`SELECT p.due_at, m.match_date FROM pending_post_match_effects p JOIN matches m ON m.match_id = p.match_id
		WHERE p.kind = 'league' AND p.match_id = $1`, matchId).Scan(&dueAt, &matchDate)
	if diff := dueAt.Sub(matchDate.Add(LiveBroadcastWindow)); diff < -time.Second || diff > time.Second {
		t.Errorf("due_at is %s, want kickoff + %s (%s)", dueAt, LiveBroadcastWindow, matchDate.Add(LiveBroadcastWindow))
	}

	// The completion claims it once; a second claim finds nothing.
	got, err := ClaimPostMatchEffects(f.dbCtx, SimulationKindLeague, matchId)
	if err != nil || got == nil {
		t.Fatalf("first claim: %v, %v", got, err)
	}
	if got.HomeGoals != 2 || len(got.PendingUnavailabilities) != 1 || got.PendingUnavailabilities[0].Matches != 3 || got.HomeMinutesByPosition[7][DEFENSE] != 90 {
		t.Errorf("claimed payload does not round-trip: %+v", got)
	}
	if again, err := ClaimPostMatchEffects(f.dbCtx, SimulationKindLeague, matchId); err != nil || again != nil {
		t.Errorf("second claim: %v, %v; want nothing", again, err)
	}

	// Two drains race for a record due a minute ago; a record due in the future is left alone.
	dueId, futureId := f.leagueMatch(), f.leagueMatch()
	insert := func(id int64, dueExpr string) {
		f.exec(`INSERT INTO pending_post_match_effects (kind, match_id, due_at, payload) VALUES ('league', $1, `+dueExpr+`, $2)`, id, `{"home":1,"away":2}`)
	}
	insert(dueId, "now() - interval '1 minute'")
	insert(futureId, "now() + interval '1 hour'")
	var wg sync.WaitGroup
	var mu sync.Mutex
	claimedBy := 0
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			due, err := ClaimDuePostMatchEffects(f.dbCtx, 200)
			if err != nil {
				t.Errorf("ClaimDuePostMatchEffects: %s", err)
			}
			for _, d := range due {
				if d.MatchId == dueId {
					mu.Lock()
					claimedBy++
					mu.Unlock()
				}
				if d.MatchId == futureId {
					t.Errorf("claimed a record that is not due yet")
				}
			}
		}()
	}
	wg.Wait()
	if claimedBy != 1 {
		t.Errorf("the due record was claimed %d times, want once", claimedBy)
	}
	if left := f.count(`SELECT count(*) FROM pending_post_match_effects WHERE match_id = $1`, futureId); left != 1 {
		t.Errorf("the future record was removed")
	}
}

func TestGetSimulationHealth_ListsFailingFixturesAndCountsStuckOnes(t *testing.T) {
	f := newSimulationFixture(t)
	failing := f.leagueMatch()
	f.exec(`UPDATE matches SET simulation_attempts = 2, last_simulation_error = 'boom' WHERE match_id = $1`, failing)
	liveClaim := f.leagueMatch()
	f.exec(`UPDATE matches SET in_simulation = true, in_simulation_since = now() WHERE match_id = $1`, liveClaim)
	stuck := f.leagueMatch()
	f.exec(`UPDATE matches SET in_simulation = true, in_simulation_since = now() - interval '30 minutes' WHERE match_id = $1`, stuck)
	failingTie := f.cupTie()
	f.exec(`UPDATE cup_ties SET simulation_attempts = 1, last_simulation_error = 'tie boom' WHERE tie_id = $1`, failingTie)

	before, err := GetSimulationHealth(f.dbCtx)
	if err != nil {
		t.Fatalf("GetSimulationHealth: %s", err)
	}
	found := map[string]FailingFixture{}
	for _, fixture := range before.Failing {
		found[fixture.Kind+fmt.Sprint(fixture.Id)] = fixture
	}
	if got := found["league"+fmt.Sprint(failing)]; got.LastError != "boom" || got.Attempts != 2 || got.HomeName != "Home" || got.AwayName != "Away" {
		t.Errorf("failing match listed as %+v", got)
	}
	if got := found["cup"+fmt.Sprint(failingTie)]; got.LastError != "tie boom" {
		t.Errorf("failing tie listed as %+v", got)
	}
	if _, listed := found["league"+fmt.Sprint(stuck)]; listed {
		t.Errorf("a stuck row with no recorded failure is listed as failing")
	}
	if len(before.Failing) > MaxFailingFixturesListed {
		t.Errorf("%d failing fixtures listed, want at most %d", len(before.Failing), MaxFailingFixturesListed)
	}
	if before.FailingCounts.LeagueMatches < 1 || before.FailingCounts.CupTies < 1 {
		t.Errorf("failing counts %+v miss the seeded fixtures", before.FailingCounts)
	}

	// The live claim is not stuck, the old one is: ageing the live claim adds exactly one.
	f.exec(`UPDATE matches SET in_simulation_since = now() - interval '30 minutes' WHERE match_id = $1`, liveClaim)
	after, err := GetSimulationHealth(f.dbCtx)
	if err != nil {
		t.Fatalf("GetSimulationHealth: %s", err)
	}
	if after.Stuck != before.Stuck+1 {
		t.Errorf("stuck count went %d -> %d after ageing one claim, want +1", before.Stuck, after.Stuck)
	}
}
