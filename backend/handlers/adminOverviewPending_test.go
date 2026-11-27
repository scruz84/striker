package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

// Spec scenario "Leftover groups" (league-pyramid "Admin sees why a country is pending"): a
// country with 1/2/3 groups of 8 under its own 1/2/4 x 8 structure is listed in the admin overview
// with group III.4 missing and the doubling rule broken. Needs STRIKER_TEST_DB_DSN.
func TestAdminOverview_PendingRestructureDetails(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	dbCtx := &common.DatabaseContext{Db: db, ObjectCache: common.NewRedisCache()}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	// The overview reads the world's settings row, which a fresh test database may not have yet.
	var settingsRows int
	if err := db.QueryRow(`SELECT count(*) FROM season`).Scan(&settingsRows); err != nil {
		t.Fatal(err)
	}
	if settingsRows == 0 {
		if _, err := db.Exec(`INSERT INTO season DEFAULT VALUES`); err != nil {
			t.Fatal(err)
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	country := fmt.Sprintf("zp%c%c", 'a'+r.Intn(26), 'a'+r.Intn(26))
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country); err != nil {
		t.Fatal(err)
	}
	names := []string{"I", "II", "III"}
	for tier, groups := range []int{1, 2, 3} {
		for g := 1; g <= groups; g++ {
			var leagueId int64
			if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, $2, $3, $4) RETURNING league_id`,
				country, g, common.LeagueName(names[tier], g), tier+1).Scan(&leagueId); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 8; i++ {
				if _, err := db.Exec(`INSERT INTO teams (name, country, league_id) VALUES ('Pending Team', $1, $2)`, country, leagueId); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	structure := []common.TierConfig{
		{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8},
		{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 8},
		{Tier: 3, Name: "III", Groups: 4, TeamsPerLeague: 8},
	}
	if err := common.SetCountryStructure(dbCtx, country, structure); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/overview", AdminOverviewHandler(dbCtx))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/overview", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	var overview common.AdminOverview
	if err := json.Unmarshal(rec.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	listed := false
	for _, code := range overview.PendingRestructure {
		if code == country {
			listed = true
		}
	}
	if !listed {
		t.Errorf("%s is not in pendingRestructure %v", country, overview.PendingRestructure)
	}
	var details *common.PendingRestructure
	for i := range overview.PendingRestructureDetails {
		if overview.PendingRestructureDetails[i].Country == country {
			details = &overview.PendingRestructureDetails[i]
		}
	}
	if details == nil {
		t.Fatalf("%s has no pendingRestructureDetails entry", country)
	}
	diffs := details.Differences
	if len(diffs) != 2 ||
		diffs[0].Kind != common.ShapeDiffMissing || diffs[0].League != "III.4" || diffs[0].Tier != 3 || diffs[0].Group != 4 ||
		diffs[1].Kind != common.ShapeDiffRatio {
		t.Errorf("differences %+v, want III.4 missing and the doubling rule broken", diffs)
	}
}
