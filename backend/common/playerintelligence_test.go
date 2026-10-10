package common

import "testing"

func TestRandomIntelligence_AlwaysWithinRange(t *testing.T) {
	for i := 0; i < 500; i++ {
		level := RandomIntelligence()
		if level < IntelligenceNaive || level > IntelligenceMastermind {
			t.Fatalf("RandomIntelligence() = %d, want a value in [%d, %d]", level, IntelligenceNaive, IntelligenceMastermind)
		}
	}
}

func TestRandomIntelligence_NaiveIsMoreCommonThanMastermind(t *testing.T) {
	// intelligenceWeights = {30, 25, 20, 15, 10} - Naive should be far more common than
	// Mastermind. Statistical, not exact, but 2000 trials makes a false failure astronomically
	// unlikely.
	counts := map[int]int{}
	for i := 0; i < 2000; i++ {
		counts[RandomIntelligence()]++
	}
	if counts[IntelligenceNaive] <= counts[IntelligenceMastermind] {
		t.Errorf("expected Naive (%d) to be more common than Mastermind (%d)", counts[IntelligenceNaive], counts[IntelligenceMastermind])
	}
}

func TestIntelligenceReactionFactor_KnownTiers(t *testing.T) {
	cases := map[int]float64{
		IntelligenceNaive:      0,
		IntelligenceCasual:     0.25,
		IntelligenceAware:      0.5,
		IntelligenceShrewd:     0.75,
		IntelligenceMastermind: 1.0,
	}
	for level, want := range cases {
		if got := IntelligenceReactionFactor(level); got != want {
			t.Errorf("IntelligenceReactionFactor(%d) = %v, want %v", level, got, want)
		}
	}
}

func TestIntelligenceReactionFactor_UnknownLevelFallsBackToZero(t *testing.T) {
	for _, level := range []int{0, -1, 6, 99} {
		if got := IntelligenceReactionFactor(level); got != 0 {
			t.Errorf("IntelligenceReactionFactor(%d) = %v, want 0", level, got)
		}
	}
}

func TestIntelligenceReactionFactor_MonotonicAcrossTiers(t *testing.T) {
	prev := IntelligenceReactionFactor(IntelligenceNaive)
	for level := IntelligenceCasual; level <= IntelligenceMastermind; level++ {
		cur := IntelligenceReactionFactor(level)
		if cur <= prev {
			t.Fatalf("expected reaction factor to strictly increase by tier, level %d (%v) <= previous (%v)", level, cur, prev)
		}
		prev = cur
	}
}
