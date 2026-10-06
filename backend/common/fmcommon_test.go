package common

import "testing"

func TestPlayerWageFromSkills(t *testing.T) {
	got := PlayerWageFromSkills(40, 40, 40, 40, 40, 40, 40)
	want := 40 * 7 * WagePerSkillPointNumerator / WagePerSkillPointDenominator
	if got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
}

func TestPlayerWageFromSkills_HigherSkillsCostMore(t *testing.T) {
	low := PlayerWageFromSkills(30, 30, 30, 30, 30, 30, 30)
	high := PlayerWageFromSkills(60, 60, 60, 60, 60, 60, 60)
	if high <= low {
		t.Fatalf("expected a higher-skilled player to cost more: low=%d high=%d", low, high)
	}
}
