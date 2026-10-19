package common

import (
	"testing"
	"time"
)

func TestOfferRetryAllowed(t *testing.T) {
	now := time.Now()
	if ok, minBid := OfferRetryAllowed(nil, now); !ok || minBid != 0 {
		t.Errorf("no history: got %v, %d; want allowed with no floor", ok, minBid)
	}
	fresh := &OfferRejection{Amount: 10_000, RejectedAt: now.Add(-AiOfferCooldown / 2)}
	if ok, _ := OfferRetryAllowed(fresh, now); ok {
		t.Error("inside the cooldown a team must leave the player alone")
	}
	old := &OfferRejection{Amount: 10_000, RejectedAt: now.Add(-AiOfferCooldown - time.Minute)}
	ok, minBid := OfferRetryAllowed(old, now)
	if !ok {
		t.Fatal("after the cooldown a team may try again")
	}
	if minBid != 11_000 {
		t.Errorf("rebid floor = %d, want 10%% above the rejected 10000", minBid)
	}
}
