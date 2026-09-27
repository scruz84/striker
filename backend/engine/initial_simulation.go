package engine

import "github.com/scruz84/striker/common"

type SimulationPipelineImpl struct {
	simulations []Simulation
}

func (s SimulationPipelineImpl) Simulate(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	for _, sim := range s.simulations {
		sim.Apply(dbCtx, match)
	}
}

////////////////////////////// Simulations

// InitialSimulation calculates initial weights based on players.
type InitialSimulation struct{}

func (s InitialSimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	calculateQua(match.HomeQua, match.HomeLineup, match.HomePlayersByPosition, match.HomePlayers)
	calculateQua(match.AwayQua, match.AwayLineup, match.AwayPlayersByPosition, match.AwayPlayers)

	calculateDiffs(match)
	//Append a new match instant
	*match.MatchInstants = append(*match.MatchInstants, common.MatchInstant{
		MatchId:              match.Id,
		Minute:               match.Minute,
		HomeGoals:            match.HomeGoals,
		AwayGoals:            match.AwayGoals,
		HomeQua:              *match.HomeQua,
		AwayQua:              *match.AwayQua,
		MidfieldDiff:         match.MidfieldDiff,
		HomeAttackRightDiff:  match.HomeAttackRightDiff,
		HomeAttackLeftDiff:   match.HomeAttackLeftDiff,
		HomeAttackCenterDiff: match.HomeAttackCenterDiff,
		AwayAttackRightDiff:  match.AwayAttackRightDiff,
		AwayAttackLeftDiff:   match.AwayAttackLeftDiff,
		AwayAttackCenterDiff: match.AwayAttackCenterDiff,
	})
}

func calculateDiffs(match *common.MatchSimulation) {
	match.MidfieldDiff = calculateMidfieldDiff(match)
	match.HomeAttackCenterDiff = calculateHomeAttackDiff(match)
	match.AwayAttackCenterDiff = calculateAwayAttackDiff(match)
}

func calculateQua(teamQua *common.TeamQualification, lineup *common.Lineup, playersByPosition *map[int][]int64,
	players *map[int64]common.Player) {
	//defense
	rawLeftDefense := calculateLateralDefense(
		(*playersByPosition)[common.LATERAL_DEFENSE_LEFT], (*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], players)
	rawRightDefense := calculateLateralDefense(
		(*playersByPosition)[common.LATERAL_DEFENSE_RIGHT], (*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], players)
	rawDefense := calculateDefense(
		(*playersByPosition)[common.DEFENSE], (*playersByPosition)[common.MIDFIELD], players)
	teamQua.LeftDefense, teamQua.Defense, teamQua.RightDefense = blendLine(rawLeftDefense, rawDefense, rawRightDefense)

	//midfield
	rawLeftMidfield := calculateLateralMidfield(
		(*playersByPosition)[common.LATERAL_DEFENSE_LEFT], (*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], players)
	rawRightMidfield := calculateLateralMidfield(
		(*playersByPosition)[common.LATERAL_DEFENSE_RIGHT], (*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], players)
	rawMidfield := calculateMidfield(
		(*playersByPosition)[common.DEFENSE], (*playersByPosition)[common.MIDFIELD], players)
	teamQua.LeftMidfield, teamQua.Midfield, teamQua.RightMidfield = blendLine(rawLeftMidfield, rawMidfield, rawRightMidfield)

	//attack
	rawLeftAttack := calculateLateralAttack(
		(*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], (*playersByPosition)[common.LATERAL_STRIKER_LEFT], players)
	rawRightAttack := calculateLateralAttack(
		(*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], (*playersByPosition)[common.LATERAL_STRIKER_RIGHT], players)
	rawAttack := calculateAttack(
		(*playersByPosition)[common.STRIKER], (*playersByPosition)[common.MIDFIELD], players)
	teamQua.LeftAttack, teamQua.Attack, teamQua.RightAttach = blendLine(rawLeftAttack, rawAttack, rawRightAttack)
}

// ZoneSpillover is how much of a three-zone line's (defense, midfield, or attack) strength leaks
// into its neighboring zone, the same way a back four or midfield three actually covers for each
// other rather than each zone playing in total isolation - a busy left flank still draws a bit of
// support from the center, and vice versa. Applied identically to all three lines by blendLine.
const ZoneSpillover = 0.15

// blendLine gives each zone of a left/center/right line a share of its neighbors' raw strength:
// the center picks up a slice of both flanks, and each flank picks up a slice of the center (not
// of the opposite flank, which isn't adjacent to it). Mirrors the 0.7/0.15/0.15 weighting this
// package's match-report diffs already used, just applied to the actual TeamQualification values
// the simulation plays out with instead of only a display-only summary.
func blendLine(left float32, center float32, right float32) (blendedLeft float32, blendedCenter float32, blendedRight float32) {
	blendedLeft = left*(1-ZoneSpillover) + center*ZoneSpillover
	blendedRight = right*(1-ZoneSpillover) + center*ZoneSpillover
	blendedCenter = center*(1-2*ZoneSpillover) + left*ZoneSpillover + right*ZoneSpillover
	return blendedLeft, blendedCenter, blendedRight
}

func calculateLateralDefense(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	if len(def) == 0 {
		// the lateral defender was sent off (see DisciplineSimulation) - nobody covers that flank
		return 0
	}
	value := float32((*players)[def[0]].Lateral)*0.5 + float32((*players)[def[0]].Defense)*0.5
	return value
}

func calculateDefense(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32(0)
	for _, d := range def {
		value += float32((*players)[d].Defense)
	}
	return value
}

func calculateLateralMidfield(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	if len(def) == 0 {
		return 0
	}
	value := float32((*players)[def[0]].Lateral)*0.5 + float32((*players)[def[0]].Midfield)*0.5
	return value
}

func calculateMidfield(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32(0)
	for _, d := range def {
		value += float32((*players)[d].Midfield)
	}
	return value
}

func calculateLateralAttack(striker []int64, mid []int64, players *map[int64]common.Player) float32 {
	if len(striker) == 0 {
		return 0
	}
	value := float32((*players)[striker[0]].Lateral)*0.5 + float32((*players)[striker[0]].Shot)*0.5
	return value
}

func calculateAttack(striker []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32(0)
	for _, d := range striker {
		value += float32((*players)[d].Shot)
	}
	return value
}

// calculateMidfieldDiff/calculateHomeAttackDiff/calculateAwayAttackDiff used to each re-blend
// their own 0.7/0.15/0.15 mix of center/left/right on top of already-raw Qua values - now that
// blendLine bakes that same mix directly into Midfield/Attack/Defense themselves (see
// calculateQua), these are plain center-vs-center diffs instead of re-blending it a second time.
func calculateMidfieldDiff(match *common.MatchSimulation) float32 {
	return match.HomeQua.Midfield - match.AwayQua.Midfield
}

func calculateHomeAttackDiff(match *common.MatchSimulation) float32 {
	return match.HomeQua.Attack - match.AwayQua.Defense
}

func calculateAwayAttackDiff(match *common.MatchSimulation) float32 {
	return match.AwayQua.Attack - match.HomeQua.Defense
}
