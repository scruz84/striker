package engine

import (
	"math/rand"
	"testing"
)

func TestShouldRetire_NeverBelowEligibleAge(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for age := 0; age < RetirementEligibleAge; age++ {
		if ShouldRetire(age, r) {
			t.Fatalf("player aged %d should never retire (eligible age is %d)", age, RetirementEligibleAge)
		}
	}
}

func TestShouldRetire_AlwaysAtForcedAge(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for age := RetirementForcedAge; age < RetirementForcedAge+10; age++ {
		if !ShouldRetire(age, r) {
			t.Fatalf("player aged %d should always retire (forced age is %d)", age, RetirementForcedAge)
		}
	}
}

func TestShouldRetire_ChanceRisesWithAge(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	const trials = 5000

	countAt := func(age int) int {
		count := 0
		for i := 0; i < trials; i++ {
			if ShouldRetire(age, r) {
				count++
			}
		}
		return count
	}

	youngCount := countAt(RetirementEligibleAge)
	oldCount := countAt(RetirementForcedAge - 1)

	if oldCount <= youngCount {
		t.Fatalf("expected retirement chance to rise with age: at %d got %d/%d, at %d got %d/%d",
			RetirementEligibleAge, youngCount, trials, RetirementForcedAge-1, oldCount, trials)
	}
}

func TestAgeTrendFor(t *testing.T) {
	if trend := ageTrendFor(20); trend <= 0 {
		t.Errorf("expected a young player (20) to trend upward, got %d", trend)
	}
	if trend := ageTrendFor(27); trend != 0 {
		t.Errorf("expected a player in their prime (27) to trend flat, got %d", trend)
	}
	if trend := ageTrendFor(34); trend >= 0 {
		t.Errorf("expected an older player (34) to trend downward, got %d", trend)
	}
}

func TestDriftSkill_StaysWithinBounds(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 1000; i++ {
		if v := driftSkill(1, 35, r); v < 1 || v > 99 {
			t.Fatalf("drifted skill out of bounds: %d", v)
		}
		if v := driftSkill(99, 20, r); v < 1 || v > 99 {
			t.Fatalf("drifted skill out of bounds: %d", v)
		}
	}
}
