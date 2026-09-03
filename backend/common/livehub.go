package common

import (
	"strconv"
	"sync"
)

// LiveMatchMessage is one message of a live match's broadcast stream - see engine's
// BroadcastMatchLive, which paces a match's already-computed events/instants out to any
// subscribed WebSocket viewers to give the appearance of a match unfolding in real time.
type LiveMatchMessage struct {
	Type      string        `json:"type"` // "instant", "event", or "complete"
	Instant   *MatchInstant `json:"instant,omitempty"`
	Event     *Event        `json:"event,omitempty"`
	HomeGoals int           `json:"homeGoals"`
	AwayGoals int           `json:"awayGoals"`
}

// liveMatchChannelBuffer is generous enough that a momentarily slow consumer (a WS write in
// progress) doesn't lose a message to the non-blocking send in Publish.
const liveMatchChannelBuffer = 8

// LiveMatchHub fans out a match's live messages to every WebSocket client currently watching
// it. It's created once per DatabaseContext and is safe for concurrent use.
//
// Subscriptions are keyed by an opaque string channel, not a bare id: league match ids and cup
// tie ids are drawn from independent Postgres sequences (see runMatchSimulation's id-space
// warning) and can collide, so callers must namespace their ids - see LeagueMatchChannel and
// CupTieChannel below - rather than passing one in directly.
type LiveMatchHub struct {
	mu   sync.Mutex
	subs map[string]map[chan LiveMatchMessage]bool
}

func NewLiveMatchHub() *LiveMatchHub {
	return &LiveMatchHub{subs: make(map[string]map[chan LiveMatchMessage]bool)}
}

// LeagueMatchChannel and CupTieChannel are the two id namespaces a live channel key can come
// from - use these rather than formatting the string inline, so every caller stays consistent.
func LeagueMatchChannel(matchId int64) string { return "match:" + strconv.FormatInt(matchId, 10) }
func CupTieChannel(tieId int64) string        { return "cup:" + strconv.FormatInt(tieId, 10) }

// Subscribe registers a new listener for channel and returns its message channel plus an
// unsubscribe function the caller must run (typically deferred) once it's done reading.
func (h *LiveMatchHub) Subscribe(channel string) (chan LiveMatchMessage, func()) {
	ch := make(chan LiveMatchMessage, liveMatchChannelBuffer)

	h.mu.Lock()
	if h.subs[channel] == nil {
		h.subs[channel] = make(map[chan LiveMatchMessage]bool)
	}
	h.subs[channel][ch] = true
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		delete(h.subs[channel], ch)
		if len(h.subs[channel]) == 0 {
			delete(h.subs, channel)
		}
		h.mu.Unlock()
	}
	return ch, unsubscribe
}

// HasSubscribers reports whether anyone is currently watching channel - used to skip the
// broadcast pacing loop entirely when nobody is listening.
func (h *LiveMatchHub) HasSubscribers(channel string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs[channel]) > 0
}

// Publish sends msg to every current subscriber of channel. Sends are non-blocking and best
// effort: a full channel (a stalled consumer) drops the message rather than stalling the
// broadcast for everyone else - the client can always fall back to the REST match/cup tie
// details endpoint for the final result.
func (h *LiveMatchHub) Publish(channel string, msg LiveMatchMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[channel] {
		select {
		case ch <- msg:
		default:
		}
	}
}
