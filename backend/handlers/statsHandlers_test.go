package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

func TestParseLeagueWeek(t *testing.T) {
	cases := []struct {
		name       string
		league     string
		week       string
		wantLeague int64
		wantWeek   int
		wantErr    bool
	}{
		{"week zero", "7", "0", 0, 0, true},
		{"negative week", "7", "-1", 0, 0, true},
		{"non-numeric week", "7", "abc", 0, 0, true},
		{"non-numeric league", "abc", "3", 0, 0, true},
		{"first week", "7", "1", 7, 1, false},
		{"past the season", "7", "30", 7, 30, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			league, week, err := parseLeagueWeek(c.league, c.week)
			if (err != nil) != c.wantErr {
				t.Fatalf("parseLeagueWeek(%q, %q) error = %v, wantErr %v", c.league, c.week, err, c.wantErr)
			}
			if !c.wantErr && (league != c.wantLeague || week != c.wantWeek) {
				t.Errorf("parseLeagueWeek(%q, %q) = %d, %d; want %d, %d", c.league, c.week, league, week, c.wantLeague, c.wantWeek)
			}
		})
	}
}

// The division live scores endpoint (openspec division-live-scores "Division live scores
// endpoint"). Needs STRIKER_TEST_DB_DSN, no Redis: a league with nothing in play never reads a
// broadcast.
func TestLeagueLiveScoresHandler(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	dbCtx := &common.DatabaseContext{Db: db, ObjectCache: &loansTestCache{values: map[string]string{}}}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/division/:"+ID+"/live", LeagueLiveScoresHandler(dbCtx))
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	if rec := get("/division/abc/live"); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid id: got %d, want 400", rec.Code)
	}
	rec := get("/division/-1/live")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("empty league: got %d %q, want 200 []", rec.Code, rec.Body.String())
	}
}
