package engine

import (
	"math"

	"github.com/scruz84/striker/common"
)

// FatigueMaxPenalty and FatigueMatchLength are the fatigue formula's constants, shared with the
// CPU lineup planner (see common.FatigueMultiplier).
const FatigueMaxPenalty = common.FatigueMaxPenalty

// FatigueMatchLength must match runMatchSimulation's own endMinute.
const FatigueMatchLength = common.FatigueMatchLength

// StaminaSimulation rebuilds each team's fatigue-adjusted "effective" player snapshot
// (EffectiveHomePlayers/EffectiveAwayPlayers) every simulated minute, from each player's Stamina
// and how many minutes they've played so far this match (HomeMinutesByPosition/
// AwayMinutesByPosition - see fatigueMultiplier). It runs first in the pipeline (see
// runMatchSimulation), before InitialSimulation recomputes TeamQualification and before
// ScorerSimulation reads any individual player's skills, so both read the same fatigued numbers
// for this minute rather than one seeing fresh skills while the other sees tired ones.
type StaminaSimulation struct{}

func (s StaminaSimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	match.EffectiveHomePlayers = applyFatigue(match.HomePlayers, match.HomeMinutesByPosition, match.HomeMentality)
	match.EffectiveAwayPlayers = applyFatigue(match.AwayPlayers, match.AwayMinutesByPosition, match.AwayMentality)
}

// applyFatigue returns a fresh copy of players with their seven core skills scaled down by each
// player's own fatigueMultiplier - every other field (traits, name, age, Stamina/Form
// themselves...) is carried over unchanged, since fatigue only ever diminishes footballing
// output, not who the player is.
func applyFatigue(players *map[int64]common.Player, minutesByPosition map[int64]map[int]int, mentality int) *map[int64]common.Player {
	effective := make(map[int64]common.Player, len(*players))
	maxPenalty := common.FatigueMaxPenaltyFor(mentality)
	for id, p := range *players {
		multiplier := common.FatigueMultiplierWithPenalty(p.Stamina, float64(minutesPlayedSoFar(minutesByPosition[id])), maxPenalty)
		p.GoalKeeper = scaleSkill(p.GoalKeeper, multiplier)
		p.Defense = scaleSkill(p.Defense, multiplier)
		p.Lateral = scaleSkill(p.Lateral, multiplier)
		p.Pass = scaleSkill(p.Pass, multiplier)
		p.Midfield = scaleSkill(p.Midfield, multiplier)
		p.Dribbling = scaleSkill(p.Dribbling, multiplier)
		p.Shot = scaleSkill(p.Shot, multiplier)
		effective[id] = p
	}
	return &effective
}

func minutesPlayedSoFar(byPosition map[int]int) int {
	total := 0
	for _, minutes := range byPosition {
		total += minutes
	}
	return total
}

// fatigueMultiplier is common.FatigueMultiplier - the formula lives in common so the CPU lineup
// planner reads exactly the same numbers as this simulation.
func fatigueMultiplier(stamina int, minutesPlayed int) float64 {
	return common.FatigueMultiplier(stamina, minutesPlayed)
}

func scaleSkill(value int, multiplier float64) int {
	scaled := int(math.Round(float64(value) * multiplier))
	if scaled < 0 {
		return 0
	}
	if scaled > 99 {
		return 99
	}
	return scaled
}
