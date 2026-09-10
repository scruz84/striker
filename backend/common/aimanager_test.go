package common

import "testing"

func TestRandomAiLevel_AlwaysWithinRange(t *testing.T) {
	for _, tier := range []int{1, 2, 3, 99} {
		for i := 0; i < 200; i++ {
			level := RandomAiLevel(tier)
			if level < AiLevelCaretaker || level > AiLevelDirector {
				t.Fatalf("RandomAiLevel(%d) = %d, want a value in [%d, %d]", tier, level, AiLevelCaretaker, AiLevelDirector)
			}
		}
	}
}

func TestRandomAiLevel_UnknownTierFallsBackToDefaultWeights(t *testing.T) {
	// Tier 99 isn't in aiLevelWeightsByTier, so it should behave like defaultAiLevelWeights
	// {60, 30, 10} - Caretaker most common, Director rarest. Statistical, not exact, but 2000
	// trials makes a false failure astronomically unlikely.
	counts := map[int]int{}
	for i := 0; i < 2000; i++ {
		counts[RandomAiLevel(99)]++
	}
	if counts[AiLevelCaretaker] <= counts[AiLevelDirector] {
		t.Errorf("expected Caretaker (%d) to be far more common than Director (%d) under default weights",
			counts[AiLevelCaretaker], counts[AiLevelDirector])
	}
}

func TestRandomAiLevel_Tier1SkewsTowardDirector(t *testing.T) {
	// Tier 1 uses {10, 30, 60} - Director should be the most common level, unlike the default
	// weights where Caretaker dominates.
	counts := map[int]int{}
	for i := 0; i < 2000; i++ {
		counts[RandomAiLevel(1)]++
	}
	if counts[AiLevelDirector] <= counts[AiLevelCaretaker] {
		t.Errorf("expected Director (%d) to be more common than Caretaker (%d) for tier 1",
			counts[AiLevelDirector], counts[AiLevelCaretaker])
	}
}
