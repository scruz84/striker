package engine

import (
	"math"

	"github.com/scruz84/striker/common"
)

// ExpectedGoals is both sides' expected open-play goals over 90 minutes as the simulation would
// produce them from the minute-1 zone ratings of ProjectZoneDuel: chance rate from midfield
// dominance (chanceProbability), each side's share (homeChanceShare), and per chance the
// center/left/right split, the 70/20/10 striker/midfielder/defender shooter mix and the
// clear-cut/half-chance mixture of chanceSuccessProbability - the engine's own formulas, without
// the randomness. Fatigue, mentality, cards, injuries and set pieces are left out; they move both
// sides alike or are a fixed share of chances, so they rarely change which of two XIs is better.
type ExpectedGoals struct {
	Own      float64
	Opponent float64
}

// ProjectExpectedGoals rates two XIs (see ProjectZoneDuel for the sides and home advantage). Pure.
func ProjectExpectedGoals(own ZoneDuelSide, opponent ZoneDuelSide, homeAdvantage float32, ownIsHome bool) ExpectedGoals {
	duel := ProjectZoneDuel(own, opponent, homeAdvantage, ownIsHome)
	ownPlayers := playersToMapById(&own.Squad)
	opponentPlayers := playersToMapById(&opponent.Squad)

	totalMidfield := duel.Own.Midfield + duel.Opponent.Midfield
	chance, ownShare := float64(ProbChance), 0.5
	if totalMidfield > 0 {
		dominance := math.Abs(float64((duel.Own.Midfield - duel.Opponent.Midfield) / totalMidfield))
		chance += dominance * ProbChanceDominanceBonus
		t := math.Tanh(float64(PossessionSteepness * (duel.Own.Midfield - duel.Opponent.Midfield) / totalMidfield))
		ownShare = 0.5 + (PossessionMaxShare-0.5)*t
	}
	openPlay := 1 - ProbPenaltyPerChance - ProbDirectFreeKickPerChance
	perMinute := chance * openPlay * 90
	return ExpectedGoals{
		Own:      perMinute * ownShare * expectedConversion(duel.Own, duel.Opponent, duel.OpponentGoalkeeper, own.Lineup, ownPlayers),
		Opponent: perMinute * (1 - ownShare) * expectedConversion(duel.Opponent, duel.Own, duel.OwnGoalkeeper, opponent.Lineup, opponentPlayers),
	}
}

// expectedConversion is the average chance of a goal from one open-play chance of attacker
// against defender (see processChance).
func expectedConversion(attacker common.TeamQualification, defender common.TeamQualification, keeper float32,
	lineup *common.Lineup, players *map[int64]common.Player) float64 {
	shooters := shooterMix(lineup)
	if len(shooters) == 0 {
		return 0
	}
	zones := []struct {
		weight          float64
		attack, defense float32
		lateral         bool
	}{
		{ProbCenterChance, attacker.Attack, defender.Defense, false},
		{ProbLateralChance, attacker.RightAttach + flankMidfieldSupport(attacker.RightMidfield, defender.LeftMidfield), defender.LeftDefense, true},
		{1 - ProbCenterChance - ProbLateralChance, attacker.LeftAttack + flankMidfieldSupport(attacker.LeftMidfield, defender.RightMidfield), defender.RightDefense, true},
	}
	total := 0.0
	for _, z := range zones {
		for _, s := range shooters {
			attackValue := z.attack + getPlayerScorerValue(z.lateral, (*players)[s.id])
			defenseValue := z.defense + keeper
			edge := float32(0)
			if sum := attackValue + defenseValue; sum > 0 {
				edge = (attackValue - defenseValue) / sum
			}
			clearCut := float64(clampProbability(ProbClearCutChance+edge*ProbClearCutEdgeBonus, 0.05, 0.5))
			p := clearCut*float64(chanceSuccessProbability(attackValue*ClearCutQualityMultiplier, defenseValue)) +
				(1-clearCut)*float64(chanceSuccessProbability(attackValue*HalfChanceQualityMultiplier, defenseValue))
			total += z.weight * s.weight * p
		}
	}
	return total
}

type weightedShooter struct {
	id     int64
	weight float64
}

// shooterMix is who takes an open-play chance and how often (see getPlayerForChance): 70% a
// striker, 20% a midfielder, 10% a defender, each line's share spread evenly over its players and
// an empty line's share falling back to the striker, midfield, then defense lines.
func shooterMix(lineup *common.Lineup) []weightedShooter {
	byLine := map[int][]int64{}
	for _, p := range lineup.Positions {
		if p.Position == common.STRIKER || p.Position == common.MIDFIELD || p.Position == common.DEFENSE {
			byLine[p.Position] = append(byLine[p.Position], p.PlayerId)
		}
	}
	fallback := func() int {
		for _, line := range []int{common.STRIKER, common.MIDFIELD, common.DEFENSE} {
			if len(byLine[line]) > 0 {
				return line
			}
		}
		return -1
	}
	shares := map[int]float64{}
	for line, share := range map[int]float64{common.STRIKER: 0.70, common.MIDFIELD: 0.20, common.DEFENSE: 0.10} {
		if len(byLine[line]) > 0 {
			shares[line] += share
		} else if f := fallback(); f >= 0 {
			shares[f] += share
		}
	}
	mix := make([]weightedShooter, 0, len(lineup.Positions))
	for line, share := range shares {
		for _, id := range byLine[line] {
			mix = append(mix, weightedShooter{id: id, weight: share / float64(len(byLine[line]))})
		}
	}
	return mix
}
