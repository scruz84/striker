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
