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
	applyHomeAdvantage(match.HomeQua, match.HomeAdvantage)
	applyMentality(match.HomeQua, match.HomeMentality)
	applyMentality(match.AwayQua, match.AwayMentality)

	calculateDiffs(match)
	//Append a new match instant
	*match.MatchInstants = append(*match.MatchInstants, common.MatchInstant{
		MatchId:              match.Id,
		Minute:               match.Minute,
		AddedMinute:          match.AddedMinute,
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

// HomeAdvantageBase is the home side's edge with nobody in the stands, and
// HomeAdvantageCrowdWeight what a full stadium adds on top: the home zone ratings are multiplied by
// 1 + HomeAdvantageBase + HomeAdvantageCrowdWeight x fill (see homeAdvantageMultiplier), so 1.04
// in an empty ground, 1.075 half full and 1.11 full. Calibrated so that, between identical squads at
// a half-full ground, the home side wins about 41.5% and the away side about 33.5% (see
// TestHomeAdvantageCalibration).
const HomeAdvantageBase = 0.04
const HomeAdvantageCrowdWeight = 0.07

// homeAdvantageMultiplier is the home side's zone-rating multiplier for a match watched by
// attendance in a stadium of capacity seats. fill is attendance / capacity, clamped to [0, 1], and
// 0 when the capacity is 0.
func homeAdvantageMultiplier(attendance int, capacity int) float32 {
	fill := 0.0
	if capacity > 0 {
		fill = math.Max(0, math.Min(1, float64(attendance)/float64(capacity)))
	}
	return float32(1 + HomeAdvantageBase + HomeAdvantageCrowdWeight*fill)
}

// homeAdvantageFor is the home side's multiplier for a match with this pre-kickoff crowd: 1 for a
// neutral fixture (a World Cup group friendly or knockout tie), otherwise homeAdvantageMultiplier
// of the crowd's attendance and capacity.
func homeAdvantageFor(crowd matchdayCrowd, neutral bool) float32 {
	if neutral {
		return 1
	}
	return homeAdvantageMultiplier(crowd.Attendance, crowd.Capacity)
}

// applyHomeAdvantage multiplies the nine zone ratings of a side (attack, midfield and defense;
// centre and both flanks) by multiplier, right after calculateQua's zone blend and before the
// diffs, so chance volume, chance share and conversion all see it. Goalkeeper is never populated
// (see common.TeamQualification) and is left alone. A multiplier of 0 (unset) or 1 changes nothing.
func applyHomeAdvantage(qua *common.TeamQualification, multiplier float32) {
	if multiplier <= 0 || multiplier == 1 {
		return
	}
	qua.LeftDefense *= multiplier
	qua.Defense *= multiplier
	qua.RightDefense *= multiplier
	qua.LeftMidfield *= multiplier
	qua.Midfield *= multiplier
	qua.RightMidfield *= multiplier
	qua.LeftAttack *= multiplier
	qua.Attack *= multiplier
	qua.RightAttach *= multiplier
}

func calculateDiffs(match *common.MatchSimulation) {
	match.MidfieldDiff = calculateMidfieldDiff(match)
	match.HomeAttackCenterDiff = calculateHomeAttackDiff(match)
	match.AwayAttackCenterDiff = calculateAwayAttackDiff(match)
}

// Line weights of the zone model (openspec match-simulation "Team qualification recomputed every
// minute"). Together they make a formation trade strength between its lines instead of gaining it
// outright, and make a formation pay off when the squad has the players it fields more of - the
// "Formation balance" and "Squad fit" requirements, checked by the TACTICS_SIM calibration in
// tactics_study_test.go. Retune them only against that calibration.
const (
	// LineCoverWeight is how much of himself each central player lends to each adjacent central
	// line: a centre-back to the midfield, a central midfielder to the defense and the attack, a
	// central striker to the midfield. Kept small: more cover drags specialists down in the lines
	// they do not play (a pure centre-back's poor Midfield would sink a back five's midfield).
	LineCoverWeight = 0.1
	// LateralStrikerCentralWeight is the share of a lateral striker (a 4-3-3's wide forward) that
	// also counts toward the central attack. It used to be 1, which made 4-3-3 dominate every other
	// shape: its nine zones added up to 700 where every other formation's add up to 600.
	LateralStrikerCentralWeight = 0.5
	// LateralMidfieldCentralWeight is the share of a lateral midfielder that tucks into the central
	// midfield.
	LateralMidfieldCentralWeight = 0.65
	// LateralMidfieldFlankShare is the lateral midfielder's share of his flank's midfield rating;
	// the full-back behind him supplies the rest.
	LateralMidfieldFlankShare = 0.5
	// BackThreeWingBackCentralWeight is the share of his Defense each full-back of a back three
	// (one DEFENSE slot) adds to the central defense, as a wing-back tucks in.
	BackThreeWingBackCentralWeight = 0.15
	// MidfieldPassWeight is Pass's share of a player's midfield value (the rest is Midfield).
	MidfieldPassWeight = 0.3
	// AttackDribblingWeight is Dribbling's share of a player's attacking value (the rest is Shot).
	AttackDribblingWeight = 0.25
	// CrowdedBackLineWeight is how much of the central defense's per-player value each flank
	// defense gains per extra half-player of central defensive weight over two (see
	// crowdedBackLineFlankBonus): 1.5 x ZoneSpillover, tuned so a back five with good centre-backs
	// pays off without making 5-3-2 the natural choice for most squads.
	CrowdedBackLineWeight = 0.225
)

// midfieldValue is what a player brings to a midfield zone: 0.7 x Midfield + 0.3 x Pass.
func midfieldValue(p common.Player) float32 {
	return float32(p.Midfield)*(1-MidfieldPassWeight) + float32(p.Pass)*MidfieldPassWeight
}

// attackValue is what a player brings to an attacking zone: 0.75 x Shot + 0.25 x Dribbling.
func attackValue(p common.Player) float32 {
	return float32(p.Shot)*(1-AttackDribblingWeight) + float32(p.Dribbling)*AttackDribblingWeight
}

// centralLine accumulates a central line's weighted total and the sum of its contributors'
// weights: every player in the line itself weighs 1, cover and tucked-in wide players less.
type centralLine struct {
	total, weight float32
}

func (l *centralLine) add(value float32, weight float32) {
	l.total += value * weight
	l.weight += weight
}

// average is the line's per-player value, its weighted total over its weight sum; 0 for an empty
// line (every central player sent off).
func (l centralLine) average() float32 {
	if l.weight <= 0 {
		return 0
	}
	return l.total / l.weight
}

// centralLines builds the three central lines from whoever is on the pitch (see the "Core
// ratings", "Line cover", "Wide forwards and wide midfielders" and "Back-three wing-backs"
// scenarios). Shared by calculateQua and playerContributions, so a player's own contribution is
// exactly what the zone total counts for him.
func centralLines(byPos map[int][]int64, players *map[int64]common.Player) (defense, midfield, attack centralLine) {
	pl := func(id int64) common.Player { return (*players)[id] }
	for _, id := range byPos[common.DEFENSE] {
		defense.add(float32(pl(id).Defense), 1)
		midfield.add(midfieldValue(pl(id)), LineCoverWeight)
	}
	for _, id := range byPos[common.MIDFIELD] {
		midfield.add(midfieldValue(pl(id)), 1)
		defense.add(float32(pl(id).Defense), LineCoverWeight)
		attack.add(attackValue(pl(id)), LineCoverWeight)
	}
	for _, id := range byPos[common.STRIKER] {
		attack.add(attackValue(pl(id)), 1)
		midfield.add(midfieldValue(pl(id)), LineCoverWeight)
	}
	for _, pos := range []int{common.LATERAL_STRIKER_LEFT, common.LATERAL_STRIKER_RIGHT} {
		for _, id := range byPos[pos] {
			attack.add(attackValue(pl(id)), LateralStrikerCentralWeight)
		}
	}
	for _, pos := range []int{common.LATERAL_MIDFIELD_LEFT, common.LATERAL_MIDFIELD_RIGHT} {
		for _, id := range byPos[pos] {
			midfield.add(midfieldValue(pl(id)), LateralMidfieldCentralWeight)
		}
	}
	if len(byPos[common.DEFENSE]) == backThreeCentralDefenderCount {
		for _, pos := range []int{common.LATERAL_DEFENSE_LEFT, common.LATERAL_DEFENSE_RIGHT} {
			for _, id := range byPos[pos] {
				defense.add(float32(pl(id).Defense), BackThreeWingBackCentralWeight)
			}
		}
	}
	return defense, midfield, attack
}

func calculateQua(teamQua *common.TeamQualification, lineup *common.Lineup, playersByPosition *map[int][]int64,
	players *map[int64]common.Player) {
	byPos := *playersByPosition
	centralDefenderCount := len(byPos[common.DEFENSE])
	defense, midfield, attack := centralLines(byPos, players)

	//defense
	rawLeftDefense := calculateLateralDefense(byPos[common.LATERAL_DEFENSE_LEFT], byPos[common.LATERAL_MIDFIELD_LEFT], players, centralDefenderCount)
	rawRightDefense := calculateLateralDefense(byPos[common.LATERAL_DEFENSE_RIGHT], byPos[common.LATERAL_MIDFIELD_RIGHT], players, centralDefenderCount)
	teamQua.LeftDefense, teamQua.Defense, teamQua.RightDefense = blendLine(rawLeftDefense, defense.total, defense.weight, rawRightDefense)
	crowded := crowdedBackLineFlankBonus(defense)
	teamQua.LeftDefense += crowded
	teamQua.RightDefense += crowded

	//midfield
	rawLeftMidfield := calculateLateralMidfield(byPos[common.LATERAL_DEFENSE_LEFT], byPos[common.LATERAL_MIDFIELD_LEFT], players, centralDefenderCount)
	rawRightMidfield := calculateLateralMidfield(byPos[common.LATERAL_DEFENSE_RIGHT], byPos[common.LATERAL_MIDFIELD_RIGHT], players, centralDefenderCount)
	teamQua.LeftMidfield, teamQua.Midfield, teamQua.RightMidfield = blendLine(rawLeftMidfield, midfield.total, midfield.weight, rawRightMidfield)

	//attack
	rawLeftAttack := calculateLateralAttack(byPos[common.LATERAL_STRIKER_LEFT], byPos[common.LATERAL_MIDFIELD_LEFT], players)
	rawRightAttack := calculateLateralAttack(byPos[common.LATERAL_STRIKER_RIGHT], byPos[common.LATERAL_MIDFIELD_RIGHT], players)
	teamQua.LeftAttack, teamQua.Attack, teamQua.RightAttach = blendLine(rawLeftAttack, attack.total, attack.weight, rawRightAttack)
}

// ZoneSpillover is how much of a three-zone line's (defense, midfield, or attack) strength leaks
// into its neighboring zone, the same way a back four or midfield three actually covers for each
// other rather than each zone playing in total isolation - a busy left flank still draws a bit of
// support from the center, and vice versa. Applied identically to all three lines by blendLine.
const ZoneSpillover = 0.15

// blendLine gives each zone of a left/center/right line a share of its neighbors' *per-player
// average* strength, not their raw magnitude: a flank is always one player, but the center sums
// every contributor weighted by his share of the line (see centralLines), so comparing the raw
// total with a flank would swing a flank's rating from the formation's shape alone. The center is
// averaged over centerWeight (its contributors' summed weights) first and the blended center is
// then scaled back up by centerWeight, so a fuller center's greater total presence is otherwise
// unaffected. centerWeight <= 0 (every central player sent off) skips blending entirely.
func blendLine(left float32, centerTotal float32, centerWeight float32, right float32) (blendedLeft float32, blendedCenter float32, blendedRight float32) {
	if centerWeight <= 0 {
		return left, 0, right
	}
	centerAvg := centerTotal / centerWeight
	blendedLeft = left*(1-ZoneSpillover) + centerAvg*ZoneSpillover
	blendedRight = right*(1-ZoneSpillover) + centerAvg*ZoneSpillover
	blendedCenterAvg := centerAvg*(1-2*ZoneSpillover) + left*ZoneSpillover + right*ZoneSpillover
	blendedCenter = blendedCenterAvg * centerWeight
	return blendedLeft, blendedCenter, blendedRight
}

// crowdedBackLineFlankBonus is what each flank defense gains from how crowded the central defense
// is (match-simulation "Crowded back line"): CrowdedBackLineWeight x the line's per-player average
// x (W / 2 - 1) for a central weight sum W. Near zero for a back four, positive for a back five and
// negative for a back three - blendLine's own spillover uses the per-player average, so without
// this a third centre-back would never reach the flanks.
func crowdedBackLineFlankBonus(defense centralLine) float32 {
	if defense.weight <= 0 {
		return 0
	}
	return CrowdedBackLineWeight * defense.average() * (defense.weight/2 - 1)
}

// backThreeCentralDefenderCount is how many DEFENSE-slot players a back-three formation fields
// (FormationAttacking's 3-4-3 and FormationThreeFiveTwo's 3-5-2 both set DEFENSE to 1, see
// formation.go) - with only one true centre-back, the two lateral defenders are wing-backs who
// tuck in to cover centrally rather than a back-four's fullbacks, who already have two centre-backs
// behind them and can commit further forward. calculateLateralDefense/calculateLateralMidfield
// and centralLines use this to detect that shape structurally, from the DEFENSE slot's own player
// count, instead of threading the formation ID through.
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

// calculateLateralMidfield is a flank's midfield rating (match-simulation "Flank midfield"): the
// full-back's flank-midfield value alone, or, when the flank also has a lateral midfielder, the
// LateralMidfieldFlankShare blend of the two. 0 when both were sent off.
func calculateLateralMidfield(def []int64, mid []int64, players *map[int64]common.Player, centralDefenderCount int) float32 {
	switch {
	case len(def) == 0 && len(mid) == 0:
		return 0
	case len(mid) == 0:
		return lateralMidfieldValue((*players)[def[0]], centralDefenderCount)
	case len(def) == 0:
		return wideMidfielderFlankValue((*players)[mid[0]]) * LateralMidfieldFlankShare
	}
	return lateralMidfieldValue((*players)[def[0]], centralDefenderCount)*(1-LateralMidfieldFlankShare) +
		wideMidfielderFlankValue((*players)[mid[0]])*LateralMidfieldFlankShare
}

// lateralMidfieldValue is one full-back's flank-midfield value (see calculateLateralMidfield).
func lateralMidfieldValue(p common.Player, centralDefenderCount int) float32 {
	lateralWeight, midfieldWeight := float32(0.5), float32(0.5)
	if centralDefenderCount == backThreeCentralDefenderCount {
		lateralWeight, midfieldWeight = 0.65, 0.35
	}
	return float32(p.Lateral)*lateralWeight + float32(p.Midfield)*midfieldWeight
}

// wideMidfielderFlankValue is a lateral midfielder's own flank-midfield value: 0.5 x Lateral +
// 0.5 x his midfield value.
func wideMidfielderFlankValue(p common.Player) float32 {
	return float32(p.Lateral)*0.5 + midfieldValue(p)*0.5
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

// lateralAttackValue is one wide attacker's flank-attack value: 0.5 x Lateral + 0.5 x his attacking
// value (see calculateLateralAttack).
func lateralAttackValue(p common.Player) float32 {
	return float32(p.Lateral)*0.5 + attackValue(p)*0.5
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
// they feed (left/right indexes 0/1), from the same value formulas calculateQua uses: a full-back
// feeds his flank's defense and midfield, a lateral midfielder his flank's midfield and (when the
// formation has no lateral striker) its attack, and every striker, lateral ones included, the
// central attack. Partial cover (LineCoverWeight, a tucked-in wide midfielder or wing-back) adds to
// the zone totals and so to the rival's per-player averages below, but is not a duel of its own.
type sideDuelists struct {
	goalkeepers    []zoneDuelist
	centralDefense []zoneDuelist
	centralMid     []zoneDuelist
	centralAttack  []zoneDuelist
	flankDefense   [2]*zoneDuelist
	flankMid       [2][]zoneDuelist
	flankAttack    [2]*zoneDuelist
	// defenseAvg, midfieldAvg and attackAvg are the central lines' per-player values (see
	// centralLines), what a rival duelist is compared against.
	defenseAvg, midfieldAvg, attackAvg float32
	// flankMidfield is each flank's midfield rating before blending (calculateLateralMidfield).
	flankMidfield [2]float32
}

func newSideDuelists(playersByPosition *map[int][]int64, players *map[int64]common.Player) sideDuelists {
	byPos := *playersByPosition
	d := sideDuelists{}
	centralDefenderCount := len(byPos[common.DEFENSE])
	defense, midfield, attack := centralLines(byPos, players)
	d.defenseAvg, d.midfieldAvg, d.attackAvg = defense.average(), midfield.average(), attack.average()
	for _, id := range byPos[common.GOALKEEPER] {
		d.goalkeepers = append(d.goalkeepers, zoneDuelist{id, float32((*players)[id].GoalKeeper)})
	}
	for _, id := range byPos[common.DEFENSE] {
		d.centralDefense = append(d.centralDefense, zoneDuelist{id, float32((*players)[id].Defense)})
	}
	for _, id := range byPos[common.MIDFIELD] {
		d.centralMid = append(d.centralMid, zoneDuelist{id, midfieldValue((*players)[id])})
	}
	for _, pos := range []int{common.STRIKER, common.LATERAL_STRIKER_LEFT, common.LATERAL_STRIKER_RIGHT} {
		for _, id := range byPos[pos] {
			d.centralAttack = append(d.centralAttack, zoneDuelist{id, attackValue((*players)[id])})
		}
	}
	flanks := [2]struct{ defense, midfield, striker int }{
		{common.LATERAL_DEFENSE_LEFT, common.LATERAL_MIDFIELD_LEFT, common.LATERAL_STRIKER_LEFT},
		{common.LATERAL_DEFENSE_RIGHT, common.LATERAL_MIDFIELD_RIGHT, common.LATERAL_STRIKER_RIGHT},
	}
	for side, flank := range flanks {
		d.flankMidfield[side] = calculateLateralMidfield(byPos[flank.defense], byPos[flank.midfield], players, centralDefenderCount)
		if def := byPos[flank.defense]; len(def) > 0 {
			p := (*players)[def[0]]
			d.flankDefense[side] = &zoneDuelist{def[0], lateralDefenseValue(p, centralDefenderCount)}
			d.flankMid[side] = append(d.flankMid[side], zoneDuelist{def[0], lateralMidfieldValue(p, centralDefenderCount)})
		}
		if mid := byPos[flank.midfield]; len(mid) > 0 {
			d.flankMid[side] = append(d.flankMid[side], zoneDuelist{mid[0], wideMidfielderFlankValue((*players)[mid[0]])})
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
		add(&own.centralDefense[i], rival.attackAvg)
	}
	for i := range own.centralMid {
		add(&own.centralMid[i], rival.midfieldAvg)
	}
	for i := range own.centralAttack {
		add(&own.centralAttack[i], rival.defenseAvg)
	}
	for side := 0; side < 2; side++ {
		mirrored := 1 - side
		add(own.flankDefense[side], duelistValue(rival.flankAttack[mirrored]))
		for i := range own.flankMid[side] {
			add(&own.flankMid[side][i], rival.flankMidfield[mirrored])
		}
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

// applyMentality shifts a side's ratings for this minute by its mentality (see common/mentality.go):
// an Attacking side's three attack ratings go up by common.MentalityAttackFactor and its three
// defense ratings down by common.MentalityDefenseFactor, a Defensive side's the other way round.
// Midfield, the goalkeeper and a Balanced side are untouched, so two Balanced sides play exactly as
// before mentality existed.
func applyMentality(qua *common.TeamQualification, mentality int) {
	if qua == nil || mentality == common.MENTALITY_BALANCED {
		return
	}
	attack := float32(1 + common.MentalityAttackFactor*float64(mentality))
	defense := float32(1 - common.MentalityDefenseFactor*float64(mentality))
	qua.Attack *= attack
	qua.LeftAttack *= attack
	qua.RightAttach *= attack
	qua.Defense *= defense
	qua.LeftDefense *= defense
	qua.RightDefense *= defense
}
