package common

import "testing"

func TestCreateLoanListing_RejectsOutOfRangeDurationWithoutTouchingDatabase(t *testing.T) {
	// The duration bound check runs before any database access, so this is safe to exercise
	// with a nil DatabaseContext - same trick as TestSetTicketPrice_RejectsOutOfRange.
	if err := CreateLoanListing(nil, 1, 1, MinLoanDurationSeasons-1); err != ErrInvalidLoanDuration {
		t.Fatalf("expected ErrInvalidLoanDuration for a too-short duration, got %v", err)
	}
	if err := CreateLoanListing(nil, 1, 1, MaxLoanDurationSeasons+1); err != ErrInvalidLoanDuration {
		t.Fatalf("expected ErrInvalidLoanDuration for a too-long duration, got %v", err)
	}
}
