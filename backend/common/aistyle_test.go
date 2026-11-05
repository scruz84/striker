package common

import (
	"database/sql"
	"math/rand"
	"testing"
)

// cpu-team-management "AI level after a division change".
func TestStepAiLevelToward(t *testing.T) {
	cases := []struct{ current, drawn, want int }{
		{AiLevelCaretaker, AiLevelDirector, AiLevelManager},
		{AiLevelDirector, AiLevelCaretaker, AiLevelManager},
		{AiLevelManager, AiLevelManager, AiLevelManager},
		{AiLevelManager, AiLevelDirector, AiLevelDirector},
	}
	for _, c := range cases {
		if got := StepAiLevelToward(c.current, c.drawn); got != c.want {
			t.Errorf("from %d towards %d: %d, want %d", c.current, c.drawn, got, c.want)
		}
	}
}

// RandomAiLevelFrom keeps RandomAiLevel's tier weights and is reproducible with a seeded rng.
func TestRandomAiLevelFrom(t *testing.T) {
	counts := map[int]int{}
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 10000; i++ {
		counts[RandomAiLevelFrom(1, rng)]++
	}
	if share := float64(counts[AiLevelDirector]) / 10000; share < 0.57 || share > 0.63 {
		t.Errorf("tier 1 Director share %.3f, want about 0.60", share)
	}
	a, b := rand.New(rand.NewSource(9)), rand.New(rand.NewSource(9))
	for i := 0; i < 20; i++ {
		if RandomAiLevelFrom(2, a) != RandomAiLevelFrom(2, b) {
			t.Fatal("the same seed drew different levels")
		}
	}
}

// cpu-team-management "Club style": "Youth club", "Frugal club", "Ambitious club" and "Stable".
func TestApplyClubStyle(t *testing.T) {
	youth := ApplyClubStyle(AiTransferProfileFor(AiLevelManager), AiStyleYouth)
	if !youth.SignsProjects || !youth.LoansOutYouth || !youth.EstimatesPotential {
		t.Errorf("youth Manager %+v, want it to sign projects and lend youngsters", youth)
	}
	frugal := ApplyClubStyle(AiTransferProfileFor(AiLevelDirector), AiStyleFrugal)
	if !near(frugal.BudgetReserve, 0.30) || !near(frugal.KeyPlayerPriceMultiplier, 2.0) {
		t.Errorf("frugal Director reserve %.2f key x%.2f, want 0.30 and 2.0", frugal.BudgetReserve, frugal.KeyPlayerPriceMultiplier)
	}
	ambitious := ApplyClubStyle(AiTransferProfileFor(AiLevelManager), AiStyleAmbitious)
	if !near(ambitious.BudgetReserve, 0.40) || ambitious.MinScore != 4 || !near(ambitious.KeyPlayerPriceMultiplier, 2.5) {
		t.Errorf("ambitious Manager reserve %.2f min score %.1f key x%.2f, want 0.40, 4 and 2.5", ambitious.BudgetReserve, ambitious.MinScore, ambitious.KeyPlayerPriceMultiplier)
	}
	if got := ApplyClubStyle(AiTransferProfileFor(AiLevelDirector), AiStyleAmbitious); !near(got.BudgetReserve, 0.05) || got.MinScore != 2 {
		t.Errorf("ambitious Director reserve %.2f min score %.1f, want 0.05 and 2", got.BudgetReserve, got.MinScore)
	}
	caretaker := AiTransferProfileFor(AiLevelCaretaker)
	for _, style := range []string{AiStyleYouth, AiStyleFrugal, AiStyleAmbitious} {
		if got := ApplyClubStyle(caretaker, style); got != caretaker {
			t.Errorf("Caretaker profile changed by style %s", style)
		}
	}
	if balanced := ApplyClubStyle(AiTransferProfileFor(AiLevelManager), AiStyleBalanced); balanced != AiTransferProfileFor(AiLevelManager) {
		t.Error("a balanced club's profile changed")
	}
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

// Of every 5 consecutive ids 2 are balanced and 1 each youth, frugal and ambitious; a style
// round-trips through its stored code, and NULL reads as balanced.
func TestAiClubStyleFor(t *testing.T) {
	counts := map[string]int{}
	for id := int64(100); id < 105; id++ {
		counts[AiClubStyleFor(id)]++
		if AiClubStyleFor(id) != AiClubStyleFor(id) {
			t.Fatal("unstable style")
		}
	}
	if counts[AiStyleBalanced] != 2 || counts[AiStyleYouth] != 1 || counts[AiStyleFrugal] != 1 || counts[AiStyleAmbitious] != 1 {
		t.Errorf("styles of 5 consecutive ids %v, want 2 balanced and one of each other", counts)
	}
	for _, style := range []string{AiStyleBalanced, AiStyleYouth, AiStyleFrugal, AiStyleAmbitious} {
		if got := AiStyleFromCode(sql.NullInt64{Int64: int64(AiStyleCode(style)), Valid: true}); got != style {
			t.Errorf("%s round-trips as %s", style, got)
		}
	}
	if got := AiStyleFromCode(sql.NullInt64{}); got != AiStyleBalanced {
		t.Errorf("NULL reads as %s, want balanced", got)
	}
}
