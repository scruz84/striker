package engine

import (
	"github.com/scruz84/striker/common"
	"math"
	"math/rand"
	"time"
)

const ProbChance = 0.14
const ProbChanceDominanceBonus = 0.10 // extra per-minute chance probability at maximum midfield dominance
const ProbCenterChance = 0.50
const ProbLateralChance = 0.25

// Per-chance conversion is a logistic curve rather than a plain attack/(attack+defense) ratio,
// so it's centered on a realistic baseline and saturates instead of scaling linearly with the
// stat gap - a real shot conversion rate rarely leaves roughly this 5%-65% band even for a huge
// mismatch, whereas the old ratio could approach 0% or 100% for a large enough gap.
const ConversionBaseline = 0.28 // average real-world conversion rate for a team of even strength
const ConversionMin = 0.05
const ConversionMax = 0.65
const ConversionSteepness = 2.0

// conversionLogitOffset shifts the logistic so that an even matchup (diff == 0) lands exactly on
// ConversionBaseline instead of the curve's natural midpoint between ConversionMin/ConversionMax.
var conversionLogitOffset = math.Log((float64(ConversionBaseline) - float64(ConversionMin)) / (float64(ConversionMax) - float64(ConversionBaseline)))

// Possession/chance share is likewise bounded: real-world dominance rarely buys a team much more
// than roughly two-thirds of the chances in a match, however lopsided the midfield battle is.
const PossessionMaxShare = 0.68
const PossessionSteepness = 1.4

// Most attacking chances are half-chances rather than clean looks at goal; only a minority are
// clear-cut. Modeling that separately from the base conversion rate keeps outcomes swingy for the
// right reason (which specific chance a team gets) instead of every chance sharing identical odds.
const ProbClearCutChance = 0.20
const ProbClearCutEdgeBonus = 0.15 // extra clear-cut chance probability at maximum attack-vs-defense edge
const ClearCutQualityMultiplier = float32(1.6)
const HalfChanceQualityMultiplier = float32(0.7)

// Set pieces are carved out of the chances the simulator already generates (a share of each side's
// chances is a penalty or a direct free kick instead of an open-play chance), so the number of
// chances - and with it the goals per match - barely moves, and the side that dominates midfield
// also wins more penalties and free kicks, as in real football. Real-world reference: 0.24-0.41
// penalties per match (~78% long-run conversion), ~0.85 direct free-kick shots per match
// (~5-6% conversion). At ~12.6 chances per even match: 12.6 x 0.022 = 0.28 penalties and
// 12.6 x 0.065 = 0.82 free kicks.
const ProbPenaltyPerChance = 0.022
const ProbDirectFreeKickPerChance = 0.065

// Penalty conversion is a clamped line, not a logistic: real penalty rates sit in a narrow band, so
// 0.78 + 0.30 x (taker Shot - keeper GoalKeeper)/99 bounded to [0.60, 0.92] is easy to tune.
const PenaltyConversionBaseline = 0.78
const PenaltySkillWeight = 0.30
const PenaltyConversionMin = 0.60
const PenaltyConversionMax = 0.92

// Direct free-kick conversion, same shape: 0.06 + 0.10 x (taker value - keeper GoalKeeper)/99
// bounded to [0.02, 0.15], where the taker value blends Shot and Pass.
const FreeKickConversionBaseline = 0.06
const FreeKickSkillWeight = 0.10
const FreeKickConversionMin = 0.02
const FreeKickConversionMax = 0.15
const FreeKickShotWeight = 0.6
const FreeKickPassWeight = 0.4

// ScorerSimulation calculates initial weights based on players.
type ScorerSimulation struct {
	r *rand.Rand
}

func NewScorerSimulation() *ScorerSimulation {
	return &ScorerSimulation{r: rand.New(rand.NewSource(time.Now().UnixMilli()))}
}

func (s ScorerSimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	if rand.Float32() < chanceProbability(match) {
		teamChance := getTeamChance(s, match)
		processChance(s, dbCtx, match, teamChance)
	}
}

// chanceProbability returns the probability of an attacking chance occurring this minute.
// A more lopsided midfield battle raises the odds, so a dominant team generates more chances
// over the course of a match rather than just converting a fixed share of chances more often.
func chanceProbability(match *common.MatchSimulation) float32 {
	// Each Attacking side opens the match up by common.MentalityChanceShift and each Defensive side
	// closes it down by as much (an Attacking side against a Defensive one cancels out).
	shift := float32(common.MentalityChanceShift * float64(match.HomeMentality+match.AwayMentality))
	totalMidfield := match.HomeQua.Midfield + match.AwayQua.Midfield
	if totalMidfield <= 0 {
		return ProbChance + shift
	}
	dominance := (match.HomeQua.Midfield - match.AwayQua.Midfield) / totalMidfield
	if dominance < 0 {
		dominance = -dominance
	}
	return ProbChance + dominance*ProbChanceDominanceBonus + shift
}

// FlankMidfieldSupportWeight is how much a flank's midfield battle nudges that same flank's
// attacking value for a left/right chance in processChance - modest support, not a replacement
// for the flank's own attack rating. Crossed the same way attack-vs-defense already is: a team's
// own left flank is physically the same strip of pitch as the opponent's right flank, since the
// two sides attack in opposite directions (see ZoneComparisonRows.tsx's doc comment on the
// frontend side of this), so the attacking side's left-midfield battle is fought against the
// defending side's right-midfield, not their left.
const FlankMidfieldSupportWeight = 0.2

// flankMidfieldSupport is the crossed midfield edge (attacking side's own flank vs the defending
// side's mirrored flank) that processChance adds into a left/right chance's attack value.
func flankMidfieldSupport(attackerFlankMidfield float32, defenderCrossedFlankMidfield float32) float32 {
	return FlankMidfieldSupportWeight * (attackerFlankMidfield - defenderCrossedFlankMidfield)
}

func processChance(s ScorerSimulation, dbCtx *common.DatabaseContext, match *common.MatchSimulation, teamChance int64) {
	if kind := classifyChance(s.r); kind != common.SET_PIECE_NONE {
		processSetPiece(s, dbCtx, match, teamChance, kind)
		return
	}
	isRight := false
	isLeft := false
	//decide if is right, left or center chance
	randValue := rand.Float32()
	if randValue > ProbCenterChance {
		if randValue < (ProbLateralChance + ProbCenterChance) {
			isRight = true
		} else {
			isLeft = true
		}
	}
	var otherTeam int64
	//get the attack and defense values
	var attack float32
	var defense float32
	var goalKeeperValue int
	var attackPlayers *map[int][]int64
	var players *map[int64]common.Player
	if teamChance == match.Home {
		players = match.EffectiveHomePlayers
		attackPlayers = match.HomePlayersByPosition
		otherTeam = match.Away
		goalKeeperValue = (*match.EffectiveAwayPlayers)[(*match.AwayPlayersByPosition)[common.GOALKEEPER][0]].GoalKeeper
		if isRight {
			attack = match.HomeQua.RightAttach + flankMidfieldSupport(match.HomeQua.RightMidfield, match.AwayQua.LeftMidfield)
			defense = match.AwayQua.LeftDefense
		} else if isLeft {
			attack = match.HomeQua.LeftAttack + flankMidfieldSupport(match.HomeQua.LeftMidfield, match.AwayQua.RightMidfield)
			defense = match.AwayQua.RightDefense
		} else {
			attack = match.HomeQua.Attack
			defense = match.AwayQua.Defense
		}
	} else {
		players = match.EffectiveAwayPlayers
		attackPlayers = match.AwayPlayersByPosition
		otherTeam = match.Home
		goalKeeperValue = (*match.EffectiveHomePlayers)[(*match.HomePlayersByPosition)[common.GOALKEEPER][0]].GoalKeeper
		if isRight {
			attack = match.AwayQua.RightAttach + flankMidfieldSupport(match.AwayQua.RightMidfield, match.HomeQua.LeftMidfield)
			defense = match.HomeQua.LeftDefense
		} else if isLeft {
			attack = match.AwayQua.LeftAttack + flankMidfieldSupport(match.AwayQua.LeftMidfield, match.HomeQua.RightMidfield)
			defense = match.HomeQua.RightDefense
		} else {
			attack = match.AwayQua.Attack
			defense = match.HomeQua.Defense
		}
	}
	//probability of success chance
	attackPlayer := getPlayerForChance(s.r, attackPlayers)
	if attackPlayer == -1 {
		// no eligible outfield player left on this team (heavily reduced by red cards)
		return
	}
	attackPlayerValue := getPlayerScorerValue(isLeft || isRight, (*players)[attackPlayer])
	attackValue := attack + attackPlayerValue
	defenseValue := defense + float32(goalKeeperValue)
	quality := chanceQualityMultiplier(s.r, attackValue, defenseValue)
	probSuccess := chanceSuccessProbability(attackValue*quality, defenseValue)
	//calculates if the chance was a success!
	successfullyChance := s.r.Float32() < probSuccess

	var eventType int
	if successfullyChance {
		eventType = common.GOAL
		if teamChance == match.Home {
			match.HomeGoals++
		} else {
			match.AwayGoals++
		}
	} else {
		eventType = common.NO_GOAL
	}

	attackPlayerName, _ := GetPlayerFullName(dbCtx, attackPlayer)

	*match.Events = append(*match.Events, common.Event{
		EventType:             eventType,
		Minute:                match.Minute,
		TeamOwner:             teamChance,
		TeamOwnerPlayers:      []int64{attackPlayer},
		TeamOwnerPlayersNames: []string{attackPlayerName},
		OtherTeam:             otherTeam,
		OtherTeamPlayers:      nil,
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
	})
}

// classifyChance decides whether a chance is a penalty, a direct free kick or open play, with one
// uniform roll: [0, ProbPenaltyPerChance) penalty, the next ProbDirectFreeKickPerChance direct
// free kick, the rest open play.
func classifyChance(r *rand.Rand) int {
	v := r.Float32()
	if v < ProbPenaltyPerChance {
		return common.SET_PIECE_PENALTY
	}
	if v < ProbPenaltyPerChance+ProbDirectFreeKickPerChance {
		return common.SET_PIECE_DIRECT_FREE_KICK
	}
	return common.SET_PIECE_NONE
}

// bestOnPitchOutfielder returns the on-pitch outfield player with the highest score (ties go to
// the lowest player id), or -1 when no outfield player is left. The goalkeeper is excluded; players
// sent off, injured off or substituted are no longer in byPosition, so they're skipped by
// construction.
func bestOnPitchOutfielder(players *map[int64]common.Player, byPosition *map[int][]int64, score func(common.Player) float32) int64 {
	best := int64(-1)
	var bestScore float32
	for position, ids := range *byPosition {
		if position == common.GOALKEEPER {
			continue
		}
		for _, id := range ids {
			v := score((*players)[id])
			if best == -1 || v > bestScore || (v == bestScore && id < best) {
				best = id
				bestScore = v
			}
		}
	}
	return best
}

func clampProbability(v float32, min float32, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// penaltyConversionProbability is the chance a penalty is scored given the taker's Shot and the
// defending goalkeeper's GoalKeeper skill.
func penaltyConversionProbability(shot int, goalKeeper int) float32 {
	p := PenaltyConversionBaseline + PenaltySkillWeight*float32(shot-goalKeeper)/99
	return clampProbability(p, PenaltyConversionMin, PenaltyConversionMax)
}

// freeKickTakerValue is how good a player is from a direct free kick: mostly Shot, some Pass.
func freeKickTakerValue(player common.Player) float32 {
	return FreeKickShotWeight*float32(player.Shot) + FreeKickPassWeight*float32(player.Pass)
}

// freeKickConversionProbability is the chance a direct free kick is scored given the taker's
// freeKickTakerValue and the defending goalkeeper's GoalKeeper skill.
func freeKickConversionProbability(value float32, goalKeeper int) float32 {
	p := FreeKickConversionBaseline + FreeKickSkillWeight*(value-float32(goalKeeper))/99
	return clampProbability(p, FreeKickConversionMin, FreeKickConversionMax)
}

// processSetPiece resolves a penalty or direct free kick for the side that won the chance. The
// taker is picked from the on-pitch effective (fatigue-adjusted) players, the outcome is a normal
// GOAL / NO_GOAL event tagged with the set-piece kind and naming the defending goalkeeper.
func processSetPiece(s ScorerSimulation, dbCtx *common.DatabaseContext, match *common.MatchSimulation, teamChance int64, kind int) {
	var players *map[int64]common.Player
	var byPosition *map[int][]int64
	var otherTeam int64
	var goalKeeperId int64
	var goalKeeper common.Player
	if teamChance == match.Home {
		players = match.EffectiveHomePlayers
		byPosition = match.HomePlayersByPosition
		otherTeam = match.Away
		goalKeeperId = (*match.AwayPlayersByPosition)[common.GOALKEEPER][0]
		goalKeeper = (*match.EffectiveAwayPlayers)[goalKeeperId]
	} else {
		players = match.EffectiveAwayPlayers
		byPosition = match.AwayPlayersByPosition
		otherTeam = match.Home
		goalKeeperId = (*match.HomePlayersByPosition)[common.GOALKEEPER][0]
		goalKeeper = (*match.EffectiveHomePlayers)[goalKeeperId]
	}

	var taker int64
	var probSuccess float32
	if kind == common.SET_PIECE_PENALTY {
		taker = bestOnPitchOutfielder(players, byPosition, func(p common.Player) float32 { return float32(p.Shot) })
		if taker == -1 {
			return
		}
		probSuccess = penaltyConversionProbability((*players)[taker].Shot, goalKeeper.GoalKeeper)
	} else {
		taker = bestOnPitchOutfielder(players, byPosition, freeKickTakerValue)
		if taker == -1 {
			return
		}
		probSuccess = freeKickConversionProbability(freeKickTakerValue((*players)[taker]), goalKeeper.GoalKeeper)
	}

	eventType := common.NO_GOAL
	if s.r.Float32() < probSuccess {
		eventType = common.GOAL
		if teamChance == match.Home {
			match.HomeGoals++
		} else {
			match.AwayGoals++
		}
	}

	takerName, _ := GetPlayerFullName(dbCtx, taker)
	goalKeeperName, _ := GetPlayerFullName(dbCtx, goalKeeperId)

	*match.Events = append(*match.Events, common.Event{
		EventType:             eventType,
		Minute:                match.Minute,
		TeamOwner:             teamChance,
		TeamOwnerPlayers:      []int64{taker},
		TeamOwnerPlayersNames: []string{takerName},
		OtherTeam:             otherTeam,
		OtherTeamPlayers:      []int64{goalKeeperId},
		OtherTeamPlayersNames: []string{goalKeeperName},
		HomeGoals:             match.HomeGoals,
		AwayGoals:             match.AwayGoals,
		SetPiece:              kind,
	})
}

func getTeamChance(s ScorerSimulation, match *common.MatchSimulation) int64 {
	if s.r.Float32() < homeChanceShare(match) {
		return match.Home
	}
	return match.Away
}

// homeChanceShare returns home's share of attacking chances, bounded within
// [1-PossessionMaxShare, PossessionMaxShare] via tanh instead of a raw proportional split -
// a team with almost all the midfield strength still shouldn't get almost all the chances.
func homeChanceShare(match *common.MatchSimulation) float32 {
	total := match.HomeQua.Midfield + match.AwayQua.Midfield
	if total <= 0 {
		return 0.5
	}
	diff := match.HomeQua.Midfield - match.AwayQua.Midfield
	t := float32(math.Tanh(float64(PossessionSteepness * diff / total)))
	return 0.5 + (PossessionMaxShare-0.5)*t
}

// chanceSuccessProbability converts an attack/defense stat gap into a goal probability using a
// logistic curve bounded to [ConversionMin, ConversionMax] and centered on ConversionBaseline for
// an even matchup, instead of a plain ratio that can approach 0% or 100% for a large stat gap.
func chanceSuccessProbability(attackValue float32, defenseValue float32) float32 {
	scale := (attackValue + defenseValue) / 2
	if scale <= 0 {
		return ConversionBaseline
	}
	diff := attackValue - defenseValue
	x := float64(ConversionSteepness*diff/scale) + conversionLogitOffset
	sigmoid := 1 / (1 + math.Exp(-x))
	return ConversionMin + (ConversionMax-ConversionMin)*float32(sigmoid)
}

// chanceQualityMultiplier models that most chances are half-chances and only a minority are
// clear-cut, with the clear-cut odds nudged (not dominated) by which side holds the attack/defense
// edge for this particular chance.
func chanceQualityMultiplier(r *rand.Rand, attackValue float32, defenseValue float32) float32 {
	total := attackValue + defenseValue
	edge := float32(0)
	if total > 0 {
		edge = (attackValue - defenseValue) / total
	}
	clearCutProb := ProbClearCutChance + edge*ProbClearCutEdgeBonus
	if clearCutProb < 0.05 {
		clearCutProb = 0.05
	}
	if clearCutProb > 0.5 {
		clearCutProb = 0.5
	}
	if r.Float32() < clearCutProb {
		return ClearCutQualityMultiplier
	}
	return HalfChanceQualityMultiplier
}

func getPlayerForChance(r *rand.Rand, playersByPosition *map[int][]int64) int64 {
	//70% striker
	//20% midfield
	//10% defense
	v := r.Float32()
	var playersToSelect []int64
	if v <= 0.70 {
		playersToSelect = (*playersByPosition)[common.STRIKER]
	} else if v <= 0.90 {
		playersToSelect = (*playersByPosition)[common.MIDFIELD]
	} else {
		playersToSelect = (*playersByPosition)[common.DEFENSE]
	}
	//Next fallbacks are for cases when there are no more players on the positions
	//(e.g. a red card emptied that group - see DisciplineSimulation).
	if len(playersToSelect) == 0 {
		playersToSelect = (*playersByPosition)[common.STRIKER]
	}
	if len(playersToSelect) == 0 {
		playersToSelect = (*playersByPosition)[common.MIDFIELD]
	}
	if len(playersToSelect) == 0 {
		playersToSelect = (*playersByPosition)[common.DEFENSE]
	}
	if len(playersToSelect) == 0 {
		return -1
	}
	return playersToSelect[r.Int()%len(playersToSelect)]
}

func getPlayerScorerValue(isLateral bool, player common.Player) float32 {
	if isLateral {
		return float32(player.Shot)*0.5 + float32(player.Lateral)*0.2 + float32(player.Pass)*0.15 + float32(player.Dribbling)*0.15
	}
	return float32(player.Shot)*0.7 + float32(player.Pass)*0.15 + float32(player.Dribbling)*0.15
}
