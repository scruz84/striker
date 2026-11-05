package common

import (
	"context"
	"testing"
	"time"
)

func TestPickRandomLiveMatch_Empty(t *testing.T) {
	if _, ok := PickRandomLiveMatch(nil); ok {
		t.Fatalf("expected no pick from an empty match list")
	}
}

func TestPickRandomLiveMatch_PrefersTopDivision(t *testing.T) {
	matches := []LiveMatchSummary{
		{Kind: MatchKindLeague, MatchId: 1, Tier: 3},
		{Kind: MatchKindLeague, MatchId: 2, Tier: 1},
		{Kind: MatchKindCup, MatchId: 3},
		{Kind: MatchKindFriendly, MatchId: 4},
	}

	for i := 0; i < 20; i++ {
		picked, ok := PickRandomLiveMatch(matches)
		if !ok {
			t.Fatalf("expected a pick")
		}
		if picked.MatchId != 2 {
			t.Fatalf("expected the tier-1 league match to always be picked over lower-priority matches, got match %d", picked.MatchId)
		}
	}
}

func TestPickRandomLiveMatch_FallsBackWhenNoLeagueMatchIsLive(t *testing.T) {
	matches := []LiveMatchSummary{
		{Kind: MatchKindCup, MatchId: 1},
		{Kind: MatchKindFriendly, MatchId: 2},
	}

	seen := make(map[int64]bool)
	for i := 0; i < 40; i++ {
		picked, ok := PickRandomLiveMatch(matches)
		if !ok {
			t.Fatalf("expected a pick")
		}
		seen[picked.MatchId] = true
	}

	if !seen[1] || !seen[2] {
		t.Fatalf("expected both non-league matches to be picked at least once across attempts, got %v", seen)
	}
}

func TestPickRandomLiveMatch_OnlyPicksAmongLowestTier(t *testing.T) {
	matches := []LiveMatchSummary{
		{Kind: MatchKindLeague, MatchId: 1, Tier: 2},
		{Kind: MatchKindLeague, MatchId: 2, Tier: 2},
		{Kind: MatchKindLeague, MatchId: 3, Tier: 4},
	}

	seen := make(map[int64]bool)
	for i := 0; i < 40; i++ {
		picked, ok := PickRandomLiveMatch(matches)
		if !ok {
			t.Fatalf("expected a pick")
		}
		if picked.MatchId == 3 {
			t.Fatalf("expected the higher-tier-number (lower priority) match never to be picked while a tier-2 match is live")
		}
		seen[picked.MatchId] = true
	}

	if !seen[1] || !seen[2] {
		t.Fatalf("expected both tier-2 matches to be picked at least once across attempts, got %v", seen)
	}
}

func TestLiveMatchChannel_PicksTheRightNamespace(t *testing.T) {
	cases := []struct {
		match LiveMatchSummary
		want  string
	}{
		{LiveMatchSummary{Kind: MatchKindLeague, MatchId: 7}, LeagueMatchChannel(7)},
		{LiveMatchSummary{Kind: MatchKindCup, MatchId: 7}, CupTieChannel(7)},
		{LiveMatchSummary{Kind: MatchKindFriendly, MatchId: 7}, FriendlyMatchChannel(7)},
	}
	for _, c := range cases {
		if got := liveMatchChannel(c.match); got != c.want {
			t.Errorf("liveMatchChannel(%+v) = %q, want %q", c.match, got, c.want)
		}
	}
}

func TestMinuteFromLiveMessage(t *testing.T) {
	cases := []struct {
		name       string
		msg        LiveMatchMessage
		wantMinute int
		wantOk     bool
	}{
		{"instant", LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 23}}, 23, true},
		{"event", LiveMatchMessage{Type: "event", Event: &Event{Minute: 67}}, 67, true},
		{"halftime", LiveMatchMessage{Type: "halftime"}, liveMatchFirstHalfMinutes, true},
		{"complete", LiveMatchMessage{Type: "complete"}, liveMatchFirstHalfMinutes + liveMatchSecondHalfMinutes, true},
		{"lineups has no minute yet", LiveMatchMessage{Type: "lineups"}, 0, false},
		{"instant missing its payload", LiveMatchMessage{Type: "instant"}, 0, false},
		{"event missing its payload", LiveMatchMessage{Type: "event"}, 0, false},
	}
	for _, c := range cases {
		minute, ok := minuteFromLiveMessage(c.msg)
		if minute != c.wantMinute || ok != c.wantOk {
			t.Errorf("%s: minuteFromLiveMessage(%+v) = (%d, %v), want (%d, %v)", c.name, c.msg, minute, ok, c.wantMinute, c.wantOk)
		}
	}
}

// TestEnrichSuggestedLiveMatch_PrefersRealBroadcastOverFormula reproduces the reported dashboard
// bug: the suggested-live-match card used to recompute its minute from elapsed real time
// (CurrentMatchMinute) instead of reading the same source of truth the match's own live-viewer
// page does (the channel's last published LiveMatchHub message). matchDate is picked so the
// formula's own estimate would land nowhere near the real broadcast's last reported minute,
// proving this reads the real message rather than estimating.
func TestEnrichSuggestedLiveMatch_PrefersRealBroadcastOverFormula(t *testing.T) {
	hub := liveHubForTest(t)
	dbCtx := &DatabaseContext{LiveHub: hub}

	match := LiveMatchSummary{Kind: MatchKindFriendly, MatchId: time.Now().UnixNano(), MatchDate: time.Now().Add(-70 * time.Minute)}
	channel := FriendlyMatchChannel(match.MatchId)
	t.Cleanup(func() { hub.client.Del(context.Background(), historyKey(channel), lastPublishKey(channel)) })

	hub.Publish(channel, LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 12}, HomeGoals: 2, AwayGoals: 1})

	if formulaMinute := CurrentMatchMinute(match.MatchDate); formulaMinute == 12 {
		t.Fatalf("test setup invalid: the formula's own estimate (%d) must differ from the real broadcast minute to prove which one enrichSuggestedLiveMatch actually used", formulaMinute)
	}

	if err := enrichSuggestedLiveMatch(dbCtx, &match); err != nil {
		t.Fatalf("enrichSuggestedLiveMatch returned an error: %s", err)
	}
	if match.Minute != 12 {
		t.Errorf("expected Minute to come from the real broadcast message (12), got %d", match.Minute)
	}
	if match.HomeGoals != 2 || match.AwayGoals != 1 {
		t.Errorf("expected goals from the real broadcast message (2-1), got %d-%d", match.HomeGoals, match.AwayGoals)
	}
}

// TestEnrichSuggestedLiveMatch_CompleteMessageReportsFinalMinute checks the other end of the same
// fix: once the broadcast has actually closed itself out (a normal "complete", or one
// engine.RecoverStaleLiveBroadcasts published after a stale timeout), the dashboard card reports
// the match as finished (minute 90) with the final score, instead of an estimate that could still
// show an in-progress-looking minute for a match that's actually done.
func TestEnrichSuggestedLiveMatch_CompleteMessageReportsFinalMinute(t *testing.T) {
	hub := liveHubForTest(t)
	dbCtx := &DatabaseContext{LiveHub: hub}

	match := LiveMatchSummary{Kind: MatchKindFriendly, MatchId: time.Now().UnixNano() + 1, MatchDate: time.Now().Add(-10 * time.Minute)}
	channel := FriendlyMatchChannel(match.MatchId)
	t.Cleanup(func() { hub.client.Del(context.Background(), historyKey(channel), lastPublishKey(channel)) })

	hub.Publish(channel, LiveMatchMessage{Type: "complete", HomeGoals: 3, AwayGoals: 0})

	if err := enrichSuggestedLiveMatch(dbCtx, &match); err != nil {
		t.Fatalf("enrichSuggestedLiveMatch returned an error: %s", err)
	}
	if want := liveMatchFirstHalfMinutes + liveMatchSecondHalfMinutes; match.Minute != want {
		t.Errorf("expected Minute %d for a completed match, got %d", want, match.Minute)
	}
	if match.HomeGoals != 3 || match.AwayGoals != 0 {
		t.Errorf("expected the final score (3-0), got %d-%d", match.HomeGoals, match.AwayGoals)
	}
}

// publishLiveForTest publishes msgs to a friendly's channel and returns the dbCtx and id to read
// them back with liveStateAsOfNow.
func publishLiveForTest(t *testing.T, idOffset int64, msgs ...LiveMatchMessage) (*DatabaseContext, int64) {
	t.Helper()
	hub := liveHubForTest(t)
	id := time.Now().UnixNano() + idOffset
	channel := FriendlyMatchChannel(id)
	t.Cleanup(func() { hub.client.Del(context.Background(), historyKey(channel), lastPublishKey(channel)) })
	for _, msg := range msgs {
		hub.Publish(channel, msg)
	}
	return &DatabaseContext{LiveHub: hub}, id
}

func TestLiveStateAsOfNow_SimulatedAheadOfKickoffIsPending(t *testing.T) {
	// A message is already on the channel, but kickoff is still 5 minutes away: no Redis lookup,
	// no score.
	dbCtx, id := publishLiveForTest(t, 10, LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 3}, HomeGoals: 1})
	state, err := liveStateAsOfNow(dbCtx, MatchKindFriendly, id, time.Now().Add(5*time.Minute), true)
	if err != nil || state != (LiveState{Status: LiveStatusPending}) {
		t.Errorf("got %+v, %v; want pending 0-0 at minute 0", state, err)
	}
}

func TestLiveStateAsOfNow_NotSimulatedYetIsPending(t *testing.T) {
	// Kickoff was 3 minutes ago but the engine has not simulated the match (played = false).
	state, err := liveStateAsOfNow(&DatabaseContext{}, MatchKindLeague, 1, time.Now().Add(-3*time.Minute), false)
	if err != nil || state != (LiveState{Status: LiveStatusPending}) {
		t.Errorf("got %+v, %v; want pending 0-0 at minute 0", state, err)
	}
}

func TestLiveStateAsOfNow_ReadsBroadcastStatus(t *testing.T) {
	cases := []struct {
		name string
		msg  LiveMatchMessage
		want LiveState
	}{
		{"live", LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 63}, HomeGoals: 1, AwayGoals: 2},
			LiveState{Status: LiveStatusLive, Minute: 63, HomeGoals: 1, AwayGoals: 2}},
		{"half-time", LiveMatchMessage{Type: "halftime", HomeGoals: 0, AwayGoals: 1},
			LiveState{Status: LiveStatusHalftime, Minute: 45, HomeGoals: 0, AwayGoals: 1}},
		{"broadcast finished", LiveMatchMessage{Type: "complete", HomeGoals: 3, AwayGoals: 2},
			LiveState{Status: LiveStatusFinished, Minute: 90, HomeGoals: 3, AwayGoals: 2}},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dbCtx, id := publishLiveForTest(t, int64(20+i), c.msg)
			state, err := liveStateAsOfNow(dbCtx, MatchKindFriendly, id, time.Now().Add(-40*time.Minute), true)
			if err != nil || state != c.want {
				t.Errorf("got %+v, %v; want %+v", state, err, c.want)
			}
		})
	}
}
