package engine

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// FanbaseResultDriftProbability is the per-match chance a result-based Fanbase move happens at
// all (see driftFanbaseForResult) - deliberately rare: a club's Fanbase is a slow-moving identity,
// not a weekly mood like Form (engine/progression.go's driftFormPlayed moves every match a player
// features in). Most matches leave Fanbase untouched even mid-streak.
const FanbaseResultDriftProbability = 0.15

// FanbaseWinDelta/FanbaseLossDelta are how much Fanbase moves when a result-drift roll succeeds
// (see FanbaseResultDriftProbability) - a draw moves it by 0, so a successful roll on a draw
// simply does nothing.
const FanbaseWinDelta = 1
const FanbaseLossDelta = -1

// FanbaseTierPullProbability is a separate, independent per-match roll that nudges Fanbase one
// point toward wherever its current division tier's own target level sits (see
// fanbaseTierTarget) - what gives promotion/relegation a slow, multi-season effect on crowd size
// on top of the immediate prize-money/sponsorship-tier jump they already cause.
const FanbaseTierPullProbability = 0.05

// FanbaseTierTarget is the Fanbase level each division tier slowly pulls a team toward (see
// FanbaseTierPullProbability) - top-flight clubs build bigger followings over time, bottom-tier
// ones smaller, independent of any single match's result.
var FanbaseTierTarget = map[int]int{1: 80, 2: 50, 3: 30}

// fanbaseTierTarget falls back to the lowest listed tier's own target for a tier beyond the ones
// FanbaseTierTarget lists (a deeper admin-configured world - see common.TierConfig), rather than
// pulling toward the zero value of a missing map entry.
func fanbaseTierTarget(tier int) int {
	if target, ok := FanbaseTierTarget[tier]; ok {
		return target
	}
	return FanbaseTierTarget[3]
}

// driftFanbaseForResult computes Fanbase's new value after one match's result, independent of
// driftFanbaseTierPull - a pure function so the probability/delta logic is directly testable
// without a database, same split as driftStamina/driftFormPlayed (engine/progression.go) vs. the
// DB-writing callers that use them.
func driftFanbaseForResult(fanbase int, outcome matchOutcome, r *rand.Rand) int {
	if r.Float64() >= FanbaseResultDriftProbability {
		return fanbase
	}
	switch outcome {
	case outcomeWin:
		return clampAttribute(fanbase + FanbaseWinDelta)
	case outcomeLoss:
		return clampAttribute(fanbase + FanbaseLossDelta)
	default:
		return fanbase
	}
}

// driftFanbaseTierPull nudges fanbase one point toward its division tier's own target level (see
// FanbaseTierTarget), independent of this match's result - a pure function, same testability
// reasoning as driftFanbaseForResult.
func driftFanbaseTierPull(fanbase int, tier int, r *rand.Rand) int {
	if r.Float64() >= FanbaseTierPullProbability {
		return fanbase
	}
	target := fanbaseTierTarget(tier)
	switch {
	case fanbase < target:
		return clampAttribute(fanbase + 1)
	case fanbase > target:
		return clampAttribute(fanbase - 1)
	default:
		return fanbase
	}
}

// applyFanbaseDrift updates teamId's Fanbase after one league match or cup tie result
// (goalsFor/goalsAgainst from teamId's own perspective, same per-side call shape
// creditSponsorship uses) and, if it crossed into a new ten-point band, lets the owner know (see
// notifyFanbaseMilestone). See UpdateTeamFinancesAfterMatch, the only caller - friendlies never
// reach this, same "not a competitive result" carve-out sponsorship already uses.
func applyFanbaseDrift(dbCtx *common.DatabaseContext, teamId int64, goalsFor int, goalsAgainst int) {
	tier, err := common.GetTeamTier(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting tier for team %d fanbase drift: %s", teamId, err)
		return
	}
	fanbase, err := common.GetFanbase(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting fanbase for team %d: %s", teamId, err)
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	updated := driftFanbaseForResult(fanbase, outcomeFor(goalsFor, goalsAgainst), r)
	updated = driftFanbaseTierPull(updated, tier, r)
	if updated == fanbase {
		return
	}

	if err := common.SetFanbase(dbCtx, teamId, updated); err != nil {
		log.Errorf("error saving fanbase for team %d: %s", teamId, err)
		return
	}

	notifyFanbaseMilestone(dbCtx, teamId, fanbase, updated)
}

// notifyFanbaseMilestone tells a team's owner when Fanbase crosses into a new ten-point band
// (e.g. 47 -> 51, or 53 -> 49) - frequent enough to notice a real trend, rare enough not to spam a
// notification on every single small move (most of which don't even happen - see
// FanbaseResultDriftProbability). Silent for an unowned (AI-managed) team, same as every other
// NotifyTeamOwner call - there's nobody to tell.
func notifyFanbaseMilestone(dbCtx *common.DatabaseContext, teamId int64, oldFanbase int, newFanbase int) {
	if oldFanbase/10 == newFanbase/10 {
		return
	}

	key := "notifications.messages.fanbaseShrank"
	text := fmt.Sprintf("Your fanbase has shrunk to %d", newFanbase)
	if newFanbase > oldFanbase {
		key = "notifications.messages.fanbaseGrew"
		text = fmt.Sprintf("Your fanbase has grown to %d", newFanbase)
	}
	msg := common.NotificationMessage{
		Text:   text,
		Key:    key,
		Params: map[string]any{"fanbase": newFanbase},
	}
	if err := common.NotifyTeamOwner(dbCtx, teamId, common.NOTIFICATION_FANBASE, msg); err != nil {
		log.Errorf("error notifying team %d owner of fanbase milestone: %s", teamId, err)
	}
}

// FanbaseCommercialMinFactor/MaxFactor scale sponsorship and season-ticket/merchandising income
// by Fanbase (see fanbaseCommercialFactor) - a gentler swing than stadium.go's own
// fanbaseAttendanceFactor (0.5x-1.5x), since a sponsor or season-ticket buyer already cares most
// about which tier a team plays in (see sponsorshipForTier) or how big its stadium is; a bigger
// Fanbase just makes an otherwise-identical club a somewhat more attractive commercial partner on
// top of that.
const FanbaseCommercialMinFactor = 0.8
const FanbaseCommercialMaxFactor = 1.2

// fanbaseCommercialFactor maps a 0-99 Fanbase value onto [FanbaseCommercialMinFactor,
// FanbaseCommercialMaxFactor] - see creditSponsorship (engine/sponsorship.go) and
// awardSeasonTicketsAndMerchandising (engine/seasonincome.go), both of which multiply their own
// base income figure by this. common.DefaultFanbase (50) lands almost exactly at 1.0x, so an
// unestablished team's income behaves the same as before this existed.
func fanbaseCommercialFactor(fanbase int) float64 {
	return FanbaseCommercialMinFactor + float64(fanbase)/99.0*(FanbaseCommercialMaxFactor-FanbaseCommercialMinFactor)
}
