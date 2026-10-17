package engine

import (
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// TestAdvanceCup_ConcurrentCallsDontDuplicateRound runs against a real Postgres
// (STRIKER_TEST_DB_DSN; skipped when unset). Reported from production running two --engine
// replicas: the cup page showed one scheduled tie for the next round (correct), but a wider
// upcoming-matches view showed two, a minute apart. advanceCup's "is this round resolved, if so
// create the next one" was a plain check-then-act with no lock at all - two replicas both
// ticking CupJob (once a minute, see StartCupScheduler) could both see the round resolved and
// both call createRoundTies before either's insert was visible to the other. This fires
// advanceCup concurrently from two goroutines against the same cup and asserts exactly one
// round-2 tie landed (not a duplicate).
func TestAdvanceCup_ConcurrentCallsDontDuplicateRound(t *testing.T) {
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

	exec(`INSERT INTO countries (name, current_season) VALUES ('zv', 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`)
	const season = 1
	cupId := scan(`INSERT INTO cups (country, season, name) VALUES ('zv', $1, 'Cup') RETURNING cup_id`, season)

	newTeam := func(name string) int64 {
		return scan(`INSERT INTO teams (name, country) VALUES ($1, 'zv') RETURNING team_id`, name)
	}
	teamA, teamB, teamC, teamD := newTeam("A"), newTeam("B"), newTeam("C"), newTeam("D")

	// Round 1, fully resolved: two played ties, each well outside the live-broadcast window.
	playedAt := time.Now().Add(-3 * time.Hour)
	createTie := func(round, slot int, home, away, winner int64) {
		exec(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date, played, winner_team_id)
			VALUES ($1, $2, $3, $4, $5, $6, true, $7)`, cupId, round, slot, home, away, playedAt, winner)
	}
	createTie(1, 1, teamA, teamB, teamA)
	createTie(1, 2, teamC, teamD, teamC)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := advanceCup(dbCtx, cupId, season); err != nil {
				t.Errorf("advanceCup: %s", err)
			}
		}()
	}
	wg.Wait()

	round2Count := scan(`SELECT count(*) FROM cup_ties WHERE cup_id = $1 AND round = 2`, cupId)
	if round2Count != 1 {
		t.Errorf("concurrent advanceCup calls produced %d round-2 ties, want exactly 1 (no duplicate round)", round2Count)
	}
}
