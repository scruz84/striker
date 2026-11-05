package engine

import "github.com/scruz84/striker/common"

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
