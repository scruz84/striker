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
	calculateQua(match.HomeQua, match.HomeLineup, match.HomePlayersByPosition, match.EffectiveHomePlayers)
	calculateQua(match.AwayQua, match.AwayLineup, match.AwayPlayersByPosition, match.EffectiveAwayPlayers)

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
	centralDefenders := (*playersByPosition)[common.DEFENSE]
	centralDefenderCount := len(centralDefenders)

	//defense
	rawLeftDefense := calculateLateralDefense(
		(*playersByPosition)[common.LATERAL_DEFENSE_LEFT], (*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], players, centralDefenderCount)
	rawRightDefense := calculateLateralDefense(
		(*playersByPosition)[common.LATERAL_DEFENSE_RIGHT], (*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], players, centralDefenderCount)
	rawDefense := calculateDefense(centralDefenders, (*playersByPosition)[common.MIDFIELD], players)
	teamQua.LeftDefense, teamQua.Defense, teamQua.RightDefense =
		blendLine(rawLeftDefense, rawDefense, centralDefenderCount, rawRightDefense)

	//midfield
	rawLeftMidfield := calculateLateralMidfield(
		(*playersByPosition)[common.LATERAL_DEFENSE_LEFT], (*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], players, centralDefenderCount)
	rawRightMidfield := calculateLateralMidfield(
		(*playersByPosition)[common.LATERAL_DEFENSE_RIGHT], (*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], players, centralDefenderCount)
	centralMidfielders := (*playersByPosition)[common.MIDFIELD]
	rawMidfield := calculateMidfield((*playersByPosition)[common.DEFENSE], centralMidfielders, players)
	teamQua.LeftMidfield, teamQua.Midfield, teamQua.RightMidfield =
		blendLine(rawLeftMidfield, rawMidfield, len(centralMidfielders), rawRightMidfield)

	//attack
	rawLeftAttack := calculateLateralAttack(
		(*playersByPosition)[common.LATERAL_STRIKER_LEFT], (*playersByPosition)[common.LATERAL_MIDFIELD_LEFT], players)
	rawRightAttack := calculateLateralAttack(
		(*playersByPosition)[common.LATERAL_STRIKER_RIGHT], (*playersByPosition)[common.LATERAL_MIDFIELD_RIGHT], players)
	// allStrikers folds LATERAL_STRIKER_LEFT/RIGHT into the central attack pool alongside STRIKER.
	// Unlike LATERAL_DEFENSE (a full-back, a different job from a central defender) or
	// LATERAL_MIDFIELD (a winger, a different job from a central midfielder), a 4-3-3's wide
	// forward is still a forward by trade - just stationed wide - so it should count toward the
	// team's overall central scoring presence the same way a 3-4-3's third central striker does,
	// in addition to the dedicated flank rating it already earns above via calculateLateralAttack.
	// Without this, swapping a formation's second central striker for two wide forwards (4-3-3)
	// read as a pure attack downgrade versus 4-4-2 despite fielding an extra nominal striker,
	// since blendLine scales a line's center by how many players occupy it (see its doc comment)
	// and only the lone central STRIKER slot was ever counted.
	strikers := (*playersByPosition)[common.STRIKER]
	allStrikers := make([]int64, 0, len(strikers)+2)
	allStrikers = append(allStrikers, strikers...)
	allStrikers = append(allStrikers, (*playersByPosition)[common.LATERAL_STRIKER_LEFT]...)
	allStrikers = append(allStrikers, (*playersByPosition)[common.LATERAL_STRIKER_RIGHT]...)
	rawAttack := calculateAttack(allStrikers, (*playersByPosition)[common.MIDFIELD], players)
	teamQua.LeftAttack, teamQua.Attack, teamQua.RightAttach =
		blendLine(rawLeftAttack, rawAttack, len(allStrikers), rawRightAttack)
}

// ZoneSpillover is how much of a three-zone line's (defense, midfield, or attack) strength leaks
// into its neighboring zone, the same way a back four or midfield three actually covers for each
// other rather than each zone playing in total isolation - a busy left flank still draws a bit of
// support from the center, and vice versa. Applied identically to all three lines by blendLine.
const ZoneSpillover = 0.15

// blendLine gives each zone of a left/center/right line a share of its neighbors' *per-player
// average* strength, not their raw magnitude: a flank is always exactly one player
// (LATERAL_DEFENSE_LEFT etc. are always 1-wide, see formation.go), but the center can field one
// to three (DEFENSE/MIDFIELD/STRIKER), and calculateDefense/calculateMidfield/calculateAttack sum
// every one of them - so a raw three-player center is 2-3x a raw one-player flank for identical
// per-player quality, and blending them directly would swing a flank's rating from this team's
// own formation shape alone (a back three vs a lone sweeper), not from anyone's actual skill.
// Averaging the center over centerCount first keeps the blend comparable player-for-player; the
// returned center is then scaled back up by centerCount, so a wider center's greater total
// presence (the whole reason the game sums it) is otherwise unaffected - only how much it gives
// to/takes from its flanks changes. centerCount == 0 (every central player sent off) skips
// blending entirely rather than dividing by zero, leaving the flanks as they are.
func blendLine(left float32, centerTotal float32, centerCount int, right float32) (blendedLeft float32, blendedCenter float32, blendedRight float32) {
	if centerCount == 0 {
		return left, 0, right
	}
	centerAvg := centerTotal / float32(centerCount)
	blendedLeft = left*(1-ZoneSpillover) + centerAvg*ZoneSpillover
	blendedRight = right*(1-ZoneSpillover) + centerAvg*ZoneSpillover
	blendedCenterAvg := centerAvg*(1-2*ZoneSpillover) + left*ZoneSpillover + right*ZoneSpillover
	blendedCenter = blendedCenterAvg * float32(centerCount)
	return blendedLeft, blendedCenter, blendedRight
}

// backThreeCentralDefenderCount is how many DEFENSE-slot players a back-three formation fields
// (FormationAttacking's 3-4-3 and FormationThreeFiveTwo's 3-5-2 both set DEFENSE to 1, see
// formation.go) - with only one true centre-back, the two lateral defenders are wing-backs who
// tuck in to cover centrally rather than a back-four's fullbacks, who already have two centre-backs
// behind them and can commit further forward. calculateLateralDefense/calculateLateralMidfield use
// this to detect that shape structurally, from the DEFENSE slot's own player count, instead of
// threading the formation ID through.
const backThreeCentralDefenderCount = 1

func calculateLateralDefense(def []int64, mid []int64, players *map[int64]common.Player, centralDefenderCount int) float32 {
	if len(def) == 0 {
		// the lateral defender was sent off (see DisciplineSimulation) - nobody covers that flank
		return 0
	}
	lateralWeight, defenseWeight := float32(0.5), float32(0.5)
	if centralDefenderCount == backThreeCentralDefenderCount {
		lateralWeight, defenseWeight = 0.35, 0.65
	}
	value := float32((*players)[def[0]].Lateral)*lateralWeight + float32((*players)[def[0]].Defense)*defenseWeight
	return value
}

func calculateDefense(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32(0)
	for _, d := range def {
		value += float32((*players)[d].Defense)
	}
	return value
}

func calculateLateralMidfield(def []int64, mid []int64, players *map[int64]common.Player, centralDefenderCount int) float32 {
	if len(def) == 0 {
		return 0
	}
	lateralWeight, midfieldWeight := float32(0.5), float32(0.5)
	if centralDefenderCount == backThreeCentralDefenderCount {
		lateralWeight, midfieldWeight = 0.65, 0.35
	}
	value := float32((*players)[def[0]].Lateral)*lateralWeight + float32((*players)[def[0]].Midfield)*midfieldWeight
	return value
}

func calculateMidfield(def []int64, mid []int64, players *map[int64]common.Player) float32 {
	value := float32(0)
	for _, d := range mid {
		value += float32((*players)[d].Midfield)
	}
	return value
}

// calculateLateralAttack rates a flank's attacking threat from whichever position actually plays
// there: a 4-3-3's wide forward (LATERAL_STRIKER_LEFT/RIGHT) if the formation has one, otherwise
// the wide midfielder/winger (LATERAL_MIDFIELD_LEFT/RIGHT) that formations like 4-4-2 use instead -
// a formation only ever fields one of the two (see FormationPositionCounts), never both.
func calculateLateralAttack(striker []int64, mid []int64, players *map[int64]common.Player) float32 {
	wide := striker
	if len(wide) == 0 {
		wide = mid
	}
	if len(wide) == 0 {
		return 0
	}
	value := float32((*players)[wide[0]].Lateral)*0.5 + float32((*players)[wide[0]].Shot)*0.5
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
