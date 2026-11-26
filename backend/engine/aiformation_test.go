package engine

import (
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "Director formation against the opponent": the pick is the formation with
// the highest projected goal difference, and over varied squads it often differs from the
// strongest-by-value shape (the rank a Manager plays) - "Exploiting a weak flank".
func TestChooseFormationAgainst_PicksTheBestExpectedGoalDifference(t *testing.T) {
	differs := 0
	for seed := int64(0); seed < 30; seed++ {
		own := calibrationSquad(rand.New(rand.NewSource(seed)), 1)
		other := calibrationSquad(rand.New(rand.NewSource(seed+500)), 101)
		opponent := planCalibrationSide(other, common.AiLevelManager, 1)
		side := ZoneDuelSide{Squad: other, Lineup: opponent.lineup(2, 0)}

		picked := chooseFormationAgainst(own, side, 1, true)
		bestDiff := -1e9
		for _, f := range common.AllFormations {
			lineup := &common.Lineup{Formation: f, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(own, f))}
			xg := ProjectExpectedGoals(ZoneDuelSide{Squad: own, Lineup: lineup}, side, 1, true)
			if d := xg.Own - xg.Opponent; d > bestDiff {
				bestDiff = d
			}
		}
		lineup := &common.Lineup{Formation: picked, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(own, picked))}
		xg := ProjectExpectedGoals(ZoneDuelSide{Squad: own, Lineup: lineup}, side, 1, true)
		if got := xg.Own - xg.Opponent; got < bestDiff-1e-9 {
			t.Errorf("seed %d: picked formation %d expects %.3f, another expects %.3f", seed, picked, got, bestDiff)
		}
		if strongest, _ := common.SelectBestFormationAndLineup(own, common.AiLevelManager, nil); strongest != picked {
			differs++
		}
	}
	if differs == 0 {
		t.Error("the expected-goals pick never differed from the strongest-by-value formation over 30 squads")
	}
}

// openspec cpu-team-management "Lineup and formation": the engine's registered scorer makes a
// Manager play the formation whose best XI projects the highest goal difference against the same
// squad's best 4-4-2 XI.
func TestManagerFormation_IsTheBestProjectedAgainstItsOwnFourFourTwo(t *testing.T) {
	rng := rand.New(rand.NewSource(12))
	for trial := 0; trial < 20; trial++ {
		squad := make([]*common.Player, 0, 18)
		r := func() int { return 30 + rng.Intn(31) }
		for id := int64(1); id <= 18; id++ {
			squad = append(squad, &common.Player{Id: id, GoalKeeper: r(), Defense: r(), Lateral: r(), Pass: r(), Midfield: r(), Dribbling: r(), Shot: r()})
		}
		reference := &common.Lineup{Formation: common.FormationBalanced, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(squad, common.FormationBalanced))}
		best, bestDiff := -1, -1e9
		for _, f := range common.AllFormations {
			lineup := &common.Lineup{Formation: f, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(squad, f))}
			xg := ProjectExpectedGoals(ZoneDuelSide{Lineup: lineup, Squad: squad}, ZoneDuelSide{Lineup: reference, Squad: squad}, 1, true)
			if d := xg.Own - xg.Opponent; d > bestDiff+1e-9 {
				best, bestDiff = f, d
			}
		}
		if got, _ := common.SelectBestFormationAndLineup(squad, common.AiLevelManager, rand.New(rand.NewSource(1))); got != best {
			t.Errorf("trial %d: Manager plays %d, want %d (best projected difference %.3f)", trial, got, best, bestDiff)
		}
	}
}
