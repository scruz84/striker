package common

import (
	"math/rand"
	"testing"
)

func TestRollSkillCap_NeverBelowStartingValueOrAbove99(t *testing.T) {
	rand.Seed(1)
	for _, start := range []int{1, 30, 60, 90, 95, 99} {
		for i := 0; i < 1000; i++ {
			max := rollSkillCap(start)
			if max < start {
				t.Fatalf("rollSkillCap(%d) = %d, expected at least the starting value", start, max)
			}
			if max > 99 {
				t.Fatalf("rollSkillCap(%d) = %d, expected at most 99", start, max)
			}
		}
	}
}

func TestRollSkillCap_MostRollsAreAModestGap(t *testing.T) {
	rand.Seed(2)
	const trials = 5000
	const start = 40
	smallGap := 0
	for i := 0; i < trials; i++ {
		if max := rollSkillCap(start); max-start <= 15 {
			smallGap++
		}
	}
	// ~70% of rolls should fall in the 5-15 point gap bucket - a generous margin around that for
	// a randomized test.
	if smallGap < trials/2 {
		t.Fatalf("expected most rolls to be a modest gap, got %d/%d <= 15 points above start", smallGap, trials)
	}
}
