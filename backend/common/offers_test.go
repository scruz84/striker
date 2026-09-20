package common

import (
	"testing"
	"time"
)

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

func TestOfferAcceptableAt(t *testing.T) {
	createdAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	want := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	if got := offerAcceptableAt(createdAt); !got.Equal(want) {
		t.Errorf("offerAcceptableAt(%v) = %v, want %v", createdAt, got, want)
	}
}

func TestCanAcceptOfferAt(t *testing.T) {
	createdAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"just made", createdAt, false},
		{"one hour later", createdAt.Add(time.Hour), false},
		{"one second before the window elapses", createdAt.Add(OfferNegotiationWindow - time.Second), false},
		{"exactly when the window elapses", createdAt.Add(OfferNegotiationWindow), true},
		{"well after the window elapses", createdAt.Add(3 * time.Hour), true},
	}
	for _, c := range cases {
		if got := canAcceptOfferAt(createdAt, c.now); got != c.want {
			t.Errorf("canAcceptOfferAt(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}
