package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// TestBroadcastMatchLive_GoalReflectedImmediately reproduces the reported bug: a goal scored in
// minute M used to only show up in the score a whole broadcast-minute late, because the
// MatchInstant snapshot for minute M is recorded (by InitialSimulation) *before* that same
// minute's ScorerSimulation stage runs - see BroadcastMatchLive's own comment. This pins the fix:
// the "event" message for the goal, and the "instant" message published right after it for the
// same minute, must both already carry the post-goal score, not the stale pre-goal one still
// sitting on the MatchInstant itself.
func TestBroadcastMatchLive_GoalReflectedImmediately(t *testing.T) {
	hub := common.NewLiveMatchHub()
	if err := hub.Ping(); err != nil {
		t.Skipf("redis not reachable, skipping: %s", err)
	}

	channel := "test-broadcast-goal"
	sub, unsubscribe := hub.Subscribe(channel)
	defer unsubscribe()

	events := []common.Event{
		{EventType: 1, Minute: 2, HomeGoals: 1, AwayGoals: 0},
	}
	instants := []common.MatchInstant{
		{Minute: 1, HomeGoals: 0, AwayGoals: 0},
		// Stale on purpose: this is what InitialSimulation actually records for minute 2 - before
		// that minute's goal - matching the real pipeline's ordering.
		{Minute: 2, HomeGoals: 0, AwayGoals: 0},
	}
	match := &common.MatchSimulation{
		Events:        &events,
		MatchInstants: &instants,
	}

	// matchDate far enough in the past that every minutePublishOffset wait has already elapsed -
	// the broadcast goroutine runs straight through with no real sleeping.
	matchDate := time.Now().Add(-1 * time.Hour)

	done := make(chan struct{})
	dbCtx := &common.DatabaseContext{LiveHub: hub}
	BroadcastMatchLive(dbCtx, channel, match, matchDate, func() { close(done) })

	msg1 := awaitBroadcastMessage(t, sub)
	if msg1.Type != "instant" || msg1.Instant.Minute != 1 {
		t.Fatalf("expected minute-1 instant first, got %+v", msg1)
	}
	if msg1.HomeGoals != 0 || msg1.AwayGoals != 0 {
		t.Fatalf("expected 0-0 before the goal, got %d-%d", msg1.HomeGoals, msg1.AwayGoals)
	}

	msg2 := awaitBroadcastMessage(t, sub)
	if msg2.Type != "event" {
		t.Fatalf("expected the minute-2 goal event next, got %+v", msg2)
	}
	if msg2.HomeGoals != 1 || msg2.AwayGoals != 0 {
		t.Fatalf("goal event must carry the post-goal score, got %d-%d", msg2.HomeGoals, msg2.AwayGoals)
	}

	msg3 := awaitBroadcastMessage(t, sub)
	if msg3.Type != "instant" || msg3.Instant.Minute != 2 {
		t.Fatalf("expected minute-2 instant next, got %+v", msg3)
	}
	if msg3.HomeGoals != 1 || msg3.AwayGoals != 0 {
		t.Fatalf("minute-2 instant must already reflect the goal scored that same minute, got %d-%d (this is the regression: the instant's own stale snapshot must not leak into the broadcast score)", msg3.HomeGoals, msg3.AwayGoals)
	}

	msg4 := awaitBroadcastMessage(t, sub)
	if msg4.Type != "complete" {
		t.Fatalf("expected complete last, got %+v", msg4)
	}
	if msg4.HomeGoals != 1 || msg4.AwayGoals != 0 {
		t.Fatalf("complete must keep the final score, got %d-%d", msg4.HomeGoals, msg4.AwayGoals)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("onComplete was never called")
	}
}

func awaitBroadcastMessage(t *testing.T, ch chan common.LiveMatchMessage) common.LiveMatchMessage {
	t.Helper()
	select {
	case msg, ok := <-ch:
		if !ok {
			t.Fatal("channel closed before a message arrived")
		}
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a message")
		panic("unreachable")
	}
}
