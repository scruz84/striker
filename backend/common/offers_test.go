package common

import "testing"

func TestMakeTransferOffer_RejectsNonPositiveAmountWithoutTouchingDatabase(t *testing.T) {
	// The amount check runs before any database access, so this is safe to exercise with a nil
	// DatabaseContext - same trick as TestSetTicketPrice_RejectsOutOfRange.
	if err := MakeTransferOffer(nil, 1, 1, 0); err != ErrInvalidOfferAmount {
		t.Fatalf("expected ErrInvalidOfferAmount for a zero amount, got %v", err)
	}
	if err := MakeTransferOffer(nil, 1, 1, -100); err != ErrInvalidOfferAmount {
		t.Fatalf("expected ErrInvalidOfferAmount for a negative amount, got %v", err)
	}
}
