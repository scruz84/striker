package engine

import (
	"github.com/scruz84/striker/common"
	"time"
)

// liveMatchMinuteDelay paces the broadcast of an already-simulated match's minutes: real
// matches here are computed instantly (see runMatchSimulation), so this is what turns that
// instant result into something that looks like a match unfolding live for anyone watching -
// one real minute per simulated minute, so a viewer who checks in partway through a match's
// scheduled kickoff-to-final-whistle window sees it at roughly the corresponding point, the same
// way a real broadcast would. A much shorter delay here was tried first, but compressing the
// whole match into a few seconds around kickoff meant anyone who didn't happen to be watching in
// that exact narrow window never saw anything "live" at all - by the time they checked back it
// had already finished (issue #37).
const liveMatchMinuteDelay = 60 * time.Second

// liveMatchHalftimeMinute is the simulated minute after which the broadcast pauses for a
// halftime break (see liveMatchHalftimeDelay) instead of the ordinary per-minute delay.
const liveMatchHalftimeMinute = 45

// liveMatchHalftimeDelay is how long after minute 45 is published minute 46 is - a real match
// doesn't play straight through from kickoff to full time, and a viewer who'd been following the
// first half live would otherwise see the second half start exactly as fast as any other minute,
// with no sense that a break happened at all. The first half's added minutes (45+1..45+N, see
// playMatchMinutes) are played out of this same window, so the break itself lasts 15 - N minutes
// and minute 46 always publishes 59 minutes after kickoff. A match that kicks off at 14:00 with 3
// and 5 added minutes reaches 45 at 14:44, plays 45+1..45+3 until 14:48, resumes at 14:59,
// reaches 90 at 15:43 and blows the final whistle after 90+5, at 15:48.
const liveMatchHalftimeDelay = 15 * time.Minute

// minutePublishOffset is how long after kickoff (matchDate) the minute (minute, addedMinute)
// should be published, under the pacing above - the mirror image of common.CurrentMatchMinute,
// which goes the other way (elapsed time -> minute) for the same schedule. Minute 1 publishes
// right at kickoff (offset zero); 45+N one minute per added minute after 45; minute 46 right as
// the halftime break ends; 90+N one minute per added minute after 90.
func minutePublishOffset(minute int, addedMinute int) time.Duration {
	if minute <= liveMatchHalftimeMinute {
		return time.Duration(minute-1+addedMinute) * liveMatchMinuteDelay
	}
	return time.Duration(minute-2+addedMinute)*liveMatchMinuteDelay + liveMatchHalftimeDelay
}

// MaxBroadcastDuration is how long after kickoff the longest possible replay publishes its final
// whistle: minute 90 plus the most added minutes a second half can get. It must stay well inside
// common.LiveBroadcastWindow (see TestLongestBroadcastFitsLiveWindow), which every time-based
// spoiler gate relies on to know a match has finished.
func MaxBroadcastDuration() time.Duration {
	return minutePublishOffset(secondHalfLastMinute, MaxAddedSecondHalf)
}

// BroadcastMatchLive replays a just-simulated match's events/instants, minute by minute, to
// any WebSocket clients currently subscribed to channel (see common.LeagueMatchChannel /
// common.CupTieChannel, common.LiveMatchHub, and handlers.LiveMatchHandler) - paced against
// matchDate, the fixture's own scheduled kickoff, rather than against whenever this function
// itself happens to run.
//
// That distinction matters because this can run well after kickoff: matches are normally claimed
// for simulation up to 15 minutes early (see GetSimulableMatchesStmtSql/
// GetSimulableFriendliesStmtSql) so the broadcast is already under way by kickoff, but a scheduler
// gap (a restart, a stuck pass - see GetStuckMatches/FinishStuckMatches) or an admin-triggered
// re-run can leave a match unsimulated for much longer, and a cup tie's own claiming query has no
// upper bound on match_date staleness at all (see GetSimulableCupTiesStmtSql). Pacing every minute
// against matchDate+minutePublishOffset(minute) rather than a fixed per-iteration delay means any
// minute whose publish time has already passed by the time this loop reaches it is caught up on
// immediately (no sleep) - so a match that starts broadcasting an hour late still shows viewers
// the minute that actually corresponds to real elapsed time since kickoff, instead of restarting
// its own clock from 1 as if kickoff were happening right now (the bug this replaced).
//
// This always runs, even when nobody is watching (the common case) - Publish-ing to a Redis
// channel with zero subscribers is a harmless no-op, and skipping the whole thing on an upfront
// "does anyone have this channel open right now" check would race a viewer who opens the match
// page a moment later: once that check said no, nothing would ever get published for this match
// again, even though the check and the viewer's page load could easily land within the same
// handful of milliseconds. A few hundred lightweight Redis publishes over up to 108 real minutes
// per match (90 minutes of play, the halftime break and the added time - see liveMatchHalftimeDelay
// and MaxBroadcastDuration) is cheap enough
// at this app's scale (see maxConcurrentSimulations) - it's just a sleeping goroutine between
// publishes, not active work - that there's no real efficiency case for that risk.
//
// This runs in its own goroutine so it never delays the caller (simulateMatch/simulateCupTie):
// the match is already fully computed and persisted by this point, so pacing its broadcast out
// over up to 108 minutes is purely cosmetic for live viewers and must not hold up match processing.
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
// notification has no equivalent per-viewer gating, so it has to not exist yet instead). Pacing
// minute 1 itself against matchDate (rather than publishing it immediately) keeps that guarantee
// intact for an on-time match too: nothing publishes before kickoff has actually arrived.
func BroadcastMatchLive(dbCtx *common.DatabaseContext, channel string, match *common.MatchSimulation, matchDate time.Time, onComplete func()) {
	instants := append([]common.MatchInstant(nil), *match.MatchInstants...)
	// Keyed by clock minute and added minute: 45 and 45+1 are different minutes of the replay.
	eventsByMinute := make(map[[2]int][]common.Event)
	for _, event := range *match.Events {
		key := [2]int{event.Minute, event.AddedMinute}
		eventsByMinute[key] = append(eventsByMinute[key], event)
	}

	// match.HomeLineup/AwayLineup are always the XI submitted before kickoff - nothing in the
	// simulation pipeline ever updates them when a substitution happens (see
	// common.ApplySubstitutionToLineup's doc comment) - so these two local copies are what
	// actually evolve as the broadcast replays SUBSTITUTION events below, never the originals.
	homeLineup := common.CopyLineup(match.HomeLineup)
	awayLineup := common.CopyLineup(match.AwayLineup)
	// Every viewer gets the broadcast: the captain's armband is public, the set-piece takers aren't.
	common.RedactLineupTakers(homeLineup)
	common.RedactLineupTakers(awayLineup)

	go func() {
		// Published once, before the first "instant". A cup tie or friendly missing either
		// lineup (shouldn't normally happen, but the zero value leaves these nil either way) just
		// skips it - a live viewer then falls back to the team-level qualification comparison
		// only, same as before this existed.
		if homeLineup != nil && awayLineup != nil && match.HomePlayers != nil && match.AwayPlayers != nil {
			dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
				Type:        "lineups",
				HomeLineup:  homeLineup,
				AwayLineup:  awayLineup,
				HomePlayers: match.HomePlayers,
				AwayPlayers: match.AwayPlayers,
			})
		}

		// homeGoals/awayGoals is tracked from each Event's own HomeGoals/AwayGoals (set from
		// match.HomeGoals/match.AwayGoals at the instant it was recorded - see ScorerSimulation and
		// friends), not from the MatchInstant snapshot below: InitialSimulation appends that
		// snapshot at the start of its minute's pipeline stage, before ScorerSimulation runs for
		// that same minute, so instant.HomeGoals/AwayGoals always lag a goal scored that minute by
		// one full minute of broadcast - the scoreboard's minute counter would tick forward on
		// schedule while the score itself stayed stuck at the pre-goal count until the next
		// instant. Events, appended after whichever stage actually changed the score, don't have
		// that lag.
		homeGoals, awayGoals := 0, 0
		// The minute the replay ended on, which the "complete" message carries (90+N for a full
		// match, earlier for one abandoned for lack of players).
		lastMinute, lastAddedMinute := secondHalfLastMinute, match.SecondHalfAddedTime
		if len(instants) > 0 {
			lastMinute, lastAddedMinute = 0, 0
		}
		for _, instant := range instants {
			if wait := time.Until(matchDate.Add(minutePublishOffset(instant.Minute, instant.AddedMinute))); wait > 0 {
				time.Sleep(wait)
			}
			for _, event := range eventsByMinute[[2]int{instant.Minute, instant.AddedMinute}] {
				e := event
				homeGoals, awayGoals = e.HomeGoals, e.AwayGoals
				msg := common.LiveMatchMessage{
					Type: "event", Event: &e, HomeGoals: homeGoals, AwayGoals: awayGoals,
				}
				// A substitution changes who's actually on the pitch - update this broadcast's
				// own evolving lineup copies and publish them alongside the event itself, so a
				// live viewer's head-to-head pitch (MatchHeadToHeadPitch, fed by these same
				// homeLineup/awayLineup fields via useLiveMatch) reflects it instead of staying
				// frozen at the starting XI for the rest of the match.
				if e.EventType == common.SUBSTITUTION {
					common.ApplySubstitutionToLineup(homeLineup, e)
					common.ApplySubstitutionToLineup(awayLineup, e)
					msg.HomeLineup = homeLineup
					msg.AwayLineup = awayLineup
				}
				dbCtx.LiveHub.Publish(channel, msg)
			}
			i := instant
			dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
				Type: "instant", Instant: &i, HomeGoals: homeGoals, AwayGoals: awayGoals,
			})
			// Half time comes after the last minute of the first half: 45 plus its added time.
			if instant.Minute == liveMatchHalftimeMinute && instant.AddedMinute == match.FirstHalfAddedTime {
				dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
					Type: "halftime", HomeGoals: homeGoals, AwayGoals: awayGoals,
					Minute: instant.Minute, AddedMinute: instant.AddedMinute,
				})
			}
			lastMinute, lastAddedMinute = instant.Minute, instant.AddedMinute
		}
		// Falls back to the match's own authoritative final score rather than whatever the last
		// event happened to leave homeGoals/awayGoals at - the two always agree except when the
		// match was abandoned before kickoff for a side falling to MinPlayersToContinueMatch (see
		// applyForfeitIfUnderstaffed), which leaves `instants` empty and the loop above never runs,
		// so homeGoals/awayGoals would otherwise still be sitting at their initial 0-0 here.
		homeGoals, awayGoals = match.HomeGoals, match.AwayGoals
		dbCtx.LiveHub.Publish(channel, common.LiveMatchMessage{
			Type: "complete", HomeGoals: homeGoals, AwayGoals: awayGoals,
			Minute: lastMinute, AddedMinute: lastAddedMinute,
		})
		onComplete()
	}()
}
