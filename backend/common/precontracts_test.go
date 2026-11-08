package common

import "testing"

// pre-contracts "Player demands": "Below the premium".
func TestPreContractMinimumWage(t *testing.T) {
	if got := PreContractMinimumWage(2000, IntelligenceAware); got != 2200 {
		t.Errorf("minimum for an expectation of 2,000: %d, want 2,200", got)
	}
	if got := PreContractMinimumWage(1001, IntelligenceAware); got != 1102 {
		t.Errorf("minimum rounds up: %d, want 1,102", got)
	}
	// "Mastermind": 1.20 x; a Naive player only 1.05 x.
	if got := PreContractMinimumWage(2000, IntelligenceMastermind); got != 2400 {
		t.Errorf("Mastermind minimum: %d, want 2,400", got)
	}
	if got := PreContractMinimumWage(2000, IntelligenceNaive); got != 2100 {
		t.Errorf("Naive minimum: %d, want 2,100", got)
	}
	if got := PreContractKeepWage(2400, IntelligenceMastermind); got != 2000 {
		t.Errorf("Mastermind keep wage against 2,400: %d, want 2,000", got)
	}
}

// pre-contracts "Right to counter-offer": "Matching a rival" and "Better offer during the wait".
func TestPreContractKeepWage(t *testing.T) {
	for _, c := range []struct{ best, want int }{{3300, 3000}, {3520, 3200}, {2200, 2000}, {0, 0}} {
		if got := PreContractKeepWage(c.best, IntelligenceAware); got != c.want {
			t.Errorf("keep wage against %d: %d, want %d", c.best, got, c.want)
		}
	}
}

// pre-contracts "Making a pre-contract offer": "Too early" and "Offer made".
func TestPreContractSeasonOpen(t *testing.T) {
	if PreContractSeasonOpen(14, 16) {
		t.Error("14 of 30 rounds played opened pre-contracts")
	}
	if !PreContractSeasonOpen(15, 15) {
		t.Error("15 of 30 rounds played did not open pre-contracts")
	}
	if PreContractSeasonOpen(0, 0) {
		t.Error("a league with no season opened pre-contracts")
	}
}
