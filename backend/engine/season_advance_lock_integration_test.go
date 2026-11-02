package engine

import (
	"database/sql"
	"os"
	"sync"
	"testing"

	"github.com/scruz84/striker/common"
)

// TestAdvanceCountrySeasonIfComplete_ConcurrentCallsDontDuplicateCup runs against a real
// Postgres (STRIKER_TEST_DB_DSN; skipped when unset). Reported from production running two
// --engine replicas: a country's cup page showed one upcoming tie (correct), but a wider
// upcoming-matches view showed two, a minute apart. The real cause: advanceCountrySeasonIfComplete
// was a plain check-then-act with no lock at all, and `cups` has no unique constraint on
// (country, season) - two replicas both ticking AdvanceSeasonJob (every 5 minutes) could both
// see the same country's season complete and both roll it over, each creating its own separate
// cup row (via StartCupForCountry) for the same country and season, each with its own first-round
// tie dated independently. This fires advanceCountrySeasonIfComplete concurrently from two
// goroutines for the same country and asserts exactly one cup landed for the new season (not a
// duplicate), and the season advanced exactly once (not twice).
func TestAdvanceCountrySeasonIfComplete_ConcurrentCallsDontDuplicateCup(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer db.Close()
	// advanceCountrySeasonIfComplete's success path refreshes the current-season cache (see
	// ResetCurrentSeasonCache) - needs a real ObjectCache, not the nil one most other
	// integration tests get away with leaving unset (RedisCache's own writes silently no-op if
	// Redis is unreachable, so this is safe even without REDIS_HOST/PORT set).
	dbCtx := &common.DatabaseContext{Db: db, ObjectCache: common.NewRedisCache()}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}

	// A fresh code per run: the test database persists across runs, and an earlier run's cups for
	// the same country and season would otherwise be counted too.
	country := uniqueCountryCode()
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`, country)
	leagueId := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, country)
	teamA := scan(`INSERT INTO teams (name, country, league_id) VALUES ('A', $1, $2) RETURNING team_id`, country, leagueId)
	teamB := scan(`INSERT INTO teams (name, country, league_id) VALUES ('B', $1, $2) RETURNING team_id`, country, leagueId)
	exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 1, 10)`, leagueId, teamA)
	exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 1, 5)`, leagueId, teamB)

	// The whole season's one match, fully played and well outside the live-broadcast window.
	exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
		VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true)`, leagueId, teamA, teamB)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			advanceCountrySeasonIfComplete(dbCtx, country)
		}()
	}
	wg.Wait()

	season, err := common.GetCurrentSeasonForCountry(dbCtx, country)
	if err != nil || season != 2 {
		t.Errorf("GetCurrentSeasonForCountry(%s): got %d, %v; want 2, nil", country, season, err)
	}

	cupCount := scan(`SELECT count(*) FROM cups WHERE country = $1 AND season = 2`, country)
	if cupCount != 1 {
		t.Errorf("concurrent advanceCountrySeasonIfComplete calls produced %d cups for %s season 2, want exactly 1 (no duplicate rollover)", cupCount, country)
	}
}
