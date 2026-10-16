package common

import (
	"database/sql"
	"os"
	"testing"
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
	if err := db.QueryRow(`INSERT INTO players (team_id) VALUES ($1) RETURNING player_id`, teamId).Scan(&playerId); err != nil {
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

	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("second EnsureSchema: %s", err)
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
