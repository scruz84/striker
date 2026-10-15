package engine

import (
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// RecoverStaleLiveBroadcasts closes out any played league match, cup tie, or friendly currently
// inside its live-broadcast window (see common.GetLiveMatches) whose channel has gone silent for
// longer than common.LiveMatchStaleTimeout, without ever reaching "complete" - the system-wide
// counterpart to handlers.serveLiveMatch's own per-connection staleness fallback. That one only
// helps whichever single viewer happens to already be connected when its own timeout fires, and
// only after waiting out the full timeout itself; every other viewer who connects later (plus the
// dashboard's own suggested-live-match card, see common.GetLiveMatchesOverview, which now reads
// the channel's own last message - see common.enrichSuggestedLiveMatch) is still stuck either
// estimating from a formula or waiting out their own fresh timeout, for as long as nothing ever
// actually closes the channel itself.
//
// A crashed or redeployed engine.BroadcastMatchLive goroutine is exactly that case: the match is
// already fully simulated and persisted by the time it would have started publishing (see
// simulateMatch/simulateCupTie/simulateFriendly), so there's nothing left to recover by resuming
// the broadcast from wherever it died - closing it with the already-known final score, once, for
// everyone, is both simpler and no less correct than trying to replay the remaining minutes from
// a resumed goroutine would be.
func RecoverStaleLiveBroadcasts(dbCtx *common.DatabaseContext) {
	matches, err := common.GetLiveMatches(dbCtx)
	if err != nil {
		log.Errorf("error listing live matches for stale-broadcast recovery: %s", err)
		return
	}

	for _, match := range matches {
		recoverIfStale(dbCtx, match)
	}
}

func liveMatchChannelForSummary(match common.LiveMatchSummary) string {
	switch match.Kind {
	case common.MatchKindCup:
		return common.CupTieChannel(match.MatchId)
	case common.MatchKindFriendly:
		return common.FriendlyMatchChannel(match.MatchId)
	default:
		return common.LeagueMatchChannel(match.MatchId)
	}
}

func recoverIfStale(dbCtx *common.DatabaseContext, match common.LiveMatchSummary) {
	channel := liveMatchChannelForSummary(match)

	if last, ok := dbCtx.LiveHub.LastMessage(channel); ok && last.Type == "complete" {
		return // already closed itself normally
	}

	publishedAt, everPublished := dbCtx.LiveHub.LastPublishedAt(channel)
	if !everPublished {
		// Nothing has ever been published for this channel - fall back to kickoff itself as the
		// reference point, so a broadcast that crashed before its very first publish (minute 1
		// publishes right at kickoff - see engine.minutePublishOffset) is still caught instead of
		// silently exempted for having no history to measure staleness against.
		publishedAt = match.MatchDate
	}
	if time.Since(publishedAt) < common.LiveMatchStaleTimeout {
		return // still well within a healthy broadcast's own pacing, including the halftime pause
	}

	log.Warnf("live broadcast for %s stale (no message for %s) - closing with final result", channel, time.Since(publishedAt))
	dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{Type: "complete", HomeGoals: match.HomeGoals, AwayGoals: match.AwayGoals})
}
