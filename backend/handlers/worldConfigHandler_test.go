package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

// Saving a world structure rejects one that breaks the pyramid rule, and refuses one with fewer
// places than an active country has teams until it is confirmed (openspec/specs/league-pyramid).
// Only refusals are exercised: a successful save would change the shared test database's world
// config. Needs STRIKER_TEST_DB_DSN.
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

	// Fewer places than xw's 9 CPU teams is refused without confirmation: a 409 listing the teams it
	// would lose, or a 400 when another country of the shared test database has more user teams
	// than 8 places. The exact 409 counts are pinned by TestAdminCountryStructure.
	code, body = put([]common.TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8}})
	refused := (code == http.StatusConflict && strings.Contains(body, `"xw":1`)) ||
		(code == http.StatusBadRequest && strings.Contains(body, "user teams but this structure only has 8 places"))
	if !refused {
		t.Errorf("8-place structure: status %d, body %s; want it refused", code, body)
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

// PUT/GET/DELETE /v1/admin/countries/:code/structure (league-pyramid "Per-country structure",
// "Confirming a structure that dissolves teams", "Structure fits existing teams"): a country with 9
// CPU teams and 2 user teams gets a 1-tier x 8 structure only once confirmed, a 1/2 x 8 one with no
// confirmation, never one with fewer places than its user teams, and removing it needs the same
// check against the global structure. An inactive country is 404. Needs STRIKER_TEST_DB_DSN.
func TestAdminCountryStructure(t *testing.T) {
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
	country := fmt.Sprintf("y%03d", time.Now().UnixNano()%1000)
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = common.SetCountryStructure(dbCtx, country, nil) })
	var leagueId int64
	if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'I.1', 1) RETURNING league_id`, country).Scan(&leagueId); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 11; i++ {
		var owner any
		if i < 2 {
			var user int64
			if err := db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ('cs', 'dev', $1) RETURNING user_id`,
				fmt.Sprintf("country-structure-%d-%d@test", time.Now().UnixNano(), i)).Scan(&user); err != nil {
				t.Fatal(err)
			}
			owner = user
		}
		if _, err := db.Exec(`INSERT INTO teams (name, country, league_id, owner_id) VALUES ('Country Structure Team', $1, $2, $3)`, country, leagueId, owner); err != nil {
			t.Fatal(err)
		}
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/countries/:code/structure", AdminGetCountryStructureHandler(dbCtx))
	router.PUT("/countries/:code/structure", AdminSetCountryStructureHandler(dbCtx))
	router.DELETE("/countries/:code/structure", AdminClearCountryStructureHandler(dbCtx))
	call := func(method, path string, body any) (int, string) {
		var reader *bytes.Reader
		if body != nil {
			raw, _ := json.Marshal(body)
			reader = bytes.NewReader(raw)
		} else {
			reader = bytes.NewReader(nil)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(method, path, reader))
		return rec.Code, rec.Body.String()
	}
	oneTier := []common.TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8}}
	twoTiers := []common.TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8}, {Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 8}}
	path := "/countries/" + country + "/structure"

	if code, body := call(http.MethodPut, path, map[string]any{"tiers": oneTier}); code != http.StatusConflict || !strings.Contains(body, `"`+country+`":3`) {
		t.Errorf("unconfirmed shrink: %d %s, want 409 listing 3 teams", code, body)
	}
	if got, _ := common.GetCountryStructure(dbCtx, country); got != nil {
		t.Fatalf("a refused save stored %+v", got)
	}
	if code, body := call(http.MethodPut, path, map[string]any{"tiers": oneTier, "confirmDissolve": true}); code != http.StatusOK {
		t.Fatalf("confirmed shrink: %d %s", code, body)
	}
	if code, body := call(http.MethodGet, path, nil); code != http.StatusOK || !strings.Contains(body, `"teamsPerLeague":8`) {
		t.Errorf("GET after save: %d %s", code, body)
	}
	if code, body := call(http.MethodPut, path, map[string]any{"tiers": twoTiers}); code != http.StatusOK {
		t.Errorf("a structure that dissolves nothing needs no confirmation: %d %s", code, body)
	}
	// The smallest valid structure has 8 places, more than 2 user teams, so make all 11 teams
	// users' to check the user-team rule.
	if _, err := db.Exec(`UPDATE teams SET owner_id = (SELECT min(user_id) FROM users) WHERE country = $1`, country); err != nil {
		t.Fatal(err)
	}
	if code, body := call(http.MethodPut, path, map[string]any{"tiers": oneTier, "confirmDissolve": true}); code != http.StatusBadRequest || !strings.Contains(body, "11 user teams") {
		t.Errorf("fewer places than user teams: %d %s, want 400", code, body)
	}
	if code, _ := call(http.MethodPut, "/countries/zz7/structure", map[string]any{"tiers": twoTiers}); code != http.StatusNotFound {
		t.Errorf("inactive country: %d, want 404", code)
	}
	if code, body := call(http.MethodDelete, path+"?confirmDissolve=true", nil); code != http.StatusOK {
		t.Errorf("removing the country's own structure: %d %s", code, body)
	}
	if got, _ := common.GetCountryStructure(dbCtx, country); got != nil {
		t.Errorf("after DELETE the country still has %+v", got)
	}
}
