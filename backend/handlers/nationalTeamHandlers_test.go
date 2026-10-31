package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

func nationalTeamsRouter(dbCtx *common.DatabaseContext) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/national-teams/round", NationalTeamRoundHandler(dbCtx))
	router.GET("/national-teams/results", NationalTeamResultsHandler(dbCtx))
	router.GET("/national-teams/world-cups", WorldCupHistoryHandler(dbCtx))
	return router
}

func getNationalTeams(router *gin.Engine, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// Malformed paging or country parameters are rejected before the database is touched.
func TestNationalTeamResultsHandler_BadParameters(t *testing.T) {
	router := nationalTeamsRouter(nil)
	for _, path := range []string{
		"/national-teams/results?page=-1",
		"/national-teams/results?page=abc",
		"/national-teams/results?page=999999",
		"/national-teams/results?country=ES",
		"/national-teams/results?country=es%27--",
		"/national-teams/results?country=x",
	} {
		if rec := getNationalTeams(router, path); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", path, rec.Code, http.StatusBadRequest)
		}
	}
}

// Runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset).
func TestNationalTeamFeedHandlers(t *testing.T) {
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
	router := nationalTeamsRouter(dbCtx)

	if rec := getNationalTeams(router, "/national-teams/results?country=zz9"); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown country: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	rec := getNationalTeams(router, "/national-teams/round")
	if rec.Code != http.StatusOK {
		t.Fatalf("round: status = %d (%s)", rec.Code, rec.Body.String())
	}
	var round []common.NationalMatch
	if err := json.Unmarshal(rec.Body.Bytes(), &round); err != nil {
		t.Fatalf("round body: %s", err)
	}

	rec = getNationalTeams(router, "/national-teams/results?page=0")
	if rec.Code != http.StatusOK {
		t.Fatalf("results: status = %d (%s)", rec.Code, rec.Body.String())
	}
	var results common.NationalTeamResults
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("results body: %s", err)
	}
	if len(results.Matches) > common.NationalTeamResultsPageSize {
		t.Errorf("results page has %d matches, want at most %d", len(results.Matches), common.NationalTeamResultsPageSize)
	}

	rec = getNationalTeams(router, "/national-teams/world-cups")
	if rec.Code != http.StatusOK {
		t.Fatalf("world cups: status = %d (%s)", rec.Code, rec.Body.String())
	}
}
