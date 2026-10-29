package engine

import (
	"math"

	"github.com/scruz84/striker/common"
)

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
		PlayerContributions:  playerContributions(match),
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
	return lateralDefenseValue((*players)[def[0]], centralDefenderCount)
}

// lateralDefenseValue is one full-back's flank-defense value - calculateLateralDefense's per-player
// figure, shared with playerContributions so a player's own contribution is exactly what the zone
// total counts for him.
func lateralDefenseValue(p common.Player, centralDefenderCount int) float32 {
	lateralWeight, defenseWeight := float32(0.5), float32(0.5)
	if centralDefenderCount == backThreeCentralDefenderCount {
		lateralWeight, defenseWeight = 0.35, 0.65
	}
	return float32(p.Lateral)*lateralWeight + float32(p.Defense)*defenseWeight
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
	return lateralMidfieldValue((*players)[def[0]], centralDefenderCount)
}

// lateralMidfieldValue is one full-back's flank-midfield value (see calculateLateralMidfield).
func lateralMidfieldValue(p common.Player, centralDefenderCount int) float32 {
	lateralWeight, midfieldWeight := float32(0.5), float32(0.5)
	if centralDefenderCount == backThreeCentralDefenderCount {
		lateralWeight, midfieldWeight = 0.65, 0.35
	}
	return float32(p.Lateral)*lateralWeight + float32(p.Midfield)*midfieldWeight
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
	return lateralAttackValue((*players)[wide[0]])
}

// lateralAttackValue is one wide attacker's flank-attack value (see calculateLateralAttack).
func lateralAttackValue(p common.Player) float32 {
	return float32(p.Lateral)*0.5 + float32(p.Shot)*0.5
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

// zoneDuelist is one player's raw value in one zone duel - the figure calculateQua adds into that
// zone for him, before blendLine's spillover.
type zoneDuelist struct {
	playerId int64
	value    float32
}

// sideDuelists is one side's per-player zone values for a single minute, split by the zone duels
// they feed (left/right indexes 0/1). It mirrors calculateQua exactly: a full-back feeds both his
// flank's defense and its midfield, a wide attacker is the lateral striker if the formation has one
// and otherwise the lateral midfielder, and every striker (lateral ones included) feeds the
// central attack with his Shot.
type sideDuelists struct {
	goalkeepers    []zoneDuelist
	centralDefense []zoneDuelist
	centralMid     []zoneDuelist
	centralAttack  []zoneDuelist
	flankDefense   [2]*zoneDuelist
	flankMid       [2]*zoneDuelist
	flankAttack    [2]*zoneDuelist
}

func newSideDuelists(playersByPosition *map[int][]int64, players *map[int64]common.Player) sideDuelists {
	byPos := *playersByPosition
	d := sideDuelists{}
	centralDefenderCount := len(byPos[common.DEFENSE])
	for _, id := range byPos[common.GOALKEEPER] {
		d.goalkeepers = append(d.goalkeepers, zoneDuelist{id, float32((*players)[id].GoalKeeper)})
	}
	for _, id := range byPos[common.DEFENSE] {
		d.centralDefense = append(d.centralDefense, zoneDuelist{id, float32((*players)[id].Defense)})
	}
	for _, id := range byPos[common.MIDFIELD] {
		d.centralMid = append(d.centralMid, zoneDuelist{id, float32((*players)[id].Midfield)})
	}
	for _, pos := range []int{common.STRIKER, common.LATERAL_STRIKER_LEFT, common.LATERAL_STRIKER_RIGHT} {
		for _, id := range byPos[pos] {
			d.centralAttack = append(d.centralAttack, zoneDuelist{id, float32((*players)[id].Shot)})
		}
	}
	flanks := [2]struct{ defense, midfield, striker int }{
		{common.LATERAL_DEFENSE_LEFT, common.LATERAL_MIDFIELD_LEFT, common.LATERAL_STRIKER_LEFT},
		{common.LATERAL_DEFENSE_RIGHT, common.LATERAL_MIDFIELD_RIGHT, common.LATERAL_STRIKER_RIGHT},
	}
	for side, flank := range flanks {
		if def := byPos[flank.defense]; len(def) > 0 {
			p := (*players)[def[0]]
			d.flankDefense[side] = &zoneDuelist{def[0], lateralDefenseValue(p, centralDefenderCount)}
			d.flankMid[side] = &zoneDuelist{def[0], lateralMidfieldValue(p, centralDefenderCount)}
		}
		wide := byPos[flank.striker]
		if len(wide) == 0 {
			wide = byPos[flank.midfield]
		}
		if len(wide) > 0 {
			d.flankAttack[side] = &zoneDuelist{wide[0], lateralAttackValue((*players)[wide[0]])}
		}
	}
	return d
}

// perPlayerAverage is a central line's per-player value - its raw total over its head count, 0
// for an empty line (everyone sent off).
func perPlayerAverage(line []zoneDuelist) float32 {
	if len(line) == 0 {
		return 0
	}
	total := float32(0)
	for _, d := range line {
		total += d.value
	}
	return total / float32(len(line))
}

func duelistValue(d *zoneDuelist) float32 {
	if d == nil {
		return 0
	}
	return d.value
}

// contributionAccumulator sums the duels one player feeds this minute, so a player in two duels
// (a full-back, a lateral striker) ends up with the mean of each figure.
type contributionAccumulator struct {
	position   int
	own, rival float32
	duels      int
}

func (a *contributionAccumulator) add(own float32, rival float32) {
	a.own += own
	a.rival += rival
	a.duels++
}

// playerContributions is this minute's zone contribution of every player on the pitch, both sides
// (see common.PlayerContribution): his own value in each zone duel his position feeds against the
// rival's per-player value in the mirrored duel - own left flank against the rival's right, the
// same crossed pairing processChance and the zone duel panel use. Read from the same
// *PlayersByPosition and fatigue-adjusted Effective*Players calculateQua used for this minute.
// A player who feeds no duel (a lateral midfielder in a formation that also fields a lateral
// striker on his flank) is still listed, with 0 against 0, so his minutes count.
func playerContributions(match *common.MatchSimulation) *common.InstantContributions {
	home := newSideDuelists(match.HomePlayersByPosition, match.EffectiveHomePlayers)
	away := newSideDuelists(match.AwayPlayersByPosition, match.EffectiveAwayPlayers)
	return &common.InstantContributions{
		Home: sideContributions(match.HomePlayersByPosition, home, away),
		Away: sideContributions(match.AwayPlayersByPosition, away, home),
	}
}

func sideContributions(playersByPosition *map[int][]int64, own sideDuelists, rival sideDuelists) map[int64]common.PlayerContribution {
	contributions := make(map[int64]common.PlayerContribution)
	acc := make(map[int64]*contributionAccumulator)
	for position, ids := range *playersByPosition {
		for _, id := range ids {
			acc[id] = &contributionAccumulator{position: position}
		}
	}
	add := func(d *zoneDuelist, rivalValue float32) {
		if d == nil {
			return
		}
		if a, ok := acc[d.playerId]; ok {
			a.add(d.value, rivalValue)
		}
	}

	rivalAttackers := make([]zoneDuelist, 0, len(rival.centralAttack)+2)
	rivalAttackers = append(rivalAttackers, rival.centralAttack...)
	for _, flank := range rival.flankAttack {
		if flank != nil {
			rivalAttackers = append(rivalAttackers, *flank)
		}
	}
	rivalAttackerAverage := perPlayerAverage(rivalAttackers)
	for i := range own.goalkeepers {
		add(&own.goalkeepers[i], rivalAttackerAverage)
	}
	for i := range own.centralDefense {
		add(&own.centralDefense[i], perPlayerAverage(rival.centralAttack))
	}
	for i := range own.centralMid {
		add(&own.centralMid[i], perPlayerAverage(rival.centralMid))
	}
	for i := range own.centralAttack {
		add(&own.centralAttack[i], perPlayerAverage(rival.centralDefense))
	}
	for side := 0; side < 2; side++ {
		mirrored := 1 - side
		add(own.flankDefense[side], duelistValue(rival.flankAttack[mirrored]))
		add(own.flankMid[side], duelistValue(rival.flankMid[mirrored]))
		add(own.flankAttack[side], duelistValue(rival.flankDefense[mirrored]))
	}

	for id, a := range acc {
		c := common.PlayerContribution{Position: a.position}
		if a.duels > 0 {
			c.Own = roundTenth(a.own / float32(a.duels))
			c.Rival = roundTenth(a.rival / float32(a.duels))
		}
		contributions[id] = c
	}
	return contributions
}

func roundTenth(v float32) float32 {
	return float32(math.Round(float64(v)*10) / 10)
}
