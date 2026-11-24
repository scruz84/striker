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

// Saving the instant archive setting (openspec/specs/match-data-retention "Admin configures the
// archive"): a valid save is stored, a count outside 0-10 or a missing field is rejected with no
// change. Needs STRIKER_TEST_DB_DSN; the setting is restored afterwards.
func TestAdminSetInstantArchive(t *testing.T) {
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
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatal(err)
	}
	before, err := common.GetInstantArchiveSettings(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = common.SetInstantArchiveSettings(dbCtx, before) })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/instant-archive", AdminSetInstantArchiveHandler(dbCtx))
	put := func(body string) int {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/instant-archive", strings.NewReader(body)))
		return rec.Code
	}
	want := common.InstantArchiveSettings{Enabled: true, KeepSeasons: 2}
	if code := put(`{"enabled": true, "keepSeasons": 2}`); code != http.StatusOK {
		t.Fatalf("valid save = %d, want 200", code)
	}
	if got, _ := common.GetInstantArchiveSettings(dbCtx); got != want {
		t.Fatalf("saved %+v, want %+v", got, want)
	}
	for _, body := range []string{`{"enabled": false, "keepSeasons": 11}`, `{"enabled": false, "keepSeasons": -1}`, `{"enabled": false}`, `{"keepSeasons": 3}`, `nope`} {
		if code := put(body); code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", body, code)
		}
	}
	if got, _ := common.GetInstantArchiveSettings(dbCtx); got != want {
		t.Errorf("a rejected save changed the setting to %+v", got)
	}
}

// Saving the instant retention: a valid save is stored, a value outside 0-365 or a missing
// field is rejected with no change. Needs STRIKER_TEST_DB_DSN; the setting is restored afterwards.
func TestAdminSetInstantRetention(t *testing.T) {
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
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatal(err)
	}
	before, err := common.GetInstantKeepDays(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = common.SetInstantKeepDays(dbCtx, before) })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/instant-retention", AdminSetInstantRetentionHandler(dbCtx))
	put := func(body string) int {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/instant-retention", strings.NewReader(body)))
		return rec.Code
	}
	if code := put(`{"keepDays": 14}`); code != http.StatusOK {
		t.Fatalf("valid save = %d, want 200", code)
	}
	if got, _ := common.GetInstantKeepDays(dbCtx); got != 14 {
		t.Fatalf("saved %d days, want 14", got)
	}
	for _, body := range []string{`{"keepDays": 366}`, `{"keepDays": -1}`, `{}`, `nope`} {
		if code := put(body); code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", body, code)
		}
	}
	if got, _ := common.GetInstantKeepDays(dbCtx); got != 14 {
		t.Errorf("a rejected save changed the setting to %d days", got)
	}
}
