package common

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

// TestEnsureSchema_RestartDoesNotTouchPlayerData runs EnsureSchema against a real, empty Postgres
// (STRIKER_TEST_DB_DSN, e.g. "host=/tmp port=5544 user=postgres dbname=striker_test sslmode=disable";
// the database must not hold a schema yet) and checks that a second call - what every --svc/--engine
// restart does - leaves existing rows alone. Skipped when the variable isn't set.
func TestEnsureSchema_RestartDoesNotTouchPlayerData(t *testing.T) {
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
		t.Fatalf("first EnsureSchema: %s", err)
	}

	// EnsureSchema creates tables only; the world (countries, teams, players) is seeded
	// elsewhere, so a minimal owned player is inserted here.
	var teamId, playerId int64
	if _, err := db.Exec(`INSERT INTO countries (name) VALUES ('zz') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
	if err := db.QueryRow(`INSERT INTO teams (name, country) VALUES ('t', 'zz') RETURNING team_id`).Scan(&teamId); err != nil {
		t.Fatalf("seeding team: %s", err)
	}
	if err := db.QueryRow(`INSERT INTO players (birth_day, team_id) VALUES (0, $1) RETURNING player_id`, teamId).Scan(&playerId); err != nil {
		t.Fatalf("seeding player: %s", err)
	}
	// 9 is outside the [2, 4] range init21_contracts.sql would re-randomize into.
	if _, err := db.Exec(`UPDATE players SET contract_years_remaining = 9 WHERE player_id = $1`, playerId); err != nil {
		t.Fatalf("setting contract: %s", err)
	}
	var namesBefore int
	if err := db.QueryRow(`SELECT count(*) FROM names_en`).Scan(&namesBefore); err != nil {
		t.Fatalf("counting names_en: %s", err)
	}

	poolsBefore := namePoolCounts(t, db)
	started := time.Now()
	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("second EnsureSchema: %s", err)
	}
	t.Logf("warm EnsureSchema (everything already migrated) took %s", time.Since(started).Round(time.Millisecond))
	checkNamePools(t, db, poolsBefore)
	for _, country := range KnownCountries {
		first, last, err := GetRandomName(dbCtx, country.Code)
		if err != nil || first == "" || last == "" {
			t.Errorf("GetRandomName(%q) = %q, %q, %v", country.Code, first, last, err)
		}
	}

	var contract, namesAfter int
	if err := db.QueryRow(`SELECT contract_years_remaining FROM players WHERE player_id = $1`, playerId).Scan(&contract); err != nil {
		t.Fatalf("reading contract: %s", err)
	}
	if contract != 9 {
		t.Errorf("contract_years_remaining changed across EnsureSchema: got %d, want 9", contract)
	}
	if err := db.QueryRow(`SELECT count(*) FROM names_en`).Scan(&namesAfter); err != nil {
		t.Fatalf("counting names_en: %s", err)
	}
	if namesAfter != namesBefore {
		t.Errorf("names_en rows changed across EnsureSchema: %d -> %d", namesBefore, namesAfter)
	}
}

// namePoolCounts returns "<table> -> row count" for every known country's name tables.
func namePoolCounts(t *testing.T, db *sql.DB) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for _, country := range KnownCountries {
		for _, table := range []string{"names_" + country.Code, "lastnames_" + country.Code} {
			var n int
			if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
				t.Fatalf("counting %s: %s", table, err)
			}
			counts[table] = n
		}
	}
	return counts
}

// checkNamePools asserts what re-applying the schema must leave behind: the name pools are
// unchanged by the second run, every generated pool is at least 300 first names / 500 surnames,
// the pre-existing 30-name pool of Russia kept its rows, and name_pool_versions holds version 2 for
// the 210 countries with a generated pool (every known country but Spain).
func checkNamePools(t *testing.T, db *sql.DB, before map[string]int) {
	t.Helper()
	after := namePoolCounts(t, db)
	for table, n := range before {
		if after[table] != n {
			t.Errorf("%s rows changed across EnsureSchema: %d -> %d", table, n, after[table])
		}
	}
	for _, country := range KnownCountries {
		if country.Code == "es" {
			continue
		}
		if n := after["names_"+country.Code]; n < minPoolFirstNames {
			t.Errorf("names_%s has %d rows, want at least %d", country.Code, n, minPoolFirstNames)
		}
		if n := after["lastnames_"+country.Code]; n < minPoolLastNames {
			t.Errorf("lastnames_%s has %d rows, want at least %d", country.Code, n, minPoolLastNames)
		}
	}
	// init69 seeded Russia with 30 first names and 30 surnames; the generated pool extends it.
	for _, name := range []string{"Ivan", "Dmitri", "Alexei", "Sergei", "Grigori"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM names_ru WHERE "name" = $1`, name).Scan(&n); err != nil || n != 1 {
			t.Errorf("names_ru lost or duplicated the original name %q: count=%d err=%v", name, n, err)
		}
	}
	var versions int
	if err := db.QueryRow(`SELECT count(*) FROM name_pool_versions WHERE version = 2`).Scan(&versions); err != nil {
		t.Fatalf("counting name_pool_versions: %s", err)
	}
	if want := len(KnownCountries) - 1; versions != want {
		t.Errorf("name_pool_versions has %d countries at version 2, want %d", versions, want)
	}
}
