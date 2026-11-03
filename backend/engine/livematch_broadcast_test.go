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
		// The final authoritative score BroadcastMatchLive's "complete" message actually reports -
		// must agree with the goal event above, or this test's own fixture (not the code under
		// test) is what's wrong.
		HomeGoals:     1,
		AwayGoals:     0,
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

// TestBroadcastMatchLive_OnCompleteWaitsForKickoff pins the timing invariant simulateMatch/
// simulateCupTie/simulateFriendly now rely on: a match is fully simulated and persisted up to 15
// minutes before its own advertised kickoff (see GetSimulableMatchesStmtSql/
// GetSimulableFriendliesStmtSql), and their onComplete callback is where they defer every bit of
// state a team owner could notice outside this match's own live-gated page - the result
// notification, but also (since this fix) new injuries/suspensions (UpdatePlayerAvailability) and
// skill growth (updateSkillsAfterPlayedMatch, including the NOTIFICATION_PROGRESSION push it can
// send). None of that is safe to run before matchDate itself has arrived, or a user checking the
// league table, their squad, or their players' evolutions minutes before kickoff would already see
// the post-match state - the exact bug this test guards against.
func TestBroadcastMatchLive_OnCompleteWaitsForKickoff(t *testing.T) {
	hub := common.NewLiveMatchHub()
	if err := hub.Ping(); err != nil {
		t.Skipf("redis not reachable, skipping: %s", err)
	}

	channel := "test-broadcast-early-simulation"
	sub, unsubscribe := hub.Subscribe(channel)
	defer unsubscribe()

	events := []common.Event{}
	instants := []common.MatchInstant{{Minute: 1, HomeGoals: 0, AwayGoals: 0}}
	match := &common.MatchSimulation{Events: &events, MatchInstants: &instants}

	// Mirrors a match claimed for simulation ahead of its own kickoff - 3 seconds here (instead of
	// up to 15 real minutes) to keep the test fast.
	matchDate := time.Now().Add(3 * time.Second)

	done := make(chan struct{})
	dbCtx := &common.DatabaseContext{LiveHub: hub}
	BroadcastMatchLive(dbCtx, channel, match, matchDate, func() { close(done) })

	select {
	case msg := <-sub:
		t.Fatalf("got a broadcast message before kickoff: %+v - nothing published, and nothing deferred to onComplete, may happen before matchDate", msg)
	case <-done:
		t.Fatal("onComplete fired before kickoff")
	case <-time.After(1500 * time.Millisecond):
		// Halfway to kickoff, nothing has fired yet - correct.
	}

	select {
	case msg := <-sub:
		if msg.Type != "instant" || msg.Instant.Minute != 1 {
			t.Fatalf("expected the minute-1 instant once kickoff arrived, got %+v", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the minute-1 instant after kickoff")
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

// A conditional order's MENTALITY_CHANGE is broadcast at its own minute (after the previous
// minute's instant, never earlier), carrying the new mentality (openspec/specs/match-viewer
// "Mentality change in the chronicle").
func TestBroadcastMatchLive_MentalityChangeArrivesAtItsMinute(t *testing.T) {
	hub := common.NewLiveMatchHub()
	if err := hub.Ping(); err != nil {
		t.Skipf("redis not reachable, skipping: %s", err)
	}
	channel := "test-broadcast-mentality"
	sub, unsubscribe := hub.Subscribe(channel)
	defer unsubscribe()

	attacking := common.MENTALITY_ATTACKING
	events := []common.Event{{EventType: common.MENTALITY_CHANGE, Minute: 70, TeamOwner: 2, OtherTeam: 1, Mentality: &attacking}}
	instants := []common.MatchInstant{{Minute: 69}, {Minute: 70}}
	match := &common.MatchSimulation{Events: &events, MatchInstants: &instants}
	done := make(chan struct{})
	// Far enough in the past that minute 70 (after the halftime pause) is already due.
	BroadcastMatchLive(&common.DatabaseContext{LiveHub: hub}, channel, match, time.Now().Add(-3*time.Hour), func() { close(done) })

	if msg := awaitBroadcastMessage(t, sub); msg.Type != "instant" || msg.Instant.Minute != 69 {
		t.Fatalf("expected the minute-69 instant first, got %+v", msg)
	}
	msg := awaitBroadcastMessage(t, sub)
	if msg.Type != "event" || msg.Event.EventType != common.MENTALITY_CHANGE || msg.Event.Minute != 70 ||
		msg.Event.Mentality == nil || *msg.Event.Mentality != common.MENTALITY_ATTACKING {
		t.Fatalf("expected the minute-70 mentality change next, got %+v", msg)
	}
	<-done
}
