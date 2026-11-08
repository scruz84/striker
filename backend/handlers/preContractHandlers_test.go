package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/scruz84/striker/common"
)

// The pre-contract endpoints (openspec/specs/pre-contracts): a refusal is a 200 carrying what the
// player wants, the rules are 409s, the buyer sees its offer and can withdraw it, and another
// club's player shows the public last-season marker. Needs STRIKER_TEST_DB_DSN.
func TestPreContractHandlers(t *testing.T) {
	r := newCounterHandlerRig(t)
	r.router.POST("/pre-contracts", MakePreContractOfferHandler(r.dbCtx))
	r.router.GET("/pre-contracts/sent", PreContractsSentHandler(r.dbCtx))
	r.router.GET("/pre-contracts/received", PreContractsReceivedHandler(r.dbCtx))
	r.router.DELETE("/pre-contracts/:"+ID, WithdrawPreContractHandler(r.dbCtx))
	get := func(as int64, path string) *httptest.ResponseRecorder {
		r.as = as
		rec := httptest.NewRecorder()
		r.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}
	body := func(wage int) string {
		return `{"playerId": ` + strconv.FormatInt(r.player, 10) + `, "wage": ` + strconv.Itoa(wage) + `, "years": 2}`
	}

	if code := r.post(r.buyerUser, "/pre-contracts", body(5000)); code != http.StatusConflict {
		t.Errorf("offer with 3 seasons left = %d, want 409", code)
	}
	if _, err := r.db.Exec(`UPDATE players SET contract_years_remaining = 1, intelligence = 1 WHERE player_id = $1`, r.player); err != nil {
		t.Fatal(err)
	}
	if code := r.post(r.buyerUser, "/pre-contracts", body(5000)); code != http.StatusConflict {
		t.Errorf("offer before half the season = %d, want 409", code)
	}
	if _, err := r.db.Exec(`INSERT INTO classification (league_id, team_id, season, matches, points)
		SELECT league_id, team_id, 1, 15, 20 FROM teams WHERE team_id = $1`, r.seller); err != nil {
		t.Fatal(err)
	}
	if code := r.post(r.buyerUser, "/pre-contracts", body(1)); code != http.StatusOK {
		t.Errorf("offer below the premium = %d, want 200 (a refusal)", code)
	}
	if code := r.post(r.buyerUser, "/pre-contracts", body(5000)); code != http.StatusOK {
		t.Fatalf("offer = %d, want 200", code)
	}

	var sent []common.PreContractOffer
	if err := json.Unmarshal(get(r.buyerUser, "/pre-contracts/sent").Body.Bytes(), &sent); err != nil || len(sent) != 1 {
		t.Fatalf("sent offers: %v, %v", sent, err)
	}
	if !sent[0].Player.ContractExpiring || sent[0].Player.ContractYearsRemaining != 0 || sent[0].KeepWage != 0 {
		t.Errorf("buyer's view of the player %+v: want the public marker only and no keep wage", sent[0])
	}
	var received []common.PreContractOffer
	if err := json.Unmarshal(get(r.sellerUser, "/pre-contracts/received").Body.Bytes(), &received); err != nil || len(received) != 1 {
		t.Fatalf("received offers: %v, %v", received, err)
	}
	if received[0].KeepWage == 0 || received[0].DecidesAt == nil {
		t.Errorf("seller's view %+v: want the keep wage and the decision time", received[0])
	}

	r.as = r.buyerUser
	rec := httptest.NewRecorder()
	r.router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/pre-contracts/"+strconv.FormatInt(sent[0].OfferId, 10), nil))
	if rec.Code != http.StatusOK {
		t.Errorf("withdraw = %d, want 200", rec.Code)
	}
	rec = httptest.NewRecorder()
	r.router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/pre-contracts/"+strconv.FormatInt(sent[0].OfferId, 10), nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("second withdraw = %d, want 404", rec.Code)
	}
}
