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

// liveMatchHalftimeMinute is the simulated minute after which the broadcast pauses for a
// halftime break (see liveMatchHalftimeDelay) instead of the ordinary per-minute delay.
const liveMatchHalftimeMinute = 45

// liveMatchHalftimeDelay replaces the ordinary liveMatchMinuteDelay just once, between minute
// 45 and minute 46 - a real match doesn't play straight through from kickoff to full time, and a
// viewer who'd been following the first half live would otherwise see the second half start
// exactly as fast as any other minute, with no sense that a break happened at all. So a match
// that kicks off at 14:00 now reaches half time at 14:45, resumes at 15:00, and finishes at
// 15:45 - 105 real minutes end to end rather than a flat 90.
const liveMatchHalftimeDelay = 15 * time.Minute

// BroadcastMatchLive replays a just-simulated match's events/instants, minute by minute, to
// any WebSocket clients currently subscribed to channel (see common.LeagueMatchChannel /
// common.CupTieChannel, common.LiveMatchHub, and handlers.LiveMatchHandler).
//
// This always runs, even when nobody is watching (the common case) - Publish-ing to a Redis
// channel with zero subscribers is a harmless no-op, and skipping the whole thing on an upfront
// "does anyone have this channel open right now" check would race a viewer who opens the match
// page a moment later: once that check said no, nothing would ever get published for this match
// again, even though the check and the viewer's page load could easily land within the same
// handful of milliseconds. A few hundred lightweight Redis publishes over ~105 real minutes per
// match (90 minutes of play plus the halftime break - see liveMatchHalftimeDelay) is cheap enough
// at this app's scale (see maxConcurrentSimulations) - it's just a sleeping goroutine between
// publishes, not active work - that there's no real efficiency case for that risk.
//
// This runs in its own goroutine so it never delays the caller (simulateMatch/simulateCupTie):
// the match is already fully computed and persisted by this point, so pacing its broadcast out
// over ~105 minutes is purely cosmetic for live viewers and must not hold up match processing.
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
		// Published once, before the first "instant" - both lineups are already fixed for the
		// whole match at this point (see LiveMatchMessage's own comment), so there's no reason to
		// wait or repeat this later. A cup tie or friendly missing either lineup (shouldn't
		// normally happen, but the zero value leaves these nil either way) just skips it - a live
		// viewer then falls back to the team-level qualification comparison only, same as before
		// this existed.
		if match.HomeLineup != nil && match.AwayLineup != nil && match.HomePlayers != nil && match.AwayPlayers != nil {
			dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
				Type:        "lineups",
				HomeLineup:  match.HomeLineup,
				AwayLineup:  match.AwayLineup,
				HomePlayers: match.HomePlayers,
				AwayPlayers: match.AwayPlayers,
			})
		}

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
			if instant.Minute == liveMatchHalftimeMinute {
				dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
					Type: "halftime", HomeGoals: homeGoals, AwayGoals: awayGoals,
				})
				time.Sleep(liveMatchHalftimeDelay)
				continue
			}
			time.Sleep(liveMatchMinuteDelay)
		}
		dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
			Type: "complete", HomeGoals: homeGoals, AwayGoals: awayGoals,
		})
		onComplete()
	}()
}
