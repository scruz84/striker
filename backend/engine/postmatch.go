package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// newMatchResult is what a just-simulated fixture's result save stores: score, events, instants,
// its final per-player ratings (see common.ComputePlayerRatings - computed once, here, from the very
// contributions and events the simulation produced) and the pending post-match effects, which carry
// the same ratings for Form's update.
func newMatchResult(match *common.MatchSimulation, wentToPenalties bool) common.MatchResult {
	events := *match.Events
	instants := *match.MatchInstants
	ratings := common.ComputePlayerRatings(match.Home, match.Away, match.HomeLineup, match.AwayLineup, instants, events,
		common.MatchFinalMinute(instants, events))
	effects := newPostMatchPayload(match, wentToPenalties)
	if byId := common.RatingsById(ratings); byId != nil {
		effects.Ratings = *byId
	}
	return common.MatchResult{
		HomeGoals:     match.HomeGoals,
		AwayGoals:     match.AwayGoals,
		Events:        events,
		Instants:      instants,
		Effects:       effects,
		PlayerRatings: ratings,
		RivalryLevel:  match.RivalryLevel,
	}
}

// newPostMatchPayload captures what a just-simulated match's post-match effects need (see
// common.PostMatchPayload), so the result save can store it as pending in its own transaction.
// The effects themselves - availability countdown and new injuries/suspensions, skill growth and
// the result notification - wait for the live replay: they change state a team owner can see
// outside the match page, which must not appear before the match has nominally finished (see
// BroadcastMatchLive).
func newPostMatchPayload(match *common.MatchSimulation, wentToPenalties bool) *common.PostMatchPayload {
	events := make([]common.Event, 0)
	if match.Events != nil {
		events = *match.Events
	}
	return &common.PostMatchPayload{
		Home:                    match.Home,
		Away:                    match.Away,
		HomeGoals:               match.HomeGoals,
		AwayGoals:               match.AwayGoals,
		WentToPenalties:         wentToPenalties,
		PendingUnavailabilities: match.PendingUnavailabilities,
		HomeMinutesByPosition:   match.HomeMinutesByPosition,
		AwayMinutesByPosition:   match.AwayMinutesByPosition,
		HomeLineup:              match.HomeLineup,
		AwayLineup:              match.AwayLineup,
		Events:                  events,
	}
}

// completePostMatchEffects is a replay's onComplete: it claims the match's pending effects and
// applies them. If the recovery job (see ApplyDuePostMatchEffects) or another replica already took
// them, or the match was saved by a build that stored none, it does nothing.
func completePostMatchEffects(dbCtx *common.DatabaseContext, kind string, matchId int64) {
	payload, err := common.ClaimPostMatchEffects(dbCtx, kind, matchId)
	if err != nil {
		log.Errorf("error claiming the post-match effects of %s %d: %s", kind, matchId, err)
		return
	}
	if payload == nil {
		return
	}
	applyPostMatchEffects(dbCtx, kind, matchId, payload)
}

// ApplyDuePostMatchEffects applies every pending post-match record whose live-broadcast window
// has passed (see common.ClaimDuePostMatchEffects) - the net under completePostMatchEffects: a
// replay interrupted by an engine restart never reaches its onComplete, and a default result has
// no replay at all. Safe to run on several replicas at once: each record goes to one of them.
func ApplyDuePostMatchEffects(dbCtx *common.DatabaseContext) int {
	due, err := common.ClaimDuePostMatchEffects(dbCtx, common.MaxDuePostMatchEffectsPerPass)
	if err != nil {
		log.Errorf("error claiming due post-match effects: %s", err)
	}
	for i := range due {
		payload := due[i].Payload
		applyPostMatchEffects(dbCtx, due[i].Kind, due[i].MatchId, &payload)
	}
	return len(due)
}

// applyPostMatchEffects runs a match's deferred effects from its stored payload: new
// injuries/suspensions and the availability countdown, skill growth (unless nobody played - a
// default result), and the result notification matching the fixture's kind.
func applyPostMatchEffects(dbCtx *common.DatabaseContext, kind string, matchId int64, payload *common.PostMatchPayload) {
	events := payload.Events
	match := &common.MatchSimulation{
		Id:                      matchId,
		Home:                    payload.Home,
		Away:                    payload.Away,
		HomeGoals:               payload.HomeGoals,
		AwayGoals:               payload.AwayGoals,
		HomeLineup:              payload.HomeLineup,
		AwayLineup:              payload.AwayLineup,
		Events:                  &events,
		PendingUnavailabilities: payload.PendingUnavailabilities,
		HomeMinutesByPosition:   payload.HomeMinutesByPosition,
		AwayMinutesByPosition:   payload.AwayMinutesByPosition,
		StoredRatings:           payload.Ratings,
	}

	UpdatePlayerAvailability(dbCtx, match)
	if !payload.SkipSkillGrowth {
		homePlayers, errHome := resolveMatchTeamPlayers(dbCtx, payload.Home, payload.HomeLineup)
		awayPlayers, errAway := resolveMatchTeamPlayers(dbCtx, payload.Away, payload.AwayLineup)
		if errHome != nil || errAway != nil {
			log.Errorf("error reading the squads for the skill growth of %s %d: %v / %v", kind, matchId, errHome, errAway)
		} else {
			match.HomePlayers = playersToMapById(homePlayers)
			match.AwayPlayers = playersToMapById(awayPlayers)
			updateSkillsAfterPlayedMatch(dbCtx, match)
		}
	}

	applyRivalryForMatch(dbCtx, kind, payload)

	switch kind {
	case common.SimulationKindFriendly:
		notifyFriendlyResult(dbCtx, match)
	case common.SimulationKindCup:
		notifyCupTieResult(dbCtx, match, payload.WentToPenalties)
	default:
		NotifyMatchResult(dbCtx, match)
	}
}
