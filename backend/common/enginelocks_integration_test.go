package common

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These run against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset). Each fires the same
// call from several goroutines at once - what several --engine replicas ticking the same cron
// trigger do - and asserts the work is done once. See openspec/specs/engine-replicas.

func openEngineLocksTestDb(t *testing.T) (*sql.DB, *DatabaseContext) {
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

func countConcurrentWins(n int, try func() bool) int64 {
	var wins atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if try() {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	return wins.Load()
}

// TestClaimEngineJobSlot_OneReplicaPerSlot: a weekly or daily job fires on every replica at the
// same instant; only one may run it per slot, and the next slot must be claimable again.
func TestClaimEngineJobSlot_OneReplicaPerSlot(t *testing.T) {
	_, dbCtx := openEngineLocksTestDb(t)
	job := fmt.Sprintf("test_job_%d", time.Now().UnixNano())
	fire := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)

	if wins := countConcurrentWins(4, func() bool { return ClaimEngineJobSlot(dbCtx, job, fire, 24*time.Hour) }); wins != 1 {
		t.Fatalf("first slot: %d replicas claimed it, want 1", wins)
	}
	// A replica whose trigger fires a few milliseconds late lands in the same slot.
	if ClaimEngineJobSlot(dbCtx, job, fire.Add(150*time.Millisecond), 24*time.Hour) {
		t.Errorf("a late fire of the same slot was claimed again")
	}
	next := fire.Add(7 * 24 * time.Hour)
	if wins := countConcurrentWins(4, func() bool { return ClaimEngineJobSlot(dbCtx, job, next, 24*time.Hour) }); wins != 1 {
		t.Errorf("next slot: %d replicas claimed it, want 1", wins)
	}
}

// TestGetStuckMatches_SkipsLiveClaimsAndReclaimsOnce: the recovery job used to re-simulate any
// claimed, unplayed match - including one a live pass had claimed the same second - and every
// replica running it got the same list.
func TestGetStuckMatches_SkipsLiveClaimsAndReclaimsOnce(t *testing.T) {
	db, dbCtx := openEngineLocksTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}
	// Leave no other test's stuck rows behind to blur the counts below.
	if _, err := db.Exec(`UPDATE matches SET in_simulation = false WHERE played = false AND in_simulation = true`); err != nil {
		t.Fatalf("clearing earlier claims: %s", err)
	}
	if _, err := db.Exec(`UPDATE friendly_matches SET in_simulation = false WHERE played = false AND in_simulation = true`); err != nil {
		t.Fatalf("clearing earlier claims: %s", err)
	}

	country := fmt.Sprintf("y%03d", time.Now().UnixNano()%1000)
	if _, err := db.Exec(`INSERT INTO countries (name) VALUES ($1) ON CONFLICT DO NOTHING`, country); err != nil {
		t.Fatalf("inserting country: %s", err)
	}
	leagueId := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L', 1) RETURNING league_id`, country)
	home := scan(`INSERT INTO teams (name, country, league_id) VALUES ('H', $1, $2) RETURNING team_id`, country, leagueId)
	away := scan(`INSERT INTO teams (name, country, league_id) VALUES ('A', $1, $2) RETURNING team_id`, country, leagueId)

	live := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, in_simulation, in_simulation_since)
		VALUES ($1, 1, 1, now(), $2, $3, true, now()) RETURNING match_id`, leagueId, home, away)
	abandoned := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, in_simulation, in_simulation_since)
		VALUES ($1, 1, 2, now(), $2, $3, true, now() - interval '1 hour') RETURNING match_id`, leagueId, home, away)
	legacy := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, in_simulation)
		VALUES ($1, 1, 3, now(), $2, $3, true) RETURNING match_id`, leagueId, home, away)
	liveFriendly := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, in_simulation, in_simulation_since)
		VALUES ($1, $2, now(), true, now()) RETURNING match_id`, home, away)
	abandonedFriendly := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, in_simulation, in_simulation_since)
		VALUES ($1, $2, now(), true, now() - interval '1 hour') RETURNING match_id`, home, away)

	var mu sync.Mutex
	gotMatches := map[int64]int{}
	gotFriendlies := map[int64]int{}
	countConcurrentWins(3, func() bool {
		matches, err := GetStuckMatches(dbCtx)
		if err != nil {
			t.Errorf("GetStuckMatches: %s", err)
			return false
		}
		friendlies, err := GetStuckFriendlies(dbCtx)
		if err != nil {
			t.Errorf("GetStuckFriendlies: %s", err)
			return false
		}
		mu.Lock()
		defer mu.Unlock()
		for _, m := range *matches {
			gotMatches[m.MatchId]++
		}
		for _, f := range friendlies {
			gotFriendlies[f.MatchId]++
		}
		return true
	})

	if gotMatches[live] != 0 || gotFriendlies[liveFriendly] != 0 {
		t.Errorf("a match claimed just now by a live pass was handed to the recovery: matches %v, friendlies %v", gotMatches, gotFriendlies)
	}
	for _, id := range []int64{abandoned, legacy} {
		if gotMatches[id] != 1 {
			t.Errorf("stuck match %d recovered %d times, want 1", id, gotMatches[id])
		}
	}
	if gotFriendlies[abandonedFriendly] != 1 {
		t.Errorf("stuck friendly %d recovered %d times, want 1", abandonedFriendly, gotFriendlies[abandonedFriendly])
	}
}
