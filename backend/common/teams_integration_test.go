package common

import (
	"database/sql"
	"os"
	"testing"
)

// TestGetAllTeamIds_ExcludesNationalTeams reproduces the reported bug: before
// GetAllTeamIdsStmtSql excluded a national team, every caller that iterates "every real,
// manageable club team" (RunAITeamManagement/RunAiTeamDailyActivity's AI loops,
// RespondToAllPendingOffers, PayAllTeamsWagesNow, TopUpTrainerMarket, and ProgressPlayers' own
// season-end academy top-up) picked up a national team too - which, having no real roster, led
// ProgressPlayers to generate it a full fake academy squad every season rollover (see
// init74_remove_national_team_academy_players.sql). Runs against a real Postgres
// (STRIKER_TEST_DB_DSN, see TestEnsureSchema_RestartDoesNotTouchPlayerData; skipped when unset).
func TestGetAllTeamIds_ExcludesNationalTeams(t *testing.T) {
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

	if _, err := db.Exec(`INSERT INTO countries (name) VALUES ('zy') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
	var clubTeamId, nationalTeamId int64
	if err := db.QueryRow(`INSERT INTO teams (name, country) VALUES ('Club', 'zy') RETURNING team_id`).
		Scan(&clubTeamId); err != nil {
		t.Fatalf("seeding club team: %s", err)
	}
	// A country has at most one national team (teams_national_team_country_idx), and the test
	// database persists across runs, so reuse the one an earlier run created.
	if err := db.QueryRow(`INSERT INTO teams (name, country, is_national_team) VALUES ('National', 'zy', true) `+
		`ON CONFLICT (country) WHERE is_national_team DO UPDATE SET name = EXCLUDED.name RETURNING team_id`).
		Scan(&nationalTeamId); err != nil {
		t.Fatalf("seeding national team: %s", err)
	}

	teamIds, err := GetAllTeamIds(dbCtx)
	if err != nil {
		t.Fatalf("GetAllTeamIds: %s", err)
	}

	seen := make(map[int64]bool, len(teamIds))
	for _, id := range teamIds {
		seen[id] = true
	}
	if !seen[clubTeamId] {
		t.Errorf("expected club team %d to be included, got %v", clubTeamId, teamIds)
	}
	if seen[nationalTeamId] {
		t.Errorf("expected national team %d to be excluded, got %v", nationalTeamId, teamIds)
	}
}
