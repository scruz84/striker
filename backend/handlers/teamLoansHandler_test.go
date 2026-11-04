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

type loansTestCache struct{ values map[string]string }

func (c *loansTestCache) GetValue(key string) (string, error) { return c.values[key], nil }
func (c *loansTestCache) SetValue(key, value string)          { c.values[key] = value }
func (c *loansTestCache) SetValueLongDuration(key, value string) {
	c.values[key] = value
}
func (c *loansTestCache) SetValueNoExpiry(key, value string) { c.values[key] = value }
func (c *loansTestCache) DeleteValue(key string)             { delete(c.values, key) }

// The public team endpoint carries loanedOut / loanedIn for an anonymous client (see
// openspec/specs/player-loans). Runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset).
func TestTeamHandler_ExposesLoanedOutAndLoanedIn(t *testing.T) {
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
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ('tl', 1) ON CONFLICT (name) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE season SET transfer_window_open = true`); err != nil {
		t.Fatal(err)
	}
	seedTeam := func(name string) int64 {
		var leagueId, teamId int64
		if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('tl', 1, '1A', 1) RETURNING league_id`).Scan(&leagueId); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`INSERT INTO teams (name, country, league_id, budget) VALUES ($1, 'tl', $2, 50000000) RETURNING team_id`, name, leagueId).Scan(&teamId); err != nil {
			t.Fatal(err)
		}
		return teamId
	}
	lender, borrower := seedTeam("Handler Lender"), seedTeam("Handler Borrower")
	var playerId, listingId int64
	if err := db.QueryRow(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield,
		dribbling, shot, "header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country, shirt_number)
		VALUES ($1, 'Test', 'Player', 0, 40, 40, 40, 58, 58, 40, 40, false, false, false, false, '', 1000, 26, 3, 'tl', 1)
		RETURNING player_id`, lender).Scan(&playerId); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee)
		VALUES ($1, $2, 1, 0, 60, 900000) RETURNING listing_id`, playerId, lender).Scan(&listingId); err != nil {
		t.Fatal(err)
	}
	if err := common.AcceptLoan(dbCtx, listingId, borrower); err != nil {
		t.Fatalf("AcceptLoan: %s", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/team/:"+ID, Team(dbCtx))
	get := func(teamId int64) map[string]json.RawMessage {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/team/"+strconv.FormatInt(teamId, 10), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("team %d: status = %d (%s)", teamId, rec.Code, rec.Body.String())
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decoding body: %s", err)
		}
		return body
	}

	body := get(lender)
	var loanedOut []map[string]any
	if err := json.Unmarshal(body["loanedOut"], &loanedOut); err != nil || len(loanedOut) != 1 {
		t.Fatalf("loanedOut = %s (%v), want one entry", body["loanedOut"], err)
	}
	for _, key := range []string{"player", "borrowingTeamId", "borrowingTeamName", "borrowingDivisionId", "borrowingDivisionName",
		"returnSeason", "wageContributionPercent", "buyOptionFee", "goalsForBorrower"} {
		if _, ok := loanedOut[0][key]; !ok {
			t.Errorf("loanedOut entry lacks %q: %v", key, loanedOut[0])
		}
	}
	if loanedOut[0]["borrowingTeamName"] != "Handler Borrower" || loanedOut[0]["buyOptionFee"] != float64(900000) {
		t.Errorf("loanedOut entry = %v", loanedOut[0])
	}
	if string(body["loanedIn"]) != "[]" {
		t.Errorf("lender loanedIn = %s, want []", body["loanedIn"])
	}

	body = get(borrower)
	var loanedIn []map[string]any
	if err := json.Unmarshal(body["loanedIn"], &loanedIn); err != nil || len(loanedIn) != 1 {
		t.Fatalf("loanedIn = %s (%v), want one entry", body["loanedIn"], err)
	}
	if loanedIn[0]["parentTeamName"] != "Handler Lender" || loanedIn[0]["playerId"] != float64(playerId) {
		t.Errorf("loanedIn entry = %v", loanedIn[0])
	}
	if string(body["loanedOut"]) != "[]" {
		t.Errorf("borrower loanedOut = %s, want []", body["loanedOut"])
	}
}
