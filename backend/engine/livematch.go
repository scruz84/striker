package engine

import (
	"github.com/scruz84/striker/common"
	"time"
)

// liveMatchMinuteDelay paces the broadcast of an already-simulated match's minutes: real
// matches here are computed instantly (see runMatchSimulation), so this is what turns that
// instant result into something that looks like a match unfolding live for anyone watching -
// 90 minutes at this delay is a ~22 second broadcast.
const liveMatchMinuteDelay = 250 * time.Millisecond

// BroadcastMatchLive replays a just-simulated match's events/instants, minute by minute, to
// any WebSocket clients currently subscribed to channel (see common.LeagueMatchChannel /
// common.CupTieChannel, common.LiveMatchHub, and handlers.LiveMatchHandler).
//
// This always runs, even when nobody is watching (the common case) - Publish-ing to a Redis
// channel with zero subscribers is a harmless no-op, and skipping the whole thing on an upfront
// "does anyone have this channel open right now" check would race a viewer who opens the match
// page a moment later: once that check said no, nothing would ever get published for this match
// again, even though the check and the viewer's page load could easily land within the same
// handful of milliseconds. A few hundred lightweight Redis publishes over ~22 seconds per match
// is cheap enough at this app's scale (see maxConcurrentSimulations) that there's no real
// efficiency case for that risk.
//
// This runs in its own goroutine so it never delays the caller (simulateMatch/simulateCupTie):
// the match is already fully computed and persisted by this point, so pacing its broadcast out
// over ~22 seconds is purely cosmetic for live viewers and must not hold up match processing.
func BroadcastMatchLive(dbCtx *common.DatabaseContext, channel string, match *common.MatchSimulation) {
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
	}()
}
