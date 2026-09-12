package common

import "testing"

func TestTrainerWageFromSkill_ScalesWithSkill(t *testing.T) {
	low := TrainerWageFromSkill(TrainerSkillMin)
	high := TrainerWageFromSkill(TrainerSkillMax)
	if high <= low {
		t.Fatal("expected a higher-skilled trainer to cost more wage")
	}
	if got := TrainerWageFromSkill(50); got != 50*TrainerWagePerSkillPoint {
		t.Errorf("TrainerWageFromSkill(50) = %d, want %d", got, 50*TrainerWagePerSkillPoint)
	}
}

func TestAcademySkillMax_NoTrainerMatchesHistoricalMax(t *testing.T) {
	// A team with no trainer (skill 0) must generate academy prospects exactly like before this
	// feature existed - see engine.ProgressPlayers, which passes 0 for such a team.
	if got := academySkillMax(0); got != AcademySkillMax {
		t.Errorf("academySkillMax(0) = %d, want the unmodified AcademySkillMax %d", got, AcademySkillMax)
	}
}

func TestAcademySkillMax_RisesWithTrainerSkill(t *testing.T) {
	low := academySkillMax(TrainerSkillMin)
	high := academySkillMax(TrainerSkillMax)
	if high <= low {
		t.Fatal("expected a higher-skilled trainer to raise the academy prospect ceiling more")
	}
}

func TestTrainerSkillFromPlayerSkills_AveragesTheSevenAttributes(t *testing.T) {
	// All seven at 60 - squarely inside [TrainerSkillMin, TrainerSkillMax] - so the average
	// itself, 60, should come back untouched.
	if got := trainerSkillFromPlayerSkills(60, 60, 60, 60, 60, 60, 60); got != 60 {
		t.Errorf("trainerSkillFromPlayerSkills(all 60) = %d, want 60", got)
	}
}

func TestTrainerSkillFromPlayerSkills_ClampsToTrainerRange(t *testing.T) {
	if got := trainerSkillFromPlayerSkills(1, 1, 1, 1, 1, 1, 1); got != TrainerSkillMin {
		t.Errorf("expected a very low-skilled player to clamp up to TrainerSkillMin (%d), got %d", TrainerSkillMin, got)
	}
	if got := trainerSkillFromPlayerSkills(99, 99, 99, 99, 99, 99, 99); got != TrainerSkillMax {
		t.Errorf("expected a maxed-out player to clamp down to TrainerSkillMax (%d), got %d", TrainerSkillMax, got)
	}
}
