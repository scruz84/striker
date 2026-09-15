package common

import "math"

// matchRatingBase/Min/Max and the per-event deltas below drive a deliberately simple per-player
// match rating heuristic (issue: Hattrick-style match report) - not a real performance model,
// just enough to make the "Calificaciones" tab feel earned rather than a flat 6.0 for everyone.
const (
	matchRatingBase        = 6.0
	matchRatingMin         = 1.0
	matchRatingMax         = 10.0
	matchRatingGoal        = 1.4
	matchRatingYellowCard  = -0.4
	matchRatingRedCard     = -1.5
	matchRatingInjury      = -0.2
	matchRatingResultBonus = 0.3
)

// ComputeMatchRatings derives a 1-10 rating for every player named in either lineup's Positions,
// adjusted by their own goals/cards/injuries during the match (see Event.TeamOwnerPlayers) and a
// small bonus/penalty for their team's result. matchRatingJitter adds a small, deterministic
// per-player nudge so a lineup doesn't read as suspiciously uniform - the same match always
// produces the same ratings, computed fresh on read rather than stored (see WeekMatchById).
// Returns nil if either lineup or the event list is missing (an unplayed match, or one with no
// lineup data).
func ComputeMatchRatings(match *WeekMatch) *map[int64]float64 {
	if match.HomeLineup == nil || match.AwayLineup == nil || match.Events == nil {
		return nil
	}

	ratings := make(map[int64]float64)
	homePlayers := make(map[int64]bool)
	for _, pos := range match.HomeLineup.Positions {
		homePlayers[pos.PlayerId] = true
		ratings[pos.PlayerId] = matchRatingBase
	}
	for _, pos := range match.AwayLineup.Positions {
		ratings[pos.PlayerId] = matchRatingBase
	}

	for _, event := range *match.Events {
		var delta float64
		switch event.EventType {
		case GOAL:
			delta = matchRatingGoal
		case YELLOW_CARD:
			delta = matchRatingYellowCard
		case RED_CARD:
			delta = matchRatingRedCard
		case INJURY:
			delta = matchRatingInjury
		default:
			continue
		}
		for _, playerId := range event.TeamOwnerPlayers {
			if _, ok := ratings[playerId]; ok {
				ratings[playerId] += delta
			}
		}
	}

	homeWon := match.HomeGoals > match.AwayGoals
	awayWon := match.AwayGoals > match.HomeGoals
	for playerId, rating := range ratings {
		isHome := homePlayers[playerId]
		switch {
		case (isHome && homeWon) || (!isHome && awayWon):
			rating += matchRatingResultBonus
		case (isHome && awayWon) || (!isHome && homeWon):
			rating -= matchRatingResultBonus
		}
		rating += matchRatingJitter(match.MatchId, playerId)
		ratings[playerId] = clampRating(rating)
	}

	return &ratings
}

func clampRating(rating float64) float64 {
	if rating < matchRatingMin {
		rating = matchRatingMin
	}
	if rating > matchRatingMax {
		rating = matchRatingMax
	}
	return math.Round(rating*10) / 10
}

// matchRatingJitter deterministically derives a small (-0.3..0.3) per-player adjustment from
// matchId and playerId, so the same match always renders the same ratings (no genuine randomness
// involved - see ComputeMatchRatings) while still keeping a lineup from reading as suspiciously
// uniform.
func matchRatingJitter(matchId int64, playerId int64) float64 {
	h := uint64(matchId)*31 + uint64(playerId)
	h = (h*2654435761 + 1) % 1000003
	return (float64(h%61)/60.0)*0.6 - 0.3
}
