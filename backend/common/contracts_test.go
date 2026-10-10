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

// sampleSkills is an arbitrary, fixed set of the seven core skills used across the
// *WithIntelligence tests below - only their combined total (PlayerWageFromSkills' own input)
// matters for these tests, not any individual value.
var sampleSkills = [7]int{60, 55, 50, 65, 58, 52, 61} // sum = 401

func TestExpectedWageWithIntelligence_NaiveOrFlatTrendMatchesSkillsOnlyWage(t *testing.T) {
	s := sampleSkills
	base := PlayerWageFromSkills(s[0], s[1], s[2], s[3], s[4], s[5], s[6])

	// A Mastermind who hasn't changed at all (trend 0) demands nothing extra.
	if got := ExpectedWageWithIntelligence(s[0], s[1], s[2], s[3], s[4], s[5], s[6], IntelligenceMastermind, 0); got != base {
		t.Fatalf("flat trend: ExpectedWageWithIntelligence() = %d, want %d (base)", got, base)
	}
	// A Naive player who improved a lot (trend 20) still doesn't notice - factor 0.
	if got := ExpectedWageWithIntelligence(s[0], s[1], s[2], s[3], s[4], s[5], s[6], IntelligenceNaive, 20); got != base {
		t.Fatalf("Naive improved: ExpectedWageWithIntelligence() = %d, want %d (base)", got, base)
	}
	// A declining trend never lowers the wage below the skills-only figure.
	if got := ExpectedWageWithIntelligence(s[0], s[1], s[2], s[3], s[4], s[5], s[6], IntelligenceMastermind, -20); got != base {
		t.Fatalf("declined: ExpectedWageWithIntelligence() = %d, want %d (base)", got, base)
	}
}

func TestExpectedWageWithIntelligence_ImprovedSkillsRaiseWageByTier(t *testing.T) {
	s := sampleSkills
	base := PlayerWageFromSkills(s[0], s[1], s[2], s[3], s[4], s[5], s[6])
	const trend = 14 // a clearly non-trivial improvement

	var previous int
	for _, level := range []int{IntelligenceNaive, IntelligenceCasual, IntelligenceAware, IntelligenceShrewd, IntelligenceMastermind} {
		wage := ExpectedWageWithIntelligence(s[0], s[1], s[2], s[3], s[4], s[5], s[6], level, trend)
		if wage < base {
			t.Fatalf("level %d: wage %d fell below base %d for a positive trend", level, wage, base)
		}
		if level > IntelligenceNaive && wage <= previous {
			t.Fatalf("level %d: wage %d did not exceed the previous tier's wage %d", level, wage, previous)
		}
		previous = wage
	}
	if previous <= base {
		t.Fatalf("expected a Mastermind's wage demand (%d) to exceed the skills-only base (%d)", previous, base)
	}
}

func TestMaxAcceptableRenewalYearsWithIntelligence_FlatOrImprovingTrendMatchesAgeOnly(t *testing.T) {
	base := MaxAcceptableRenewalYears(28)
	if got := MaxAcceptableRenewalYearsWithIntelligence(28, IntelligenceMastermind, 0); got != base {
		t.Fatalf("flat trend: got %d, want %d (base)", got, base)
	}
	if got := MaxAcceptableRenewalYearsWithIntelligence(28, IntelligenceMastermind, 20); got != base {
		t.Fatalf("improving trend: got %d, want %d (base)", got, base)
	}
}

func TestMaxAcceptableRenewalYearsWithIntelligence_DeclineExtendsTermByTierNeverPastCap(t *testing.T) {
	const age = 32 // base years = 2, so there's room to grow before hitting the cap
	base := MaxAcceptableRenewalYears(age)
	const trend = -25 // a large, sustained decline

	naive := MaxAcceptableRenewalYearsWithIntelligence(age, IntelligenceNaive, trend)
	if naive != base {
		t.Fatalf("Naive: got %d, want %d (base, unaffected by trend)", naive, base)
	}

	mastermind := MaxAcceptableRenewalYearsWithIntelligence(age, IntelligenceMastermind, trend)
	if mastermind <= base {
		t.Fatalf("Mastermind: expected a declining player to accept more than the base %d years, got %d", base, mastermind)
	}
	if mastermind > MaxContractYearsRemaining {
		t.Fatalf("Mastermind: %d years exceeds the hard cap of %d", mastermind, MaxContractYearsRemaining)
	}
}
