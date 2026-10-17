package engine

import (
	"database/sql"
	"os"
	"sync"
	"testing"

	"github.com/scruz84/striker/common"
)

// TestGenerateMatchesForLeagues_ConcurrentCallsDontDuplicate runs against a real Postgres
// (STRIKER_TEST_DB_DSN; skipped when unset). GenerateMatches/GenerateMatchesForLeagues' only
// guard used to be a plain, unlocked "does this league already have upcoming matches" read -
// two --engine replicas (a real, supported deployment shape - see docker-compose.yml) racing to
// generate the same league's fixtures could both pass that check and both insert a full
// duplicate schedule. generateMatchesForOneLeague now wraps the check-and-insert in one
// transaction guarded by a non-blocking advisory lock keyed to the league id (the same
// pg_try_advisory_xact_lock pattern common.SetDefaultLineup already uses) - this fires the same
// generation concurrently from two goroutines and asserts exactly one set of fixtures landed.
func TestGenerateMatchesForLeagues_ConcurrentCallsDontDuplicate(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer db.Close()
	dbCtx := &common.DatabaseContext{Db: db}
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

	exec(`INSERT INTO countries (name, current_season) VALUES ('zu', 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`)
	leagueId := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('zu', 1, 'L1', 1) RETURNING league_id`)
	const numTeams = 4
	for i := 0; i < numTeams; i++ {
		exec(`INSERT INTO teams (name, country, league_id) VALUES ($1, 'zu', $2)`, "team", leagueId)
	}
	// A double round-robin among numTeams teams plays (numTeams-1)*2 weeks of numTeams/2
	// matches each - see randomWeekMatches2.
	wantMatches := (numTeams - 1) * 2 * (numTeams / 2)

	leagueIds := []int64{leagueId}
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := GenerateMatchesForLeagues(dbCtx, leagueIds); err != nil {
				t.Errorf("GenerateMatchesForLeagues: %s", err)
			}
		}()
	}
	wg.Wait()

	gotMatches := scan(`SELECT count(*) FROM matches WHERE league_id = $1`, leagueId)
	if gotMatches != int64(wantMatches) {
		t.Errorf("concurrent GenerateMatchesForLeagues calls produced %d matches, want exactly %d (no duplicate schedule)", gotMatches, wantMatches)
	}
}
