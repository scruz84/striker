package common

import (
	"database/sql"
	"os"
	"testing"
)

// TestSeasonIsCountryIsolated runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when
// unset). current_season now lives on `countries` (one counter per country, see
// init75_season_per_country.sql) rather than the old global `season` singleton, so two
// countries at different season numbers must never see each other's matches when checking
// season completion (see engine.advanceCountrySeasonIfComplete) - otherwise a fast country
// could be told it still has "unplayed matches" that actually belong to a slower one still on
// an earlier season number, or vice versa.
func TestSeasonIsCountryIsolated(t *testing.T) {
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
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}

	// Two made-up countries, deliberately at different seasons.
	exec(`INSERT INTO countries (name, current_season) VALUES ('zs', 10) ON CONFLICT (name) DO UPDATE SET current_season = 10`)
	exec(`INSERT INTO countries (name, current_season) VALUES ('zt', 3) ON CONFLICT (name) DO UPDATE SET current_season = 3`)

	leagueZs := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('zs', 1, 'L1', 1) RETURNING league_id`)
	leagueZt := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('zt', 1, 'L1', 1) RETURNING league_id`)
	teamA := scan(`INSERT INTO teams (name, country, league_id) VALUES ('A', 'zs', $1) RETURNING team_id`, leagueZs)
	teamB := scan(`INSERT INTO teams (name, country, league_id) VALUES ('B', 'zs', $1) RETURNING team_id`, leagueZs)
	teamC := scan(`INSERT INTO teams (name, country, league_id) VALUES ('C', 'zt', $1) RETURNING team_id`, leagueZt)
	teamD := scan(`INSERT INTO teams (name, country, league_id) VALUES ('D', 'zt', $1) RETURNING team_id`, leagueZt)

	// The test database persists across runs, so earlier runs' 'zs'/'zt' matches are still there:
	// measure what this run adds rather than absolute counts.
	zsTotalBefore, zsUnplayedBefore, err := GetSeasonMatchCounts(dbCtx, "zs", 10)
	if err != nil {
		t.Fatalf("GetSeasonMatchCounts(zs, 10) before seeding: %s", err)
	}
	ztTotalBefore, ztUnplayedBefore, err := GetSeasonMatchCounts(dbCtx, "zt", 3)
	if err != nil {
		t.Fatalf("GetSeasonMatchCounts(zt, 3) before seeding: %s", err)
	}

	// zs's season 10: fully played. zt's season 3: one match left unplayed.
	exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
		VALUES ($1, 10, 1, now() - interval '1 hour', $2, $3, true)`, leagueZs, teamA, teamB)
	exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
		VALUES ($1, 3, 1, now() - interval '1 hour', $2, $3, false)`, leagueZt, teamC, teamD)

	zsSeason, err := GetCurrentSeasonForCountry(dbCtx, "zs")
	if err != nil || zsSeason != 10 {
		t.Fatalf("GetCurrentSeasonForCountry(zs): got %d, %v; want 10, nil", zsSeason, err)
	}
	ztSeason, err := GetCurrentSeasonForCountry(dbCtx, "zt")
	if err != nil || ztSeason != 3 {
		t.Fatalf("GetCurrentSeasonForCountry(zt): got %d, %v; want 3, nil", ztSeason, err)
	}

	zsTotal, zsUnplayed, err := GetSeasonMatchCounts(dbCtx, "zs", zsSeason)
	if err != nil || zsTotal-zsTotalBefore != 1 || zsUnplayed-zsUnplayedBefore != 0 {
		t.Fatalf("GetSeasonMatchCounts(zs, 10): this run added total=%d unplayed=%d, %v; want 1, 0, nil",
			zsTotal-zsTotalBefore, zsUnplayed-zsUnplayedBefore, err)
	}
	ztTotal, ztUnplayed, err := GetSeasonMatchCounts(dbCtx, "zt", ztSeason)
	if err != nil || ztTotal-ztTotalBefore != 1 || ztUnplayed-ztUnplayedBefore != 1 {
		t.Fatalf("GetSeasonMatchCounts(zt, 3): this run added total=%d unplayed=%d, %v; want 1, 1, nil",
			ztTotal-ztTotalBefore, ztUnplayed-ztUnplayedBefore, err)
	}

	// Setting one country's season must not touch the other's.
	if err := SetCurrentSeasonForCountry(dbCtx, "zs", 11); err != nil {
		t.Fatalf("SetCurrentSeasonForCountry(zs, 11): %s", err)
	}
	if season, err := GetCurrentSeasonForCountry(dbCtx, "zt"); err != nil || season != 3 {
		t.Fatalf("GetCurrentSeasonForCountry(zt) after advancing zs: got %d, %v; want 3, nil", season, err)
	}
}
