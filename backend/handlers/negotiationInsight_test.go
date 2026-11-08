package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// player-negotiation "Negotiation insight": "After a refused offer" and "Another club". Needs
// STRIKER_TEST_DB_DSN.
func TestNegotiationInsight(t *testing.T) {
	r := newCounterHandlerRig(t)
	r.router.POST("/offers", MakeOfferHandler(r.dbCtx))
	r.router.GET("/players/:"+ID, PlayerDetailsHandler(r.dbCtx))
	if _, err := r.db.Exec(`UPDATE players SET intelligence = 4 WHERE player_id = $1`, r.player); err != nil {
		t.Fatal(err)
	}
	style := func(as int64) *int {
		r.as = as
		rec := httptest.NewRecorder()
		r.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/players/"+strconv.FormatInt(r.player, 10), nil))
		var details struct {
			NegotiationStyle *int `json:"negotiationStyle"`
		}
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &details) != nil {
			t.Fatalf("player details = %d %s", rec.Code, rec.Body.String())
		}
		return details.NegotiationStyle
	}
	if s := style(r.buyerUser); s != nil {
		t.Fatalf("the buyer sees tier %d before negotiating", *s)
	}

	r.as = r.buyerUser
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/offers",
		strings.NewReader(`{"playerId": `+strconv.FormatInt(r.player, 10)+`, "amount": 1000, "wage": 1, "years": 1}`))
	req.Header.Set("Content-Type", "application/json")
	r.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"accepted":false`) {
		t.Fatalf("offer on bad terms = %d %s, want a refusal", rec.Code, rec.Body.String())
	}

	if s := style(r.buyerUser); s == nil || *s != 4 {
		t.Errorf("after a refused offer the buyer sees %v, want Shrewd (4)", s)
	}
	if s := style(r.sellerUser); s != nil {
		t.Errorf("the seller, who never negotiated, sees tier %d", *s)
	}
}
