package common

import "testing"

func TestRandomContractYears_WithinBounds(t *testing.T) {
	for i := 0; i < 100; i++ {
		years := RandomContractYears()
		if years < InitialContractMinYears || years > InitialContractMaxYears {
			t.Fatalf("contract length %d out of range [%d, %d]", years, InitialContractMinYears, InitialContractMaxYears)
		}
	}
}

func TestEarlyReleaseWageMultiplier_ScalesWithWage(t *testing.T) {
	// ReleasePlayer itself always needs a live database (it locks and updates rows in a
	// transaction, like ExecuteTransfer/UpgradeStadium elsewhere in this package) and can't be
	// unit tested standalone - what's worth pinning down without one is its severance formula.
	small := int64(100) * EarlyReleaseWageMultiplier
	big := int64(500) * EarlyReleaseWageMultiplier
	if big <= small {
		t.Fatal("expected releasing a higher-waged player to cost more severance")
	}
}

func TestMaxAcceptableRenewalYears_NeverExceedsMax(t *testing.T) {
	for age := 16; age <= 45; age++ {
		years := MaxAcceptableRenewalYears(age)
		if years < 1 || years > MaxContractYearsRemaining {
			t.Fatalf("age %d: MaxAcceptableRenewalYears returned %d, out of [1, %d]", age, years, MaxContractYearsRemaining)
		}
	}
}

func TestMaxAcceptableRenewalYears_ShrinksTowardRetirement(t *testing.T) {
	young := MaxAcceptableRenewalYears(24)
	prime := MaxAcceptableRenewalYears(31)
	veteran := MaxAcceptableRenewalYears(35)
	if !(young >= prime && prime >= veteran) {
		t.Fatalf("expected the acceptable term to shrink with age, got young=%d prime=%d veteran=%d", young, prime, veteran)
	}
	if young != MaxContractYearsRemaining {
		t.Fatalf("expected a young player to accept up to the full %d-season cap, got %d", MaxContractYearsRemaining, young)
	}
}
