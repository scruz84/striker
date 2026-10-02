package engine

import (
	"math"

	"github.com/scruz84/striker/common"
)

// FatigueMaxPenalty is the largest fraction a player's effective core skills can be reduced by
// from in-match fatigue alone - reached only by a Stamina-0 player who has played every minute of
// the match so far. A Stamina-99 player is never fatigued at all, whatever the minute (see
// fatigueMultiplier).
const FatigueMaxPenalty = 0.30

// FatigueMatchLength is the match length (in minutes) fatigueMultiplier normalizes elapsed
// playing time against - matches runMatchSimulation's own endMinute.
const FatigueMatchLength = 90

// StaminaSimulation rebuilds each team's fatigue-adjusted "effective" player snapshot
// (EffectiveHomePlayers/EffectiveAwayPlayers) every simulated minute, from each player's Stamina
// and how many minutes they've played so far this match (HomeMinutesByPosition/
// AwayMinutesByPosition - see fatigueMultiplier). It runs first in the pipeline (see
// runMatchSimulation), before InitialSimulation recomputes TeamQualification and before
// ScorerSimulation reads any individual player's skills, so both read the same fatigued numbers
// for this minute rather than one seeing fresh skills while the other sees tired ones.
type StaminaSimulation struct{}

func (s StaminaSimulation) Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	match.EffectiveHomePlayers = applyFatigue(match.HomePlayers, match.HomeMinutesByPosition)
	match.EffectiveAwayPlayers = applyFatigue(match.AwayPlayers, match.AwayMinutesByPosition)
}

// applyFatigue returns a fresh copy of players with their seven core skills scaled down by each
// player's own fatigueMultiplier - every other field (traits, name, age, Stamina/Form
// themselves...) is carried over unchanged, since fatigue only ever diminishes footballing
// output, not who the player is.
func applyFatigue(players *map[int64]common.Player, minutesByPosition map[int64]map[int]int) *map[int64]common.Player {
	effective := make(map[int64]common.Player, len(*players))
	for id, p := range *players {
		multiplier := fatigueMultiplier(p.Stamina, minutesPlayedSoFar(minutesByPosition[id]))
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

// fatigueMultiplier scales down linearly with both elapsed playing time (as a fraction of
// FatigueMatchLength) and how little Stamina (0-99) the player has, capped so extra time logged
// beyond FatigueMatchLength (there isn't any - matches run exactly 90 minutes - but this keeps the
// formula well-defined regardless) never pushes the penalty past FatigueMaxPenalty.
func fatigueMultiplier(stamina int, minutesPlayed int) float64 {
	if minutesPlayed > FatigueMatchLength {
		minutesPlayed = FatigueMatchLength
	}
	elapsedFraction := float64(minutesPlayed) / float64(FatigueMatchLength)
	staminaDeficit := float64(99-stamina) / 99
	return 1 - FatigueMaxPenalty*elapsedFraction*staminaDeficit
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
