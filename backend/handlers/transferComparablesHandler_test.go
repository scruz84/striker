package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

func comparablesRouter(dbCtx *common.DatabaseContext) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/player/:"+ID+"/transfer-comparables", TransferComparablesHandler(dbCtx))
	return router
}

func getComparables(router *gin.Engine, id string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/player/"+id+"/transfer-comparables", nil))
	return rec
}

// A non-numeric id is rejected before the database is touched.
func TestTransferComparablesHandler_BadId(t *testing.T) {
	if rec := getComparables(comparablesRouter(nil), "abc"); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// Runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset).
func TestTransferComparablesHandler_UnknownAndKnownPlayer(t *testing.T) {
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
	router := comparablesRouter(dbCtx)

	if rec := getComparables(router, "999999999"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown player: status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ('th', 1) ON CONFLICT (name) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var leagueId, teamId, playerId int64
	if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('th', 1, 'L1', 1) RETURNING league_id`).Scan(&leagueId); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO teams (name, country, league_id, budget) VALUES ('Handler FC', 'th', $1, 1) RETURNING team_id`, leagueId).Scan(&teamId); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM players WHERE team_id = $1`, teamId) })
	if err := db.QueryRow(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield,
		dribbling, shot, "header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country, shirt_number)
		VALUES ($1, 'Test', 'Player', 0, 40, 40, 40, 58, 58, 40, 40, false, false, false, false, '', 1000, 26, 3, 'th', 1)
		RETURNING player_id`, teamId).Scan(&playerId); err != nil {
		t.Fatal(err)
	}

	rec := getComparables(router, strconv.FormatInt(playerId, 10))
	if rec.Code != http.StatusOK {
		t.Fatalf("known player: status = %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body common.TransferComparablesResult
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body: %s", err)
	}
	if body.Position != common.MIDFIELD || body.Age != 26 || body.AverageSkill != 45 {
		t.Errorf("reference = position %d age %d average %d, want %d / 26 / 45", body.Position, body.Age, body.AverageSkill, common.MIDFIELD)
	}
	if body.Comparables == nil {
		t.Error("comparables must be an empty list, not null")
	}
}
