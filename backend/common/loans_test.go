package common

import "testing"

func TestCreateLoanListing_RejectsOutOfRangeDurationWithoutTouchingDatabase(t *testing.T) {
	// The duration bound check runs before any database access, so this is safe to exercise
	// with a nil DatabaseContext - same trick as TestSetTicketPrice_RejectsOutOfRange.
	if err := CreateLoanListing(nil, 1, 1, MinLoanDurationSeasons-1, 0, 0, 0); err != ErrInvalidLoanDuration {
		t.Fatalf("expected ErrInvalidLoanDuration for a too-short duration, got %v", err)
	}
	if err := CreateLoanListing(nil, 1, 1, MaxLoanDurationSeasons+1, 0, 0, 0); err != ErrInvalidLoanDuration {
		t.Fatalf("expected ErrInvalidLoanDuration for a too-long duration, got %v", err)
	}
}

func TestCreateLoanListing_RejectsNegativeLoanFeeWithoutTouchingDatabase(t *testing.T) {
	if err := CreateLoanListing(nil, 1, 1, MinLoanDurationSeasons, -1, 0, 0); err != ErrInvalidLoanFee {
		t.Fatalf("expected ErrInvalidLoanFee for a negative loan fee, got %v", err)
	}
}

func TestCreateLoanListing_RejectsOutOfRangeWageContributionPercentWithoutTouchingDatabase(t *testing.T) {
	if err := CreateLoanListing(nil, 1, 1, MinLoanDurationSeasons, 0, MinWageContributionPercent-1, 0); err != ErrInvalidWageContributionPercent {
		t.Fatalf("expected ErrInvalidWageContributionPercent for a negative percent, got %v", err)
	}
	if err := CreateLoanListing(nil, 1, 1, MinLoanDurationSeasons, 0, MaxWageContributionPercent+1, 0); err != ErrInvalidWageContributionPercent {
		t.Fatalf("expected ErrInvalidWageContributionPercent for a percent over 100, got %v", err)
	}
}

func TestCreateLoanListing_RejectsNegativeBuyOptionFeeWithoutTouchingDatabase(t *testing.T) {
	if err := CreateLoanListing(nil, 1, 1, MinLoanDurationSeasons, 0, 0, -1); err != ErrInvalidBuyOptionFee {
		t.Fatalf("expected ErrInvalidBuyOptionFee for a negative buy option fee, got %v", err)
	}
}
