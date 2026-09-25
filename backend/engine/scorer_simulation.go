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
	totalMidfield := match.HomeQua.Midfield + match.AwayQua.Midfield
	if totalMidfield <= 0 {
		return ProbChance
	}
	dominance := (match.HomeQua.Midfield - match.AwayQua.Midfield) / totalMidfield
	if dominance < 0 {
		dominance = -dominance
	}
	return ProbChance + dominance*ProbChanceDominanceBonus
}

func processChance(s ScorerSimulation, dbCtx *common.DatabaseContext, match *common.MatchSimulation, teamChance int64) {
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
		players = match.HomePlayers
		attackPlayers = match.HomePlayersByPosition
		otherTeam = match.Away
		goalKeeperValue = (*match.AwayPlayers)[(*match.AwayPlayersByPosition)[common.GOALKEEPER][0]].GoalKeeper
		if isRight {
			attack = match.HomeQua.RightAttach
			defense = match.AwayQua.LeftDefense
		} else if isLeft {
			attack = match.HomeQua.LeftAttack
			defense = match.AwayQua.RightDefense
		} else {
			attack = match.HomeQua.Attack
			defense = match.AwayQua.Defense
		}
	} else {
		players = match.AwayPlayers
		attackPlayers = match.AwayPlayersByPosition
		otherTeam = match.Home
		goalKeeperValue = (*match.HomePlayers)[(*match.HomePlayersByPosition)[common.GOALKEEPER][0]].GoalKeeper
		if isRight {
			attack = match.AwayQua.RightAttach
			defense = match.HomeQua.LeftDefense
		} else if isLeft {
			attack = match.AwayQua.LeftAttack
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
