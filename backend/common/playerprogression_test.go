package common

import "testing"

func TestSkillTrendFromEvolution_NilIsZero(t *testing.T) {
	if trend := SkillTrendFromEvolution(nil); trend != 0 {
		t.Fatalf("SkillTrendFromEvolution(nil) = %d, want 0", trend)
	}
}

func TestSkillTrendFromEvolution_PositiveWhenSkillsImproved(t *testing.T) {
	e := &PlayerEvolution{
		GoalKeeper: SkillDelta{Before: 50, After: 50},
		Defense:    SkillDelta{Before: 50, After: 54},
		Lateral:    SkillDelta{Before: 50, After: 50},
		Pass:       SkillDelta{Before: 50, After: 52},
		Midfield:   SkillDelta{Before: 50, After: 50},
		Dribbling:  SkillDelta{Before: 50, After: 50},
		Shot:       SkillDelta{Before: 50, After: 50},
	}
	if trend := SkillTrendFromEvolution(e); trend != 6 {
		t.Fatalf("SkillTrendFromEvolution() = %d, want 6", trend)
	}
}

func TestSkillTrendFromEvolution_NegativeWhenSkillsDeclined(t *testing.T) {
	e := &PlayerEvolution{
		GoalKeeper: SkillDelta{Before: 50, After: 50},
		Defense:    SkillDelta{Before: 50, After: 46},
		Lateral:    SkillDelta{Before: 50, After: 50},
		Pass:       SkillDelta{Before: 50, After: 50},
		Midfield:   SkillDelta{Before: 50, After: 50},
		Dribbling:  SkillDelta{Before: 50, After: 50},
		Shot:       SkillDelta{Before: 50, After: 47},
	}
	if trend := SkillTrendFromEvolution(e); trend != -7 {
		t.Fatalf("SkillTrendFromEvolution() = %d, want -7", trend)
	}
}

func TestSkillTrendFromEvolution_ZeroWhenUnchanged(t *testing.T) {
	e := &PlayerEvolution{
		GoalKeeper: SkillDelta{Before: 60, After: 60},
		Defense:    SkillDelta{Before: 60, After: 60},
		Lateral:    SkillDelta{Before: 60, After: 60},
		Pass:       SkillDelta{Before: 60, After: 60},
		Midfield:   SkillDelta{Before: 60, After: 60},
		Dribbling:  SkillDelta{Before: 60, After: 60},
		Shot:       SkillDelta{Before: 60, After: 60},
	}
	if trend := SkillTrendFromEvolution(e); trend != 0 {
		t.Fatalf("SkillTrendFromEvolution() = %d, want 0", trend)
	}
}
