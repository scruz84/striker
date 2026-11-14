package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

// Saving a world structure rejects one that breaks the pyramid rule, and one with fewer places
// than an active country already has teams (openspec/specs/league-pyramid). Only rejections are
// exercised: a successful save would change the shared test database's world config. Needs
// STRIKER_TEST_DB_DSN.
func TestAdminSetWorldConfig_RejectsInvalidStructures(t *testing.T) {
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
	// A country with 9 club teams, more than an 8-team single-tier structure holds.
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ('xw', 1) ON CONFLICT (name) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var leagueId int64
	if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('xw', 1, 'I.1', 1) RETURNING league_id`).Scan(&leagueId); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		if _, err := db.Exec(`INSERT INTO teams (name, country, league_id) VALUES ('World Config Team', 'xw', $1)`, leagueId); err != nil {
			t.Fatal(err)
		}
	}
	before := common.GetWorldConfig(dbCtx)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/world-config", AdminSetWorldConfigHandler(dbCtx))
	put := func(tiers []common.TierConfig) (int, string) {
		body, _ := json.Marshal(map[string]any{"tiers": tiers})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/world-config", bytes.NewReader(body)))
		return rec.Code, rec.Body.String()
	}

	code, body := put([]common.TierConfig{
		{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10},
		{Tier: 2, Name: "II", Groups: 4, TeamsPerLeague: 10},
	})
	if code != http.StatusBadRequest || !strings.Contains(body, "tier 2 must have exactly 2 groups (twice tier 1)") {
		t.Errorf("1/4 structure: status %d, body %s", code, body)
	}

	code, body = put([]common.TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8}})
	if code != http.StatusBadRequest || !strings.Contains(body, "only has 8 places") {
		t.Errorf("8-place structure: status %d, body %s", code, body)
	}

	after := common.GetWorldConfig(dbCtx)
	if len(after) != len(before) {
		t.Errorf("a rejected save changed the world config: %v -> %v", before, after)
	}
}

// PUT /v1/admin/countries/:code/kickoff stores a country's league kickoff and rejects one that
// pushes the deepest tier past midnight (openspec/specs/match-scheduling "Country league kickoff
// setting"). Needs STRIKER_TEST_DB_DSN.
func TestAdminSetCountryKickoff(t *testing.T) {
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
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ('xk', 1) ON CONFLICT (name) DO UPDATE SET league_kickoff_minute = 840`); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/countries/:code/kickoff", AdminSetCountryKickoffHandler(dbCtx))
	put := func(body string) int {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/countries/xk/kickoff", strings.NewReader(body)))
		return rec.Code
	}
	// The saved structure has at least the default 5 tiers' worth of room to leave.
	if code := put(`{"hour": 23, "minute": 30}`); code != http.StatusBadRequest {
		t.Errorf("23:30: status %d, want 400", code)
	}
	if code := put(`{"hour": 13, "minute": 10}`); code != http.StatusBadRequest {
		t.Errorf("13:10 (not a quarter-hour): status %d, want 400", code)
	}
	if code := put(`{"hour": 13, "minute": 0}`); code != http.StatusOK {
		t.Fatalf("13:00: status %d, want 200", code)
	}
	if got := common.GetCountryLeagueKickoff(dbCtx, "xk"); got != 13*60 {
		t.Errorf("stored kickoff %d, want 780", got)
	}
}
