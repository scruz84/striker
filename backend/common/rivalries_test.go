package common

import (
	"math"
	"testing"
)

// Rivalry levels and decay (openspec rivalries "Rivalry levels", "Rivalry decay").

func TestRivalryLevel_Thresholds(t *testing.T) {
	for _, c := range []struct {
		score float64
		want  int
	}{{0, RIVALRY_NONE}, {7.9, RIVALRY_NONE}, {8, RIVALRY_RIVAL}, {15.9, RIVALRY_RIVAL}, {16, RIVALRY_CLASICO}, {40, RIVALRY_CLASICO}} {
		if got := RivalryLevel(c.score); got != c.want {
			t.Errorf("RivalryLevel(%v) = %d, want %d", c.score, got, c.want)
		}
	}
}

func TestDecayRivalryScore_OldFeud(t *testing.T) {
	once := DecayRivalryScore(20)
	twice := DecayRivalryScore(once)
	if math.Abs(once-12) > 1e-9 || math.Abs(twice-7.2) > 1e-9 {
		t.Errorf("a pair at 20 decays to %v then %v, want 12 then 7.2", once, twice)
	}
	if got := DecayRivalryScore(0.8); got != 0 {
		t.Errorf("0.8 decays to %v, want 0 (below the 0.5 floor)", got)
	}
}
