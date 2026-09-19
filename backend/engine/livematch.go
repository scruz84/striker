package engine

import (
	"github.com/scruz84/striker/common"
	"time"
)

// liveMatchMinuteDelay paces the broadcast of an already-simulated match's minutes: real
// matches here are computed instantly (see runMatchSimulation), so this is what turns that
// instant result into something that looks like a match unfolding live for anyone watching -
// one real minute per simulated minute, so a viewer who checks in partway through a match's
// scheduled kickoff-to-90th-minute window sees it at roughly the corresponding point, the same
// way a real broadcast would. A much shorter delay here was tried first, but compressing the
// whole match into a few seconds around kickoff meant anyone who didn't happen to be watching in
// that exact narrow window never saw anything "live" at all - by the time they checked back it
// had already finished (issue #37).
const liveMatchMinuteDelay = 60 * time.Second

// BroadcastMatchLive replays a just-simulated match's events/instants, minute by minute, to
// any WebSocket clients currently subscribed to channel (see common.LeagueMatchChannel /
// common.CupTieChannel, common.LiveMatchHub, and handlers.LiveMatchHandler).
//
// This always runs, even when nobody is watching (the common case) - Publish-ing to a Redis
// channel with zero subscribers is a harmless no-op, and skipping the whole thing on an upfront
// "does anyone have this channel open right now" check would race a viewer who opens the match
// page a moment later: once that check said no, nothing would ever get published for this match
// again, even though the check and the viewer's page load could easily land within the same
// handful of milliseconds. A few hundred lightweight Redis publishes over ~90 real minutes per
// match is cheap enough at this app's scale (see maxConcurrentSimulations) - it's just a sleeping
// goroutine between publishes, not active work - that there's no real efficiency case for that
// risk.
//
// This runs in its own goroutine so it never delays the caller (simulateMatch/simulateCupTie):
// the match is already fully computed and persisted by this point, so pacing its broadcast out
// over ~90 minutes is purely cosmetic for live viewers and must not hold up match processing.
//
// onComplete runs after the final "complete" publish, once the whole broadcast has played out -
// callers use it to send the result notification (see simulateMatch/simulateCupTie/
// simulateFriendly) so a team owner never gets a push/in-app notification spelling out the score
// before their scheduled kickoff has even arrived. Matches are simulated and marked played up to
// 15 minutes ahead of kickoff (see GetSimulableMatchesStmtSql/GetSimulableFriendliesStmtSql) so
// the live broadcast has time to actually be "live" at kickoff instead of starting from zero -
// but a notification fired at that same early instant would spoil a result before the match
// nominally even started, which is exactly the bug this pattern replaced (the frontend's
// mightStillBeLive already hides the score everywhere else during this window - a persisted
// notification has no equivalent per-viewer gating, so it has to not exist yet instead).
func BroadcastMatchLive(dbCtx *common.DatabaseContext, channel string, match *common.MatchSimulation, onComplete func()) {
	instants := append([]common.MatchInstant(nil), *match.MatchInstants...)
	eventsByMinute := make(map[int][]common.Event)
	for _, event := range *match.Events {
		eventsByMinute[event.Minute] = append(eventsByMinute[event.Minute], event)
	}

	go func() {
		homeGoals, awayGoals := 0, 0
		for _, instant := range instants {
			homeGoals, awayGoals = instant.HomeGoals, instant.AwayGoals
			for _, event := range eventsByMinute[instant.Minute] {
				e := event
				dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
					Type: "event", Event: &e, HomeGoals: homeGoals, AwayGoals: awayGoals,
				})
			}
			i := instant
			dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
				Type: "instant", Instant: &i, HomeGoals: homeGoals, AwayGoals: awayGoals,
			})
			time.Sleep(liveMatchMinuteDelay)
		}
		dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
			Type: "complete", HomeGoals: homeGoals, AwayGoals: awayGoals,
		})
		onComplete()
	}()
}
