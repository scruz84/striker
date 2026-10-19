package engine

import (
	"fmt"
	"math"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// SociosFanbaseScale/SociosFanbaseExponent define a club's target member count from its Fanbase:
// target = scale x fanbase^exponent. Superlinear on purpose: a big first-division club (fanbase
// towards 80) ends up with ~1,400-1,700 socios and outgrows a 1,000-seat ground, while a small
// third-division one (fanbase towards 30) settles at ~360-500 and never needs to expand. Demand is
// created by members, not by seats.
const SociosFanbaseScale = 2.2
const SociosFanbaseExponent = 1.5

// SociosDriftRate is how much of the gap to the target is closed at each season rollover.
const SociosDriftRate = 0.3

// SociosPositionBonus is the +/- swing of the target by final league position: +10% for the
// champion down to -10% for the last team.
const SociosPositionBonus = 0.1

// sociosNotifyMinChange is the relative change (of the previous count) from which an owner is
// told their membership moved - smaller moves aren't worth a notification.
const sociosNotifyMinChange = 0.05

// sociosTarget is the member count a club with this Fanbase and final league position (1-based,
// out of teamCount) is drifting towards.
func sociosTarget(fanbase int, position int, teamCount int) float64 {
	positionFactor := 1.0
	if teamCount > 1 {
		positionFactor = 1 + SociosPositionBonus - 2*SociosPositionBonus*float64(position-1)/float64(teamCount-1)
	}
	return SociosFanbaseScale * math.Pow(float64(fanbase), SociosFanbaseExponent) * positionFactor
}

// driftSocios moves current SociosDriftRate of the way towards the target, rounded to a whole
// member count and never below 1.
func driftSocios(current int, fanbase int, position int, teamCount int) int {
	target := sociosTarget(fanbase, position, teamCount)
	next := float64(current) + SociosDriftRate*(target-float64(current))
	return int(math.Max(1, math.Round(next)))
}

// updateSociosAtSeasonEnd drifts every team in a league's final standings towards its socios
// target and tells owners whose membership moved noticeably - called from
// awardSeasonTicketsAndMerchandising, which already has the final standings on hand.
func updateSociosAtSeasonEnd(dbCtx *common.DatabaseContext, standings []common.TeamStanding) {
	for i, team := range standings {
		fanbase, err := common.GetFanbase(dbCtx, team.TeamId)
		if err != nil {
			log.Errorf("error getting fanbase for team %d socios update: %s", team.TeamId, err)
			continue
		}
		current, err := common.GetSocios(dbCtx, team.TeamId)
		if err != nil {
			log.Errorf("error getting socios for team %d: %s", team.TeamId, err)
			continue
		}
		updated := driftSocios(current, fanbase, i+1, len(standings))
		if updated == current {
			continue
		}
		if err := common.SetSocios(dbCtx, team.TeamId, updated); err != nil {
			log.Errorf("error saving socios for team %d: %s", team.TeamId, err)
			continue
		}
		notifySociosChange(dbCtx, team.TeamId, current, updated)
	}
}

// notifySociosChange tells a team's owner their membership grew or shrank at season end, when the
// change is at least sociosNotifyMinChange of the old count. Silent for an unowned team.
func notifySociosChange(dbCtx *common.DatabaseContext, teamId int64, oldSocios int, newSocios int) {
	if oldSocios <= 0 || math.Abs(float64(newSocios-oldSocios))/float64(oldSocios) < sociosNotifyMinChange {
		return
	}
	key := "notifications.messages.sociosShrank"
	text := fmt.Sprintf("Your club now has %d socios (members), down from %d", newSocios, oldSocios)
	if newSocios > oldSocios {
		key = "notifications.messages.sociosGrew"
		text = fmt.Sprintf("Your club now has %d socios (members), up from %d", newSocios, oldSocios)
	}
	msg := common.NotificationMessage{
		Text:   text,
		Key:    key,
		Params: map[string]any{"socios": newSocios, "previous": oldSocios},
	}
	if err := common.NotifyTeamOwner(dbCtx, teamId, common.NOTIFICATION_FANBASE, msg); err != nil {
		log.Errorf("error notifying team %d owner of socios change: %s", teamId, err)
	}
}
