package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

func liveHubForRecoveryTest(t *testing.T) *common.LiveMatchHub {
	t.Helper()
	hub := common.NewLiveMatchHub()
	if err := hub.Ping(); err != nil {
		t.Skipf("redis not reachable, skipping: %s", err)
	}
	return hub
}

func TestLiveMatchChannelForSummary(t *testing.T) {
	cases := []struct {
		match common.LiveMatchSummary
		want  string
	}{
		{common.LiveMatchSummary{Kind: common.MatchKindLeague, MatchId: 9}, common.LeagueMatchChannel(9)},
		{common.LiveMatchSummary{Kind: common.MatchKindCup, MatchId: 9}, common.CupTieChannel(9)},
		{common.LiveMatchSummary{Kind: common.MatchKindFriendly, MatchId: 9}, common.FriendlyMatchChannel(9)},
	}
	for _, c := range cases {
		if got := liveMatchChannelForSummary(c.match); got != c.want {
			t.Errorf("liveMatchChannelForSummary(%+v) = %q, want %q", c.match, got, c.want)
		}
	}
}

// TestRecoverIfStale_ClosesAChannelThatWentSilent reproduces the reported bug: a backend crash or
// redeploy kills BroadcastMatchLive's goroutine outright, leaving the channel's last message
// stuck at whatever minute it reached. Before this fix, nothing would ever publish a "complete"
// for it again - every future viewer's own connection, and the dashboard's suggested-live-match
// card, would be stuck estimating or waiting out their own fresh timeout. recoverIfStale must
// close it out once, with the authoritative final score, as soon as it's been silent for longer
// than common.LiveMatchStaleTimeout.
func TestRecoverIfStale_ClosesAChannelThatWentSilent(t *testing.T) {
	hub := liveHubForRecoveryTest(t)
	dbCtx := &common.DatabaseContext{LiveHub: hub}

	old := common.LiveMatchStaleTimeout
	common.LiveMatchStaleTimeout = 50 * time.Millisecond
	t.Cleanup(func() { common.LiveMatchStaleTimeout = old })

	match := common.LiveMatchSummary{
		Kind: common.MatchKindFriendly, MatchId: time.Now().UnixNano(),
		MatchDate: time.Now().Add(-20 * time.Minute), HomeGoals: 2, AwayGoals: 1,
	}
	channel := liveMatchChannelForSummary(match)

	hub.Publish(channel, common.LiveMatchMessage{Type: "instant", HomeGoals: 1, AwayGoals: 1})
	time.Sleep(100 * time.Millisecond) // well past the shrunk LiveMatchStaleTimeout

	recoverIfStale(dbCtx, match)

	last, ok := hub.LastMessage(channel)
	if !ok || last.Type != "complete" {
		t.Fatalf("expected the channel to be closed with a complete message, got: %+v (ok=%v)", last, ok)
	}
	if last.HomeGoals != 2 || last.AwayGoals != 1 {
		t.Fatalf("expected the authoritative final score (2-1), got %d-%d", last.HomeGoals, last.AwayGoals)
	}
}

// TestRecoverIfStale_LeavesAHealthyBroadcastAlone makes sure a broadcast that's merely between
// messages (the ordinary once-a-minute pace, or the halftime pause) is never closed early.
func TestRecoverIfStale_LeavesAHealthyBroadcastAlone(t *testing.T) {
	hub := liveHubForRecoveryTest(t)
	dbCtx := &common.DatabaseContext{LiveHub: hub}

	match := common.LiveMatchSummary{
		Kind: common.MatchKindFriendly, MatchId: time.Now().UnixNano() + 1,
		MatchDate: time.Now().Add(-5 * time.Minute), HomeGoals: 0, AwayGoals: 0,
	}
	channel := liveMatchChannelForSummary(match)

	hub.Publish(channel, common.LiveMatchMessage{Type: "instant", HomeGoals: 0, AwayGoals: 0})

	recoverIfStale(dbCtx, match)

	last, ok := hub.LastMessage(channel)
	if !ok || last.Type != "instant" {
		t.Fatalf("expected the healthy broadcast's own last message to be left alone, got: %+v (ok=%v)", last, ok)
	}
}

// TestRecoverIfStale_LeavesAnAlreadyCompletedBroadcastAlone makes sure a channel that already
// closed itself out normally is never republished to.
func TestRecoverIfStale_LeavesAnAlreadyCompletedBroadcastAlone(t *testing.T) {
	hub := liveHubForRecoveryTest(t)
	dbCtx := &common.DatabaseContext{LiveHub: hub}

	old := common.LiveMatchStaleTimeout
	common.LiveMatchStaleTimeout = 50 * time.Millisecond
	t.Cleanup(func() { common.LiveMatchStaleTimeout = old })

	match := common.LiveMatchSummary{
		Kind: common.MatchKindFriendly, MatchId: time.Now().UnixNano() + 2,
		MatchDate: time.Now().Add(-120 * time.Minute), HomeGoals: 4, AwayGoals: 2,
	}
	channel := liveMatchChannelForSummary(match)

	hub.Publish(channel, common.LiveMatchMessage{Type: "complete", HomeGoals: 4, AwayGoals: 2})
	time.Sleep(100 * time.Millisecond)

	seqBefore, _ := hub.LastMessage(channel)
	recoverIfStale(dbCtx, match)
	seqAfter, _ := hub.LastMessage(channel)

	if seqBefore.Seq != seqAfter.Seq {
		t.Fatalf("expected no new message published for an already-completed channel: before %+v, after %+v", seqBefore, seqAfter)
	}
}
