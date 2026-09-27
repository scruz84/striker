package common

import "time"

// LiveBroadcastWindow mirrors the frontend's mightStillBeLive heuristic (maxLiveBroadcastWindowMs,
// see frontend/src/api/useLiveMatch.ts) - generously bounds how long after kickoff a match's live
// broadcast could still be in progress. A match flips played=true in the DB the instant it's
// simulated, well before that broadcast finishes pacing out to viewers, so any backend query that
// summarizes "completed" matches for a caller who isn't looking at that specific match's own
// (correctly live-gated) page must exclude anything still inside this window, or it ends up
// revealing a result that still looks "live" everywhere else (a past-meetings summary, a top
// scorers table, a cup bracket's next round) - see GetHeadToHead, GetTopScorers,
// GetPlayerCareerStats, and engine.IsCupRoundResolved.
const LiveBroadcastWindow = 2 * time.Hour

// LiveBroadcastCutoff is the match_date a match must be at or before to be safely treated as
// "definitely finished broadcasting" - see LiveBroadcastWindow.
func LiveBroadcastCutoff() time.Time {
	return time.Now().Add(-LiveBroadcastWindow)
}

// liveMatchMinuteDelay/liveMatchHalftimeDelay mirror engine.liveMatchMinuteDelay/
// liveMatchHalftimeDelay (backend/engine/livematch.go) - kept in sync manually since engine
// already depends on common and importing it back here would cycle. They pace how real elapsed
// time since kickoff maps onto the match's own clock for CurrentMatchMinute.
const liveMatchMinuteDelay = 60 * time.Second
const liveMatchHalftimeDelay = 15 * time.Minute
const liveMatchFirstHalfMinutes = 45
const liveMatchSecondHalfMinutes = 45

// CurrentMatchMinute estimates which match-minute a live broadcast that kicked off at matchDate
// is showing right now, given the fixed real-time pacing above (45 minutes, then a real
// liveMatchHalftimeDelay pause, then another 45 minutes) - the same schedule
// engine.BroadcastMatchLive actually paces its messages on. Used for a dashboard-wide glance at a
// suggested live match (see GetLiveMatchesOverview) where opening an actual live connection just
// to read the current minute would be overkill. Clamps to the last minute of each half, so a
// caller checking in during the halftime break or right at the end still gets a sensible number
// rather than one that overshoots.
func CurrentMatchMinute(matchDate time.Time) int {
	elapsed := time.Since(matchDate)
	firstHalf := time.Duration(liveMatchFirstHalfMinutes) * liveMatchMinuteDelay
	if elapsed < firstHalf {
		return int(elapsed/liveMatchMinuteDelay) + 1
	}

	afterHalftime := elapsed - firstHalf - liveMatchHalftimeDelay
	if afterHalftime < 0 {
		return liveMatchFirstHalfMinutes
	}

	minute := liveMatchFirstHalfMinutes + int(afterHalftime/liveMatchMinuteDelay) + 1
	if maxMinute := liveMatchFirstHalfMinutes + liveMatchSecondHalfMinutes; minute > maxMinute {
		return maxMinute
	}
	return minute
}
