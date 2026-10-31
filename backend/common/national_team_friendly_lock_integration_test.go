package common

import (
	"database/sql"
	"os"
	"sync"
	"testing"
)

// TestScheduleNationalTeamFriendlies_ConcurrentCallsDontDoubleBook runs against a real Postgres
// (STRIKER_TEST_DB_DSN; skipped when unset). Reported from production: a country ended up with
// two different national-team friendlies scheduled for the exact same kickoff, with no World Cup
// involved. The real cause: ScheduleNationalTeamFriendlies was a plain check-then-act (read every
// country's pending state, then insert new fixtures for a computed pairing) with no lock at all -
// two --engine replicas both ticking it at once (every 5 minutes) could both read the same
// "nobody's pending" snapshot and both independently pair and insert a fixture for the same
// country before either's write was visible to the other. This fires
// ScheduleNationalTeamFriendlies concurrently from two goroutines for the same two newly-active
// countries and asserts neither ends up with more than one pending friendly - not asserting
// exactly one (rather than zero), since whether they actually get paired this tick depends on the
// rest of whatever countries/World-Cup state already exist in this shared test database, which
// this test doesn't control; what it does control, and what the bug was about, is never ending up
// with two.
func TestScheduleNationalTeamFriendlies_ConcurrentCallsDontDoubleBook(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer db.Close()
	dbCtx := &DatabaseContext{Db: db}
	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	scanCount := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(query, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return n
	}

	// Two made-up, newly-active countries sharing a region (see RegionForCountry's fallback for
	// an unknown code) so pairNationalTeamCountries always pairs them with each other rather than
	// leaving either one the odd one out.
	const countryA, countryB = "zw1", "zw2"
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`, countryA)
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`, countryB)
	// Clear out anything a previous run of this same test left behind, so this run starts clean.
	exec(`DELETE FROM friendly_matches WHERE home_team_id IN (SELECT team_id FROM teams WHERE country = ANY($1) AND is_national_team = true) `+
		`OR away_team_id IN (SELECT team_id FROM teams WHERE country = ANY($1) AND is_national_team = true)`, []string{countryA, countryB})

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ScheduleNationalTeamFriendlies(dbCtx); err != nil {
				t.Errorf("ScheduleNationalTeamFriendlies: %s", err)
			}
		}()
	}
	wg.Wait()

	for _, country := range []string{countryA, countryB} {
		pending := scanCount(`SELECT count(*) FROM friendly_matches fm JOIN teams t ON t.team_id = fm.home_team_id OR t.team_id = fm.away_team_id `+
			`WHERE t.country = $1 AND t.is_national_team = true AND fm.played = false`, country)
		if pending > 1 {
			t.Errorf("country %s has %d pending national-team friendlies after concurrent scheduling, want at most 1 (no double-booking)", country, pending)
		}
		// Nor two kickoffs closer together than NationalTeamFriendlyRest (see
		// TestScheduleNationalTeamFriendlies_RespectsRest).
		tooClose := scanCount(`SELECT count(*) FROM friendly_matches a JOIN friendly_matches b ON a.match_id < b.match_id `+
			`AND abs(extract(epoch FROM a.match_date - b.match_date)) < $2 `+
			`JOIN teams t ON t.is_national_team = true AND t.country = $1 `+
			`WHERE t.team_id IN (a.home_team_id, a.away_team_id) AND t.team_id IN (b.home_team_id, b.away_team_id)`,
			country, NationalTeamFriendlyRest.Seconds())
		if tooClose > 0 {
			t.Errorf("country %s has %d pairs of national-team friendlies less than %v apart", country, tooClose, NationalTeamFriendlyRest)
		}
	}
}
