package engine

import (
	"github.com/scruz84/striker/common"
	"math"
	"math/rand"
	"testing"
)

func TestGetPlayerForChance_ReturnsSentinelWhenNoOutfieldPlayersLeft(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	positions := map[int][]int64{
		common.GOALKEEPER: {1},
	}

	if player := getPlayerForChance(r, &positions); player != -1 {
		t.Fatalf("expected -1 when striker/midfield/defense are all empty, got %d", player)
	}
}

func TestGetPlayerForChance_FallsBackToDefenseWhenAttackIsEmpty(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	positions := map[int][]int64{
		common.DEFENSE: {42},
	}

	for i := 0; i < 20; i++ {
		if player := getPlayerForChance(r, &positions); player != 42 {
			t.Fatalf("expected the only remaining defender (42), got %d", player)
		}
	}
}

func TestChanceSuccessProbability_EvenMatchupReturnsBaseline(t *testing.T) {
	got := chanceSuccessProbability(100, 100)
	if math.Abs(float64(got-ConversionBaseline)) > 0.001 {
		t.Fatalf("expected an even matchup to land on the baseline %.2f, got %.4f", ConversionBaseline, got)
	}
}

func TestChanceSuccessProbability_BoundedForExtremeMismatch(t *testing.T) {
	dominant := chanceSuccessProbability(100000, 1)
	if dominant > ConversionMax {
		t.Fatalf("expected a dominant attack to be capped at %.2f, got %.4f", ConversionMax, dominant)
	}
	overwhelmed := chanceSuccessProbability(1, 100000)
	if overwhelmed < ConversionMin {
		t.Fatalf("expected an overwhelmed attack to be floored at %.2f, got %.4f", ConversionMin, overwhelmed)
	}
}

func TestChanceSuccessProbability_StrongerAttackScoresMoreOften(t *testing.T) {
	weak := chanceSuccessProbability(40, 60)
	strong := chanceSuccessProbability(60, 40)
	if !(weak < ConversionBaseline && strong > ConversionBaseline && weak < strong) {
		t.Fatalf("expected weak < baseline < strong, got weak=%.4f baseline=%.2f strong=%.4f", weak, ConversionBaseline, strong)
	}
}

func TestHomeChanceShare_EqualMidfieldIsFiftyFifty(t *testing.T) {
	match := &common.MatchSimulation{HomeQua: &common.TeamQualification{Midfield: 50}, AwayQua: &common.TeamQualification{Midfield: 50}}
	if got := homeChanceShare(match); math.Abs(float64(got-0.5)) > 0.001 {
		t.Fatalf("expected 0.5 for equal midfields, got %.4f", got)
	}
}

func TestHomeChanceShare_BoundedForExtremeDominance(t *testing.T) {
	match := &common.MatchSimulation{HomeQua: &common.TeamQualification{Midfield: 100}, AwayQua: &common.TeamQualification{Midfield: 0.0001}}
	if got := homeChanceShare(match); got > PossessionMaxShare {
		t.Fatalf("expected home share to be capped at %.2f, got %.4f", PossessionMaxShare, got)
	}
}

func TestFlankMidfieldSupport_EvenMidfieldBattleIsNeutral(t *testing.T) {
	if got := flankMidfieldSupport(50, 50); got != 0 {
		t.Fatalf("expected an even flank midfield battle to add nothing, got %v", got)
	}
}

func TestFlankMidfieldSupport_WinningTheFlankAddsAModestBoost(t *testing.T) {
	got := flankMidfieldSupport(70, 50)
	want := float32(FlankMidfieldSupportWeight) * 20
	if math.Abs(float64(got-want)) > 0.001 {
		t.Fatalf("expected a %v-weighted boost of %v, got %v", FlankMidfieldSupportWeight, want, got)
	}
	if got <= 0 {
		t.Fatalf("expected winning the flank's midfield battle to boost attack, got %v", got)
	}
}

func TestFlankMidfieldSupport_LosingTheFlankIsAPenalty(t *testing.T) {
	if got := flankMidfieldSupport(40, 60); got >= 0 {
		t.Fatalf("expected losing the flank's midfield battle to penalize attack, got %v", got)
	}
}

func TestChanceQualityMultiplier_OnlyEverReturnsTheTwoTiers(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		got := chanceQualityMultiplier(r, 60, 40)
		if got != ClearCutQualityMultiplier && got != HalfChanceQualityMultiplier {
			t.Fatalf("expected either the clear-cut or half-chance multiplier, got %.2f", got)
		}
	}
}
