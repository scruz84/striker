package common

import (
	"database/sql"
	"os"
	"testing"
)

// pyramidTestDb opens STRIKER_TEST_DB_DSN (skipping when unset), applies the schema and wipes
// country's own leagues and teams, so a test can lay out its own pyramid for it.
func pyramidTestDb(t *testing.T, country string) *DatabaseContext {
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
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
	// Move any previous run's teams out of the way rather than deleting them (many tables
	// reference teams), then retire the country's old leagues.
	if _, err := db.Exec(`UPDATE teams SET league_id = NULL WHERE league_id IN (SELECT league_id FROM leagues WHERE country = $1)`, country); err != nil {
		t.Fatalf("detaching old teams: %s", err)
	}
	if _, err := db.Exec(`UPDATE leagues SET archived_season = 0 WHERE country = $1 AND archived_season IS NULL`, country); err != nil {
		t.Fatalf("archiving old leagues: %s", err)
	}
	return dbCtx
}

func TestArchivedLeaguesAreNotListed(t *testing.T) {
	const country = "xa"
	dbCtx := pyramidTestDb(t, country)
	insert := func(name string, tier, group int, archived bool) int64 {
		var id int64
		var archivedSeason any
		if archived {
			archivedSeason = 1
		}
		if err := dbCtx.Db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier, archived_season) VALUES ($1, $2, $3, $4, $5) RETURNING league_id`,
			country, group, name, tier, archivedSeason).Scan(&id); err != nil {
			t.Fatalf("inserting league %s: %s", name, err)
		}
		return id
	}
	active := insert("I.1", 1, 1, false)
	archived := insert("II.3", 2, 3, true)

	ids, err := GetLeagueIdsForCountry(dbCtx, country)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != active {
		t.Errorf("GetLeagueIdsForCountry = %v, want only %d", ids, active)
	}
	withTier, err := GetLeaguesWithTierForCountry(dbCtx, country)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range withTier {
		if l.LeagueId == archived {
			t.Errorf("GetLeaguesWithTierForCountry lists archived league %d", archived)
		}
	}
	all, err := GetLeaguesWithTier(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range all {
		if l.LeagueId == archived {
			t.Errorf("GetLeaguesWithTier lists archived league %d", archived)
		}
	}
	directory, err := GetLeagueDirectory(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range directory {
		for _, l := range c.Leagues {
			if l.LeagueId == archived {
				t.Errorf("the league directory lists archived league %d", archived)
			}
		}
	}
	shape, err := GetCountryPyramidShape(dbCtx, country)
	if err != nil {
		t.Fatal(err)
	}
	if shape.TierCount() != 1 || len(shape.Leagues) != 1 {
		t.Errorf("pyramid shape = %+v, want only the active tier-1 league", shape)
	}
	// Lookups by id still work.
	name, err := GetDivisionName(dbCtx, archived)
	if err != nil || name != "II.3" {
		t.Errorf("GetDivisionName(archived) = %q, %v", name, err)
	}
}
