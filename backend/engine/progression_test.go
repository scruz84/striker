package engine

import (
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
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
		if v := driftSkill(1, 35, 0, r); v < 1 || v > 99 {
			t.Fatalf("drifted skill out of bounds: %d", v)
		}
		if v := driftSkill(99, 20, 90, r); v < 1 || v > 99 {
			t.Fatalf("drifted skill out of bounds: %d", v)
		}
	}
}

func TestTrainerBonus(t *testing.T) {
	if b := trainerBonus(0); b != 0 {
		t.Errorf("no trainer (skill 0) should give no bonus, got %d", b)
	}
	if b := trainerBonus(common.TrainerSkillMin - 1); b != 0 {
		t.Errorf("a trainer below TrainerSkillMin should give no bonus, got %d", b)
	}
	if b := trainerBonus(common.TrainerSkillMin); b != 1 {
		t.Errorf("a solid trainer (skill %d) should give +1, got %d", common.TrainerSkillMin, b)
	}
	if b := trainerBonus(79); b != 1 {
		t.Errorf("a trainer just below the elite tier (79) should give +1, got %d", b)
	}
	if b := trainerBonus(80); b != 2 {
		t.Errorf("an elite trainer (80) should give +2, got %d", b)
	}
	if b := trainerBonus(common.TrainerSkillMax); b != 2 {
		t.Errorf("the best possible trainer should give +2, got %d", b)
	}
}

func TestDriftSkill_TrainerRaisesExpectedGrowth(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	const trials = 5000
	const startValue = 50
	const age = 27 // ageTrendFor(27) == 0, isolating the trainer's effect

	sumWithout, sumWith := 0, 0
	for i := 0; i < trials; i++ {
		sumWithout += driftSkill(startValue, age, 0, r)
		sumWith += driftSkill(startValue, age, common.TrainerSkillMax, r)
	}

	avgWithout := float64(sumWithout) / trials
	avgWith := float64(sumWith) / trials
	if avgWith <= avgWithout {
		t.Fatalf("expected an elite trainer to raise average skill growth: without=%.2f with=%.2f", avgWithout, avgWith)
	}
}

func TestPrimaryPosition_NoMinutesIsNotPlayed(t *testing.T) {
	position, totalMinutes, played := primaryPosition(nil)
	if played {
		t.Fatalf("expected a player with no recorded minutes to not be considered played, got position=%d totalMinutes=%d",
			position, totalMinutes)
	}
	if totalMinutes != 0 {
		t.Fatalf("expected total minutes 0, got %d", totalMinutes)
	}
}

func TestPrimaryPosition_PicksMostPlayedPosition(t *testing.T) {
	position, totalMinutes, played := primaryPosition(map[int]int{
		common.MIDFIELD: 30,
		common.STRIKER:  60,
	})
	if !played {
		t.Fatal("expected played to be true")
	}
	if position != common.STRIKER {
		t.Fatalf("expected the position with more minutes (STRIKER) to win, got %d", position)
	}
	if totalMinutes != 90 {
		t.Fatalf("expected total minutes to sum across positions (90), got %d", totalMinutes)
	}
}

func TestPrimaryPosition_TiesBrokenByLowestPositionId(t *testing.T) {
	position, _, played := primaryPosition(map[int]int{
		common.STRIKER: 45,
		common.DEFENSE: 45,
	})
	if !played {
		t.Fatal("expected played to be true")
	}
	if position != common.DEFENSE {
		t.Fatalf("expected a tie to be broken by the lowest position id (DEFENSE=%d), got %d", common.DEFENSE, position)
	}
}

func striker50() common.Player {
	return common.Player{
		GoalKeeper: 50, Defense: 50, Lateral: 50, Pass: 50, Midfield: 50, Dribbling: 50, Shot: 50,
	}
}

func TestDriftSkillsForPosition_ZeroPlayChanceChangesNothing(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	p := striker50()
	goalKeeper, defense, lateral, pass, midfield, dribbling, shot :=
		driftSkillsForPosition(p, common.STRIKER, 0, 27, 0, r)

	if goalKeeper != p.GoalKeeper || defense != p.Defense || lateral != p.Lateral || pass != p.Pass ||
		midfield != p.Midfield || dribbling != p.Dribbling || shot != p.Shot {
		t.Fatalf("expected every skill unchanged at playChance 0, got %d %d %d %d %d %d %d",
			goalKeeper, defense, lateral, pass, midfield, dribbling, shot)
	}
}

func TestDriftSkillsForPosition_OnlyTouchesSkillsRelevantToStriker(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	p := striker50()
	// playChance 1 guarantees every relevant skill rolls the drift - see driftSkillsForPosition's
	// maybeDrift (r.Float32() is always < 1).
	goalKeeper, defense, lateral, pass, midfield, dribbling, shot :=
		driftSkillsForPosition(p, common.STRIKER, 1, 27, 0, r)

	if goalKeeper != p.GoalKeeper {
		t.Errorf("a striker's goalkeeping should never drift, got %d (was %d)", goalKeeper, p.GoalKeeper)
	}
	if defense != p.Defense {
		t.Errorf("a striker's defense should never drift, got %d (was %d)", defense, p.Defense)
	}
	if midfield != p.Midfield {
		t.Errorf("a striker's midfield should never drift, got %d (was %d)", midfield, p.Midfield)
	}
	if lateral != p.Lateral {
		t.Errorf("a (non-lateral) striker's lateral should never drift, got %d (was %d)", lateral, p.Lateral)
	}
	for name, v := range map[string]int{"pass": pass, "dribbling": dribbling, "shot": shot} {
		if v < 1 || v > 99 {
			t.Errorf("%s out of bounds after drift: %d", name, v)
		}
	}
}

func TestDriftSkillsForPosition_LateralStrikerAlsoDriftsLateral(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	p := striker50()
	_, defense, lateral, pass, midfield, dribbling, shot :=
		driftSkillsForPosition(p, common.LATERAL_STRIKER_LEFT, 1, 27, 0, r)

	if defense != p.Defense || midfield != p.Midfield {
		t.Errorf("a lateral striker's defense/midfield should never drift, got defense=%d midfield=%d", defense, midfield)
	}
	for name, v := range map[string]int{"lateral": lateral, "pass": pass, "dribbling": dribbling, "shot": shot} {
		if v < 1 || v > 99 {
			t.Errorf("%s out of bounds after drift: %d", name, v)
		}
	}
}
