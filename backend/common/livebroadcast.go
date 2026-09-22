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
