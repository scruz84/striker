package handlers

import (
	"encoding/json"
	"testing"
)

// The price dialog reads both fields (openspec fix-empty-gate-high-ticket-price), so they must
// always be present in the JSON, even at zero.
func TestFinancesResponse_ExposesTicketPriceGuidance(t *testing.T) {
	raw, err := json.Marshal(FinancesResponse{RecommendedTicketPrice: 18, MaxTicketPrice: 39})
	if err != nil {
		t.Fatalf("marshalling: %s", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshalling: %s", err)
	}
	if got["recommendedTicketPrice"] != float64(18) || got["maxTicketPrice"] != float64(39) {
		t.Fatalf("recommendedTicketPrice/maxTicketPrice = %v/%v, want 18/39", got["recommendedTicketPrice"], got["maxTicketPrice"])
	}
}
