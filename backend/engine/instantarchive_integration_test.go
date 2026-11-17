package engine

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// instantArchiveJobEnv opens the integration database (skipped without STRIKER_TEST_DB_DSN),
// makes sure the season row exists, restores the archive setting afterwards, and returns a helper
// that inserts a played season-1 league match with one instant in a country already in season 5.
func instantArchiveJobEnv(t *testing.T) (*common.DatabaseContext, *sql.DB, func() int64) {
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
	dbCtx := &common.DatabaseContext{Db: db}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	must := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatal(err)
	}
	before, err := common.GetInstantArchiveSettings(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = common.SetInstantArchiveSettings(dbCtx, before) })

	country := fmt.Sprintf("j%03d", time.Now().UnixNano()%1000)
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 5) ON CONFLICT (name) DO UPDATE SET current_season = 5`, country); err != nil {
		t.Fatal(err)
	}
	league := must(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'I', 1) RETURNING league_id`, country)
	home := must(`INSERT INTO teams (name, country, league_id) VALUES ('H', $1, $2) RETURNING team_id`, country, league)
	away := must(`INSERT INTO teams (name, country, league_id) VALUES ('A', $1, $2) RETURNING team_id`, country, league)
	oldMatch := func() int64 {
		t.Helper()
		id := must(`INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id, played)
			VALUES ($1, 1, now() - interval '400 days', $2, $3, true) RETURNING match_id`, league, home, away)
		if _, err := db.Exec(`INSERT INTO match_instants (match_id, minute) VALUES ($1, 1)`, id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	return dbCtx, db, oldMatch
}

func instantsOf(t *testing.T, db *sql.DB, matchId int64) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM match_instants WHERE match_id = $1`, matchId).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestInstantArchiveJob_OncePerSlot covers engine-replicas "Daily instant archive": the first
// execution in a day's slot runs the pass, a second one in the same slot does nothing.
func TestInstantArchiveJob_OncePerSlot(t *testing.T) {
	dbCtx, db, oldMatch := instantArchiveJobEnv(t)
	if err := common.SetInstantArchiveSettings(dbCtx, common.InstantArchiveSettings{Enabled: true, KeepSeasons: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM engine_job_runs WHERE job_name = $1`, instantArchiveJobName); err != nil {
		t.Fatal(err)
	}
	first := oldMatch()
	job := NewInstantArchiveJob(dbCtx)
	if err := job.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if n := instantsOf(t, db, first); n != 0 {
		t.Fatalf("the slot's first run left %d instants on a season-1 match", n)
	}
	second := oldMatch()
	if err := NewInstantArchiveJob(dbCtx).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if n := instantsOf(t, db, second); n != 1 {
		t.Errorf("a second run in the same slot archived again (%d instants left, want 1)", n)
	}
}

// TestInstantArchiveJob_OffDeletesNothing covers "Off keeps everything".
func TestInstantArchiveJob_OffDeletesNothing(t *testing.T) {
	dbCtx, db, oldMatch := instantArchiveJobEnv(t)
	if err := common.SetInstantArchiveSettings(dbCtx, common.InstantArchiveSettings{Enabled: false, KeepSeasons: 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM engine_job_runs WHERE job_name = $1`, instantArchiveJobName); err != nil {
		t.Fatal(err)
	}
	id := oldMatch()
	if err := NewInstantArchiveJob(dbCtx).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if n := instantsOf(t, db, id); n != 1 {
		t.Errorf("the archive is off but the match has %d instants left, want 1", n)
	}
}
