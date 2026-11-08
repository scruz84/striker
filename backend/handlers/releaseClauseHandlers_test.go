package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/scruz84/striker/common"
)

// The release-clause endpoint (openspec/specs/release-clauses "Paying a clause"): too soon is a
// 409, a paid clause moves the player, and the seller can't pay his own clause. Needs
// STRIKER_TEST_DB_DSN.
func TestPayReleaseClauseHandler(t *testing.T) {
	r := newCounterHandlerRig(t)
	r.router.POST("/release-clauses", PayReleaseClauseHandler(r.dbCtx))
	pay := func(as int64, wage int) *httptest.ResponseRecorder {
		r.as = as
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/release-clauses",
			strings.NewReader(`{"playerId": `+strconv.FormatInt(r.player, 10)+`, "wage": `+strconv.Itoa(wage)+`, "years": 2}`))
		req.Header.Set("Content-Type", "application/json")
		r.router.ServeHTTP(rec, req)
		return rec
	}
	if rec := pay(r.buyerUser, 5000); rec.Code != http.StatusConflict {
		t.Errorf("player without a clause = %d, want 409", rec.Code)
	}
	if _, err := r.db.Exec(`UPDATE players SET release_clause = 5000000, release_clause_set_at = now() - interval '10 days', intelligence = 1 WHERE player_id = $1`, r.player); err != nil {
		t.Fatal(err)
	}
	if rec := pay(r.buyerUser, 5000); rec.Code != http.StatusConflict {
		t.Errorf("clause inside its lock-in = %d, want 409", rec.Code)
	}
	if _, err := r.db.Exec(`UPDATE players SET release_clause_set_at = now() - interval '15 days' WHERE player_id = $1`, r.player); err != nil {
		t.Fatal(err)
	}
	if rec := pay(r.sellerUser, 5000); rec.Code != http.StatusConflict {
		t.Errorf("seller paying his own clause = %d, want 409", rec.Code)
	}
	rec := pay(r.buyerUser, 5000)
	var result common.ReleaseClauseResult
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &result) != nil || !result.Accepted || result.Fee != 5000000 {
		t.Fatalf("paying the clause = %d %s, want an accepted 5,000,000 signing", rec.Code, rec.Body.String())
	}
	var team int64
	if err := r.db.QueryRow(`SELECT team_id FROM players WHERE player_id = $1`, r.player).Scan(&team); err != nil || team != r.buyer {
		t.Errorf("player belongs to %d, want the buyer %d", team, r.buyer)
	}
}
