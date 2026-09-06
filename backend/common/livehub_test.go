package common

import (
	"context"
	"testing"
	"time"
)

// liveHubForTest returns a hub connected to this test process's Redis instance, skipping the
// test if none is reachable - these tests exercise the real pub/sub round trip (see
// LiveMatchHub's doc comment for why that round trip has to be real, not an in-process map), so
// unlike the rest of this package's tests they do need live infrastructure.
func liveHubForTest(t *testing.T) *LiveMatchHub {
	t.Helper()
	hub := NewLiveMatchHub()
	if err := hub.client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable, skipping: %s", err)
	}
	return hub
}

// awaitMessage waits up to a couple of seconds for a message - Publish/Subscribe now go through
// a real Redis round trip (network + the forwarding goroutine in Subscribe), so unlike a plain
// in-process channel a message is never available the very instant Publish returns. A closed
// channel (Subscribe's forwarding goroutine exits and closes it once the subscription itself is
// closed) is not a message - the zero value read from a closed channel must not be mistaken for
// one, so this checks the receive's ok flag rather than just the value.
func awaitMessage(t *testing.T, ch chan LiveMatchMessage) LiveMatchMessage {
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

// assertNoMessage waits briefly to make sure nothing arrives - long enough to catch a message
// that shouldn't be there without making the test suite slow. The channel closing counts as
// "nothing arrived", same as the timeout case - see awaitMessage's comment on why ok matters.
func assertNoMessage(t *testing.T, ch chan LiveMatchMessage) {
	t.Helper()
	select {
	case msg, ok := <-ch:
		if ok {
			t.Fatalf("expected no message, got: %+v", msg)
		}
	case <-time.After(300 * time.Millisecond):
	}
}

func TestLiveMatchHub_PublishDeliversToSubscriber(t *testing.T) {
	hub := liveHubForTest(t)
	channel := LeagueMatchChannel(1)

	ch, unsubscribe := hub.Subscribe(channel)
	defer unsubscribe()

	// Redis needs a moment to register the subscription before a Publish reaches it.
	time.Sleep(100 * time.Millisecond)

	hub.Publish(channel, LiveMatchMessage{Type: "instant", HomeGoals: 1})

	msg := awaitMessage(t, ch)
	if msg.Type != "instant" || msg.HomeGoals != 1 {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

func TestLiveMatchHub_UnsubscribeStopsDelivery(t *testing.T) {
	hub := liveHubForTest(t)
	channel := LeagueMatchChannel(1)

	ch, unsubscribe := hub.Subscribe(channel)
	unsubscribe()
	// Closing the subscription client-side doesn't instantly deregister it on the Redis server -
	// give the UNSUBSCRIBE a moment to actually land before relying on it.
	time.Sleep(200 * time.Millisecond)

	// Must not panic or block publishing to a channel nobody listens to any more, and the
	// unsubscribed channel must not receive it either.
	hub.Publish(channel, LiveMatchMessage{Type: "instant"})
	assertNoMessage(t, ch)
}

func TestLiveMatchHub_PublishNeverBlocksOnFullChannel(t *testing.T) {
	hub := liveHubForTest(t)
	channel := LeagueMatchChannel(1)

	_, unsubscribe := hub.Subscribe(channel)
	defer unsubscribe()
	time.Sleep(100 * time.Millisecond)

	// Flood well past the buffer - Publish must drop rather than block.
	done := make(chan struct{})
	go func() {
		for i := 0; i < liveMatchChannelBuffer*3; i++ {
			hub.Publish(channel, LiveMatchMessage{Type: "instant"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked instead of dropping messages to a full channel")
	}
}

func TestLiveMatchHub_LeagueAndCupChannelsAreIsolated(t *testing.T) {
	hub := liveHubForTest(t)

	// A league match and a cup tie can share the same numeric id (independent Postgres
	// sequences) - the hub must not let one leak into the other's subscribers.
	leagueCh, unsubLeague := hub.Subscribe(LeagueMatchChannel(1))
	defer unsubLeague()
	cupCh, unsubCup := hub.Subscribe(CupTieChannel(1))
	defer unsubCup()
	time.Sleep(100 * time.Millisecond)

	hub.Publish(LeagueMatchChannel(1), LiveMatchMessage{Type: "instant", HomeGoals: 7})

	msg := awaitMessage(t, leagueCh)
	if msg.HomeGoals != 7 {
		t.Fatalf("unexpected message on league channel: %+v", msg)
	}
	assertNoMessage(t, cupCh)
}

func TestLiveMatchHub_MultipleSubscribersAllReceive(t *testing.T) {
	hub := liveHubForTest(t)
	channel := LeagueMatchChannel(1)

	ch1, unsub1 := hub.Subscribe(channel)
	defer unsub1()
	ch2, unsub2 := hub.Subscribe(channel)
	defer unsub2()
	time.Sleep(100 * time.Millisecond)

	hub.Publish(channel, LiveMatchMessage{Type: "complete", HomeGoals: 2, AwayGoals: 1})

	for _, ch := range []chan LiveMatchMessage{ch1, ch2} {
		msg := awaitMessage(t, ch)
		if msg.Type != "complete" {
			t.Fatalf("unexpected message: %+v", msg)
		}
	}
}

func TestLiveMatchHub_HasSubscribersReflectsSubscriptionState(t *testing.T) {
	hub := liveHubForTest(t)
	channel := LeagueMatchChannel(2)

	if hub.HasSubscribers(channel) {
		t.Fatal("expected no subscribers before Subscribe")
	}

	_, unsubscribe := hub.Subscribe(channel)
	time.Sleep(100 * time.Millisecond)
	if !hub.HasSubscribers(channel) {
		t.Fatal("expected HasSubscribers to be true after Subscribe")
	}

	unsubscribe()
	time.Sleep(100 * time.Millisecond)
	if hub.HasSubscribers(channel) {
		t.Fatal("expected no subscribers after unsubscribe")
	}
}
