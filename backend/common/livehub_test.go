package common

import "testing"

func TestLiveMatchHub_PublishDeliversToSubscriber(t *testing.T) {
	hub := NewLiveMatchHub()
	channel := LeagueMatchChannel(1)

	ch, unsubscribe := hub.Subscribe(channel)
	defer unsubscribe()

	if !hub.HasSubscribers(channel) {
		t.Fatal("expected HasSubscribers to be true right after Subscribe")
	}

	hub.Publish(channel, LiveMatchMessage{Type: "instant", HomeGoals: 1})

	select {
	case msg := <-ch:
		if msg.Type != "instant" || msg.HomeGoals != 1 {
			t.Fatalf("unexpected message: %+v", msg)
		}
	default:
		t.Fatal("expected a message to be immediately available")
	}
}

func TestLiveMatchHub_UnsubscribeStopsDelivery(t *testing.T) {
	hub := NewLiveMatchHub()
	channel := LeagueMatchChannel(1)

	_, unsubscribe := hub.Subscribe(channel)
	unsubscribe()

	if hub.HasSubscribers(channel) {
		t.Fatal("expected no subscribers after unsubscribe")
	}

	// Must not panic or block publishing to a channel nobody listens to any more.
	hub.Publish(channel, LiveMatchMessage{Type: "instant"})
}

func TestLiveMatchHub_PublishNeverBlocksOnFullChannel(t *testing.T) {
	hub := NewLiveMatchHub()
	channel := LeagueMatchChannel(1)

	_, unsubscribe := hub.Subscribe(channel)
	defer unsubscribe()

	// Flood well past the buffer - Publish must drop rather than block.
	for i := 0; i < liveMatchChannelBuffer*3; i++ {
		hub.Publish(channel, LiveMatchMessage{Type: "instant"})
	}
}

func TestLiveMatchHub_LeagueAndCupChannelsAreIsolated(t *testing.T) {
	hub := NewLiveMatchHub()

	// A league match and a cup tie can share the same numeric id (independent Postgres
	// sequences) - the hub must not let one leak into the other's subscribers.
	leagueCh, unsubLeague := hub.Subscribe(LeagueMatchChannel(1))
	defer unsubLeague()
	cupCh, unsubCup := hub.Subscribe(CupTieChannel(1))
	defer unsubCup()

	hub.Publish(LeagueMatchChannel(1), LiveMatchMessage{Type: "instant", HomeGoals: 7})

	select {
	case msg := <-leagueCh:
		if msg.HomeGoals != 7 {
			t.Fatalf("unexpected message on league channel: %+v", msg)
		}
	default:
		t.Fatal("expected the league subscriber to receive the message")
	}

	select {
	case msg := <-cupCh:
		t.Fatalf("cup tie subscriber should not have received the league match's message: %+v", msg)
	default:
	}
}

func TestLiveMatchHub_MultipleSubscribersAllReceive(t *testing.T) {
	hub := NewLiveMatchHub()
	channel := LeagueMatchChannel(1)

	ch1, unsub1 := hub.Subscribe(channel)
	defer unsub1()
	ch2, unsub2 := hub.Subscribe(channel)
	defer unsub2()

	hub.Publish(channel, LiveMatchMessage{Type: "complete", HomeGoals: 2, AwayGoals: 1})

	for _, ch := range []chan LiveMatchMessage{ch1, ch2} {
		select {
		case msg := <-ch:
			if msg.Type != "complete" {
				t.Fatalf("unexpected message: %+v", msg)
			}
		default:
			t.Fatal("expected every subscriber to receive the broadcast")
		}
	}
}
