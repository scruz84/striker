package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/scruz84/striker/common"
)

// The lineup endpoints take and give back conditional orders (openspec add-conditional-match-orders
// "Conditional match orders"), validated like scheduled substitutions. Needs STRIKER_TEST_DB_DSN.

func (r *friendlySubsRig) putOrders(path string, orders []common.LineupOrder) (int, string) {
	r.t.Helper()
	body, _ := json.Marshal(SetLineupRequest{Formation: r.lineup.Formation, Positions: r.lineup.Positions, Orders: orders})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestLineupHandlers_ConditionalOrders(t *testing.T) {
	r := newFriendlySubsRig(t)
	mentality := common.LineupOrder{FromMinute: 60, Condition: common.ORDER_CONDITION_LOSING, Action: common.ORDER_ACTION_MENTALITY, Mentality: common.MENTALITY_ATTACKING}
	substitution := common.LineupOrder{FromMinute: 75, Condition: common.ORDER_CONDITION_WINNING, Action: common.ORDER_ACTION_SUBSTITUTION, PlayerOutId: r.starter[0], PlayerInId: r.bench[0]}

	// Valid orders.
	valid := []common.LineupOrder{mentality, substitution}
	if code, body := r.putOrders("/lineup", valid); code != http.StatusOK {
		t.Fatalf("valid orders = %d: %s", code, body)
	}
	if got := r.get("/lineup").Orders; !reflect.DeepEqual(got, valid) {
		t.Errorf("stored orders = %+v, want %+v", got, valid)
	}

	// Too many orders.
	if code, _ := r.putOrders("/lineup", []common.LineupOrder{mentality, mentality, mentality, mentality}); code != http.StatusBadRequest {
		t.Errorf("4 orders = %d, want 400", code)
	}
	// Invalid substitution order: the incoming player is a starter.
	bad := substitution
	bad.PlayerInId = r.starter[1]
	if code, _ := r.putOrders("/lineup", []common.LineupOrder{bad}); code != http.StatusBadRequest {
		t.Errorf("incoming starter = %d, want 400", code)
	}
	for _, o := range []common.LineupOrder{
		{FromMinute: 90, Condition: common.ORDER_CONDITION_LOSING, Action: common.ORDER_ACTION_MENTALITY},
		{FromMinute: 0, Condition: common.ORDER_CONDITION_LOSING, Action: common.ORDER_ACTION_MENTALITY},
		{FromMinute: 60, Condition: 5, Action: common.ORDER_ACTION_MENTALITY},
		{FromMinute: 60, Condition: common.ORDER_CONDITION_LOSING, Action: 2},
		{FromMinute: 60, Condition: common.ORDER_CONDITION_LOSING, Action: common.ORDER_ACTION_MENTALITY, Mentality: 2},
	} {
		if code, _ := r.putOrders("/lineup", []common.LineupOrder{o}); code != http.StatusBadRequest {
			t.Errorf("order %+v = %d, want 400", o, code)
		}
	}
	if got := r.get("/lineup").Orders; !reflect.DeepEqual(got, valid) {
		t.Errorf("a rejected save changed the stored orders: %+v", got)
	}

	// Saving without orders clears them; the friendly default never carries any.
	if code, _ := r.putOrders("/lineup", nil); code != http.StatusOK {
		t.Fatal(code)
	}
	if got := r.get("/lineup").Orders; len(got) != 0 {
		t.Errorf("orders after a save without any = %+v", got)
	}
	if code, _ := r.putOrders("/lineup/friendly", valid); code != http.StatusOK {
		t.Fatal(code)
	}
	if got := r.get("/lineup/friendly").Orders; len(got) != 0 {
		t.Errorf("friendly default orders = %+v, want none", got)
	}
}
