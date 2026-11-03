package engine

import (
	"github.com/scruz84/striker/common"
)

// Rivalry points a league match or cup tie earns its pair (see openspec rivalries "Rivalry
// events"); the human bonus and the cap are applied by common.AddRivalryEvent.
const (
	rivalryTightResultPoints = 1.0 // a draw or a one-goal margin
	rivalryThrashingPoints   = 1.5 // a margin of 3 or more
	rivalryThrashingMargin   = 3
	rivalryRedCardPoints     = 1.0 // per red card shown
	rivalryMaxRedCardPoints  = 2.0
	rivalryCupEliminationPts = 3.0 // a cup tie always eliminates one side
	rivalryCupFinalPoints    = 5.0 // instead of the elimination points, for a final
)

// rivalryPointsForMatch is the points a match earns its pair before the human bonus: a final decided
// on penalties counts its draw and the final.
func rivalryPointsForMatch(homeGoals int, awayGoals int, redCards int, cupTie bool, cupFinal bool) float64 {
	points := 0.0
	margin := homeGoals - awayGoals
	if margin < 0 {
		margin = -margin
	}
	switch {
	case margin <= 1:
		points += rivalryTightResultPoints
	case margin >= rivalryThrashingMargin:
		points += rivalryThrashingPoints
	}
	reds := float64(redCards) * rivalryRedCardPoints
	if reds > rivalryMaxRedCardPoints {
		reds = rivalryMaxRedCardPoints
	}
	points += reds
	if cupTie {
		if cupFinal {
			points += rivalryCupFinalPoints
		} else {
			points += rivalryCupEliminationPts
		}
	}
	return points
}

// applyRivalryForMatch adds a played league match's or cup tie's rivalry points to its pair, once,
// with the rest of the post-match effects - after the replay, so a level it raises is only
// announced once the match has been watched. A default result (nobody played) earns nothing.
func applyRivalryForMatch(dbCtx *common.DatabaseContext, kind string, payload *common.PostMatchPayload) {
	if payload.SkipSkillGrowth || (kind != common.SimulationKindLeague && kind != common.SimulationKindCup) {
		return
	}
	redCards := 0
	for _, e := range payload.Events {
		if e.EventType == common.RED_CARD {
			redCards++
		}
	}
	points := rivalryPointsForMatch(payload.HomeGoals, payload.AwayGoals, redCards, kind == common.SimulationKindCup, payload.CupFinal)
	common.AddRivalryEvent(dbCtx, payload.Home, payload.Away, points)
}
