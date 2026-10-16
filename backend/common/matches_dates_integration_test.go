package common

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

// TestMatchDateHelpers runs against a real Postgres (STRIKER_TEST_DB_DSN, see
// TestEnsureSchema_RestartDoesNotTouchPlayerData; skipped when unset).
func TestMatchDateHelpers(t *testing.T) {
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

	mustScan := func(dest *int64, query string, args ...any) {
		t.Helper()
		if err := db.QueryRow(query, args...).Scan(dest); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	var leagueId, teamA, teamB int64
	if _, err := db.Exec(`INSERT INTO countries (name) VALUES ('zx') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zx', 1, 'L') RETURNING league_id`)
	mustScan(&teamA, `INSERT INTO teams (name, country) VALUES ('A', 'zx') RETURNING team_id`)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('B', 'zx') RETURNING team_id`)

	// A season number far from the other integration tests' data (those use season 1).
	const season = 500
	base := time.Now().Truncate(time.Second)
	insert := func(s int, played bool, date time.Time) {
		t.Helper()
		if _, err := db.Exec(`INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
			VALUES ($1, $2, $3, $4, $5, $6)`, leagueId, s, played, date, teamA, teamB); err != nil {
			t.Fatalf("inserting match: %s", err)
		}
	}
	lastOfSeason := base.Add(10 * time.Minute)
	insert(season, true, lastOfSeason)            // simulated 15 min ahead, still dated in the future
	insert(season, true, base.Add(-24*time.Hour)) // earlier round
	insert(season, false, base.Add(24*time.Hour)) // not started: must not count as "started"

	got, err := GetLatestStartedMatchDate(dbCtx, leagueId, season)
	if err != nil || got == nil || !got.Equal(lastOfSeason) {
		t.Errorf("GetLatestStartedMatchDate: got %v, %v; want %v", got, err, lastOfSeason)
	}
	if none, err := GetLatestStartedMatchDate(dbCtx, leagueId, season+1); err != nil || none != nil {
		t.Errorf("GetLatestStartedMatchDate for a season with no matches: got %v, %v; want nil", none, err)
	}

	before, err := GetLatestMatchDateBeforeSeason(dbCtx, season+1)
	if err != nil || before == nil || before.Before(base.Add(24*time.Hour)) {
		// the unplayed round counts here: this helper looks at every earlier-season match date
		t.Errorf("GetLatestMatchDateBeforeSeason: got %v, %v; want at least %v", before, err, base.Add(24*time.Hour))
	}
}
