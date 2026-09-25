package common

import (
	"testing"
)

// UpgradeStadium itself always needs a live database (it locks and updates rows in a
// transaction, like SetDefaultLineup/ExecuteTransfer elsewhere in this package) and can't be
// unit tested standalone - what's worth pinning down without one is its cost formula.
func TestStadiumUpgradeCost_ScalesWithCurrentCapacity(t *testing.T) {
	small := int64(1000) * StadiumUpgradeCostPerCurrentSeat
	big := int64(5000) * StadiumUpgradeCostPerCurrentSeat
	if big <= small {
		t.Fatal("expected upgrading a bigger stadium to cost more")
	}
}

func TestSetTicketPrice_RejectsOutOfRange(t *testing.T) {
	// SetTicketPrice validates before touching the database, so this doesn't need a live one.
	if err := SetTicketPrice(nil, 1, MinTicketPrice-1); err != ErrInvalidTicketPrice {
		t.Fatalf("expected ErrInvalidTicketPrice for a too-low price, got %v", err)
	}
	if err := SetTicketPrice(nil, 1, MaxTicketPrice+1); err != ErrInvalidTicketPrice {
		t.Fatalf("expected ErrInvalidTicketPrice for a too-high price, got %v", err)
	}
}
