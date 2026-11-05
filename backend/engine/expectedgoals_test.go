package engine

import (
	"math"
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

// Identical XIs on a neutral ground expect the same goals, about the engine's 1.2 open-play goals
// per side per match; a stronger XI expects a better goal difference (midfield dominance raises
// the chance rate for both sides, so its opponent may expect slightly more too).
func TestProjectExpectedGoals(t *testing.T) {
	home, away := evenSquad(1), evenSquad(101)
	side := func(squad []*common.Player, teamId int64) ZoneDuelSide {
		return ZoneDuelSide{Squad: squad, Lineup: lineupFor(teamId, common.FormationBalanced, common.SelectBestLineupForFormation(squad, common.FormationBalanced))}
	}
	even := ProjectExpectedGoals(side(home, 1), side(away, 2), 1, true)
	if math.Abs(even.Own-even.Opponent) > 1e-6 || even.Own < 0.8 || even.Own > 1.8 {
		t.Errorf("even sides expect %.3f vs %.3f, want equal and about 1.2", even.Own, even.Opponent)
	}
	for _, p := range home {
		p.Shot, p.Midfield, p.Pass = p.Shot+15, p.Midfield+15, p.Pass+15
	}
	strong := ProjectExpectedGoals(side(home, 1), side(away, 2), 1, true)
	if strong.Own <= even.Own || strong.Own-strong.Opponent <= 0.2 {
		t.Errorf("a stronger XI expects %.3f vs %.3f, want clearly more than the even %.3f", strong.Own, strong.Opponent, even.Own)
	}
}

// The projection tracks the simulation: over seeded matches between an even XI and a stronger
// one, the side with the higher expected goal difference wins more points.
func TestProjectExpectedGoals_AgreesWithTheSimulation(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	agree, total := 0, 0
	for i := 0; i < 40; i++ {
		a := calibrationSquad(rand.New(rand.NewSource(int64(i))), 1)
		b := calibrationSquad(rand.New(rand.NewSource(int64(i)+1000)), 101)
		sa := calibrationSide{squad: a, plan: common.PlanAiMatchLineup(a, nil, nil, common.AiLevelDirector, false, rand.New(rand.NewSource(1)))}
		sb := calibrationSide{squad: b, plan: common.PlanAiMatchLineup(b, nil, nil, common.AiLevelDirector, false, rand.New(rand.NewSource(1)))}
		xg := ProjectExpectedGoals(ZoneDuelSide{Squad: a, Lineup: sa.lineup(1, 0)}, ZoneDuelSide{Squad: b, Lineup: sb.lineup(2, 0)}, 1, true)
		if math.Abs(xg.Own-xg.Opponent) < 0.25 {
			continue
		}
		diff := 0
		for m := 0; m < 60; m++ {
			hg, ag := playCalibrationMatch(rng, sa, common.AiLevelCaretaker, sb, common.AiLevelCaretaker)
			diff += hg - ag
		}
		total++
		if (diff > 0) == (xg.Own > xg.Opponent) {
			agree++
		}
	}
	if total < 10 || float64(agree) < 0.8*float64(total) {
		t.Errorf("the projection picked the better side in %d of %d clear matchups, want at least 80%%", agree, total)
	}
}
