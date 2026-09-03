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
// common.CupTieChannel, common.LiveMatchHub, and handlers.LiveMatchHandler). It's a no-op, and
// returns immediately, when nobody is watching - the common case, since most matches simulate
// with no live viewer at all.
//
// This runs in its own goroutine so it never delays the caller (simulateMatch/simulateCupTie):
// the match is already fully computed and persisted by this point, so pacing its broadcast out
// over ~22 seconds is purely cosmetic for live viewers and must not hold up match processing.
func BroadcastMatchLive(dbCtx *common.DatabaseContext, channel string, match *common.MatchSimulation) {
	if !dbCtx.LiveHub.HasSubscribers(channel) {
		return
	}

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
