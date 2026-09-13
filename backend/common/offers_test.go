package common

import "testing"

func TestMakeTransferOffer_RejectsNonPositiveAmountWithoutTouchingDatabase(t *testing.T) {
	// The amount check runs before any database access, so this is safe to exercise with a nil
	// DatabaseContext - same trick as TestSetTicketPrice_RejectsOutOfRange.
	if _, err := MakeTransferOffer(nil, 1, 1, 0, 1000, 2); err != ErrInvalidOfferAmount {
		t.Fatalf("expected ErrInvalidOfferAmount for a zero amount, got %v", err)
	}
	if _, err := MakeTransferOffer(nil, 1, 1, -100, 1000, 2); err != ErrInvalidOfferAmount {
		t.Fatalf("expected ErrInvalidOfferAmount for a negative amount, got %v", err)
	}
}

func TestMakeTransferOffer_RejectsInvalidPersonalTermsWithoutTouchingDatabase(t *testing.T) {
	// Checked right after amount, still before any database access.
	if _, err := MakeTransferOffer(nil, 1, 1, 1000000, 0, 2); err != ErrInvalidPersonalTerms {
		t.Fatalf("expected ErrInvalidPersonalTerms for a zero wage, got %v", err)
	}
	if _, err := MakeTransferOffer(nil, 1, 1, 1000000, 1000, 0); err != ErrInvalidPersonalTerms {
		t.Fatalf("expected ErrInvalidPersonalTerms for zero years, got %v", err)
	}
	if _, err := MakeTransferOffer(nil, 1, 1, 1000000, 1000, MaxContractYearsRemaining+1); err != ErrInvalidPersonalTerms {
		t.Fatalf("expected ErrInvalidPersonalTerms for too many years, got %v", err)
	}
}
