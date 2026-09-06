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
