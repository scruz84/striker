package engine

import "github.com/scruz84/striker/common"

// init registers the engine's formation score with common.SelectBestFormationAndLineup (openspec
// cpu-team-management "Lineup and formation"): the projected expected goal difference of a
// formation's best XI against the same squad's best 4-4-2 XI on neutral ground.
func init() {
	common.SetFormationScorer(projectedFormationScore)
}

// projectedFormationScore is the common.FormationScorer the engine registers. Pure.
func projectedFormationScore(squad []*common.Player, lineup *common.Lineup, reference *common.Lineup) float64 {
	xg := ProjectExpectedGoals(ZoneDuelSide{Lineup: lineup, Squad: squad}, ZoneDuelSide{Lineup: reference, Squad: squad}, 1, true)
	return xg.Own - xg.Opponent
}

// chooseFormationAgainst is the formation a Director picks for a competitive match once it knows
// its opponent's XI (see openspec/specs/cpu-team-management "Director formation against the
// opponent"): for each formation, the best XI it can field from available (see
// common.SelectBestLineupForFormation) is rated against the opponent with ProjectExpectedGoals, and
// the one with the highest expected goal difference wins; ties go to the stronger XI, then to the
// earlier formation in common.AllFormations. Pure.
func chooseFormationAgainst(available []*common.Player, opponent ZoneDuelSide, homeAdvantage float32, ownIsHome bool) int {
	byId := make(map[int64]*common.Player, len(available))
	for _, p := range available {
		byId[p.Id] = p
	}
	best, bestDiff, bestStrength := common.FormationBalanced, 0.0, -1
	for _, formation := range common.AllFormations {
		lineup := &common.Lineup{TeamId: -1, Formation: formation, Positions: common.FlattenLineup(common.SelectBestLineupForFormation(available, formation))}
		if len(lineup.Positions) == 0 {
			continue
		}
		xg := ProjectExpectedGoals(ZoneDuelSide{Lineup: lineup, Squad: available}, opponent, homeAdvantage, ownIsHome)
		diff := xg.Own - xg.Opponent
		strength := common.LineupStrength(lineup, byId)
		if bestStrength < 0 || diff > bestDiff+1e-9 || (diff > bestDiff-1e-9 && strength > bestStrength) {
			best, bestDiff, bestStrength = formation, diff, strength
		}
	}
	return best
}
