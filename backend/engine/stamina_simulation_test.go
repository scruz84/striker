package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

func TestFatigueMultiplier_MaxStaminaPlayerNeverFatigues(t *testing.T) {
	if m := fatigueMultiplier(99, FatigueMatchLength); m != 1 {
		t.Fatalf("expected a Stamina-99 player to never be fatigued, got multiplier %v", m)
	}
}

func TestFatigueMultiplier_ZeroMinutesPlayedNeverFatigues(t *testing.T) {
	if m := fatigueMultiplier(0, 0); m != 1 {
		t.Fatalf("expected a player who hasn't played any minutes yet to be unaffected, got multiplier %v", m)
	}
}

func TestFatigueMultiplier_WorstCaseHitsExactlyMaxPenalty(t *testing.T) {
	m := fatigueMultiplier(0, FatigueMatchLength)
	want := 1 - FatigueMaxPenalty
	if m != want {
		t.Fatalf("expected a Stamina-0 player who played the full match to drop to exactly %v, got %v", want, m)
	}
}

func TestFatigueMultiplier_RisesWithMinutesPlayed(t *testing.T) {
	early := fatigueMultiplier(30, 10)
	late := fatigueMultiplier(30, 80)
	if late >= early {
		t.Fatalf("expected more minutes played to fatigue a player further: early=%v late=%v", early, late)
	}
}

func TestFatigueMultiplier_WorsensWithLowerStamina(t *testing.T) {
	highStamina := fatigueMultiplier(90, 60)
	lowStamina := fatigueMultiplier(10, 60)
	if lowStamina >= highStamina {
		t.Fatalf("expected lower Stamina to fatigue a player further for the same minutes: high=%v low=%v", highStamina, lowStamina)
	}
}

func TestFatigueMultiplier_MinutesBeyondMatchLengthDoNotWorsenFurther(t *testing.T) {
	atLength := fatigueMultiplier(0, FatigueMatchLength)
	beyond := fatigueMultiplier(0, FatigueMatchLength+30)
	if beyond != atLength {
		t.Fatalf("expected minutes beyond FatigueMatchLength to cap at the same penalty: atLength=%v beyond=%v", atLength, beyond)
	}
}

func TestScaleSkill_ClampsToValidRange(t *testing.T) {
	if v := scaleSkill(10, 2.0); v != 20 {
		t.Fatalf("expected scaling 10 by 2.0 to give 20, got %d", v)
	}
	if v := scaleSkill(99, 2.0); v != 99 {
		t.Fatalf("expected scaling up to clamp at 99, got %d", v)
	}
	if v := scaleSkill(5, -1.0); v != 0 {
		t.Fatalf("expected a negative multiplier to clamp at 0, got %d", v)
	}
}

func TestMinutesPlayedSoFar_SumsAcrossPositions(t *testing.T) {
	total := minutesPlayedSoFar(map[int]int{common.DEFENSE: 20, common.MIDFIELD: 15})
	if total != 35 {
		t.Fatalf("expected minutes summed across positions to be 35, got %d", total)
	}
}

func TestMinutesPlayedSoFar_NilMapIsZero(t *testing.T) {
	if total := minutesPlayedSoFar(nil); total != 0 {
		t.Fatalf("expected a nil map (player hasn't played yet) to give 0 minutes, got %d", total)
	}
}

func TestApplyFatigue_LowStaminaPlayerWithManyMinutesHasReducedSkills(t *testing.T) {
	players := map[int64]common.Player{
		1: {Id: 1, Stamina: 0, Shot: 80, Defense: 80, Lateral: 80, Pass: 80, Midfield: 80, Dribbling: 80, GoalKeeper: 80},
	}
	minutesByPosition := map[int64]map[int]int{1: {common.STRIKER: FatigueMatchLength}}

	effective := applyFatigue(&players, minutesByPosition, common.MENTALITY_BALANCED)
	p := (*effective)[1]
	if p.Shot >= 80 {
		t.Fatalf("expected a fully fatigued player's Shot to drop below 80, got %d", p.Shot)
	}
	if p.Id != 1 {
		t.Fatalf("expected non-skill fields (Id) to be preserved, got %d", p.Id)
	}
}

func TestApplyFatigue_PlayerWhoHasNotPlayedIsUnaffected(t *testing.T) {
	players := map[int64]common.Player{
		1: {Id: 1, Stamina: 0, Shot: 80},
	}
	effective := applyFatigue(&players, map[int64]map[int]int{}, common.MENTALITY_BALANCED)
	if p := (*effective)[1]; p.Shot != 80 {
		t.Fatalf("expected a player with no recorded minutes to keep their raw Shot value, got %d", p.Shot)
	}
}
