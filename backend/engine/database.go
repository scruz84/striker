package engine

import (
	"fmt"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// UpdatePlayerAvailability counts this match against every already-unavailable player on
// both teams (clearing anyone whose suspension/injury has now run out), then records any new
// suspensions/injuries picked up during this match - in that order, so a player booked in
// this match doesn't have their brand-new unavailability immediately decremented by the same
// match that caused it.
func UpdatePlayerAvailability(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	if err := common.DecrementUnavailabilityForTeam(dbCtx, match.Home); err != nil {
		log.Errorf("error decrementing unavailability for team %d: %s", match.Home, err)
	}
	if err := common.DecrementUnavailabilityForTeam(dbCtx, match.Away); err != nil {
		log.Errorf("error decrementing unavailability for team %d: %s", match.Away, err)
	}
	for _, u := range match.PendingUnavailabilities {
		if err := common.AddPlayerUnavailability(dbCtx, u.PlayerId, u.Reason, u.Matches); err != nil {
			log.Errorf("error recording unavailability for player %d: %s", u.PlayerId, err)
		}
	}
}

// UpdateTeamFinancesAfterMatch credits both teams their matchday income - gate receipts for the
// home side, sponsorship for both - since that's genuinely event-driven, tied to actually
// playing. Shared by league matches and cup ties (see simulateMatch/simulateCupTie), unlike
// wages: those are charged separately, only from simulateMatch (see PayMatchdayWages), so a cup
// tie played in the same round as a league match doesn't charge a team's wage bill twice.
// This is also the once-per-match hook point for any unowned team's baseline AI reaction (see
// RunAITeamMatchReaction) - matches are the natural cadence for that, and reusing this call site
// avoids a separate scheduler job for it. Proactive market/roster action (signings, listings,
// buying, renewals) deliberately stays off this call site - see RunAITeamManagement's own doc
// comment on why it only ever runs from the local-hour-gated daily tick instead.
//
// Also where each side's Fanbase drifts for this result (see applyFanbaseDrift) - same
// event-driven, competitive-result-only reasoning as sponsorship, and the same reason friendlies
// never reach this function at all.
//
// crowd is the home crowd drawn before kickoff (see drawMatchdayConditions) - the same one the
// home advantage used and the report shows; only its gate revenue is booked here.
func UpdateTeamFinancesAfterMatch(dbCtx *common.DatabaseContext, match *common.MatchSimulation, crowd matchdayCrowd) {
	creditTicketRevenue(dbCtx, match.Home, crowd)
	creditSponsorship(dbCtx, match.Home, match.HomeGoals, match.AwayGoals)
	creditSponsorship(dbCtx, match.Away, match.AwayGoals, match.HomeGoals)
	applyFanbaseDrift(dbCtx, match.Home, match.HomeGoals, match.AwayGoals)
	applyFanbaseDrift(dbCtx, match.Away, match.AwayGoals, match.HomeGoals)
	RunAITeamMatchReaction(dbCtx, match.Home)
	RunAITeamMatchReaction(dbCtx, match.Away)
}

// NotifyMatchResult tells both teams' owners how their league match went.
func NotifyMatchResult(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	homeTeam, errHome := common.GetTeamById(match.Home, dbCtx)
	awayTeam, errAway := common.GetTeamById(match.Away, dbCtx)
	if errHome != nil || errAway != nil {
		log.Errorf("error getting team names for match result notification: %v / %v", errHome, errAway)
		return
	}
	msg := common.NotificationMessage{
		Text: fmt.Sprintf("%s %d - %d %s", homeTeam.TeamName, match.HomeGoals, match.AwayGoals, awayTeam.TeamName),
		Key:  "notifications.messages.matchResult",
		Params: map[string]any{
			"home": homeTeam.TeamName, "homeGoals": match.HomeGoals,
			"away": awayTeam.TeamName, "awayGoals": match.AwayGoals,
		},
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Home, common.NOTIFICATION_MATCH_RESULT, msg); err != nil {
		log.Errorf("error notifying home team owner of match result: %s", err)
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Away, common.NOTIFICATION_MATCH_RESULT, msg); err != nil {
		log.Errorf("error notifying away team owner of match result: %s", err)
	}
}
