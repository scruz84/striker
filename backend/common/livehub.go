package common

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

// LiveMatchMessage is one message of a live match's broadcast stream - see engine's
// BroadcastMatchLive, which paces a match's already-computed events/instants out to any
// subscribed WebSocket viewers to give the appearance of a match unfolding in real time.
type LiveMatchMessage struct {
	Type    string        `json:"type"` // "lineups", "instant", "event", "halftime", or "complete"
	Instant *MatchInstant `json:"instant,omitempty"`
	Event   *Event        `json:"event,omitempty"`
	// Seq is a per-channel, strictly increasing sequence number Publish assigns (see its own
	// comment) - a client catching up on History() and then continuing to read live off the
	// channel it Subscribe-d to (see handlers.serveLiveMatch) uses this, not a message's own
	// minute, to tell a genuine duplicate delivered on both paths (the small window between
	// Subscribe and History - see History's comment) from a brand new message that merely shares
	// a minute with one already delivered. A single simulated minute can carry more than one
	// message (an event followed by its instant, or two events back to back - see
	// BroadcastMatchLive), so "have I already seen this minute" is not the same question as "have
	// I already seen this message", and answering the wrong one silently drops the second message
	// of such a minute for a client unlucky enough to reconnect in that exact window.
	Seq int64 `json:"seq"`
	// HomeLineup/AwayLineup/HomePlayers/AwayPlayers are only set on the one "lineups" message
	// published right before the first "instant" (see BroadcastMatchLive) - the starting XI each
	// manager picked, which stays fixed for the whole match (substitutions swap who occupies a
	// position in *PlayersByPosition, never the Lineup.Positions each side kicked off with - see
	// SubstitutionSimulation.Apply). Lets a live viewer compute the same per-player match ratings
	// a played match's report shows (common.ComputeMatchRatings), evolving minute to minute as
	// events come in, without a per-minute payload any heavier than it already was.
	HomeLineup  *Lineup           `json:"homeLineup,omitempty"`
	AwayLineup  *Lineup           `json:"awayLineup,omitempty"`
	HomePlayers *map[int64]Player `json:"homePlayers,omitempty"`
	AwayPlayers *map[int64]Player `json:"awayPlayers,omitempty"`
	HomeGoals   int               `json:"homeGoals"`
	AwayGoals   int               `json:"awayGoals"`
}

// liveMatchChannelBuffer is generous enough that a momentarily slow consumer (a WS write in
// progress) doesn't lose a message forwarded from Redis.
const liveMatchChannelBuffer = 8

// liveMatchKeyPrefix namespaces every live-match Redis pub/sub channel from anything else that
// might use this same Redis instance.
const liveMatchKeyPrefix = "livematch:"

// liveMatchHistoryTTL bounds how long a broadcast's message history sticks around in Redis -
// comfortably longer than the ~105 real-minute broadcast itself (see engine.liveMatchMinuteDelay/
// engine.liveMatchHalftimeDelay) so a viewer connecting even quite late still gets the full catch-up (see History/serveLiveMatch),
// but not indefinitely - this is purely cosmetic replay data, not the source of truth (that's the
// match's own played/events columns).
const liveMatchHistoryTTL = 4 * time.Hour

func historyKey(channel string) string { return liveMatchKeyPrefix + "history:" + channel }

// lastPublishKey namespaces the timestamp Publish records of its own most recent call for
// channel - see LastPublishedAt, which is what RecoverStaleLiveBroadcasts reads to decide whether
// a channel has gone quiet for too long. A separate key from historyKey's list rather than a
// field on the stored message itself, so adding this doesn't touch the message schema every
// existing consumer (the frontend's useLiveMatch, History's callers) already decodes.
func lastPublishKey(channel string) string { return liveMatchKeyPrefix + "lastpublish:" + channel }

// seqKey namespaces the atomic counter Publish uses to assign each channel's messages their own
// strictly increasing Seq (see LiveMatchMessage.Seq) - a separate key from historyKey's list so
// the counter (a single INCR) never has to agree with the list's own length, which would require
// combining both into one round trip to stay race-free.
func seqKey(channel string) string { return liveMatchKeyPrefix + "seq:" + channel }

// LiveMatchHub fans out a match's live messages to every WebSocket client currently watching
// it. Backed by Redis pub/sub rather than an in-process map: the API server (svc, where clients
// subscribe - see handlers.LiveMatchHandler) and the simulation engine (engine, where
// BroadcastMatchLive publishes) run as separate processes - separate containers in
// docker-compose, and still separate *DatabaseContext/*LiveMatchHub instances even when both
// modes run in one combined process (see cmd/striker.go) - so a plain in-memory hub local to
// one process could never be reached by a subscriber registered in the other. Redis is already
// a shared dependency both processes connect to (see NewRedisCache), so publishing through it
// instead makes delivery actually work regardless of which process does which job.
type LiveMatchHub struct {
	client *redis.Client
}

func NewLiveMatchHub() *LiveMatchHub {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	pwd := os.Getenv("REDIS_PASSWORD")
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	client := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: pwd,
		DB:       db,
	})
	return &LiveMatchHub{client: client}
}

// Ping reports whether the underlying Redis connection is reachable - used by tests (in this
// package and others, like engine's BroadcastMatchLive tests) that need the real pub/sub round
// trip and should skip rather than fail when no Redis is available to test against.
func (h *LiveMatchHub) Ping() error {
	return h.client.Ping(context.Background()).Err()
}

// LeagueMatchChannel, CupTieChannel and FriendlyMatchChannel are the id namespaces a live
// channel key can come from - use these rather than formatting the string inline, so every
// caller stays consistent.
func LeagueMatchChannel(matchId int64) string   { return "match:" + strconv.FormatInt(matchId, 10) }
func CupTieChannel(tieId int64) string          { return "cup:" + strconv.FormatInt(tieId, 10) }
func FriendlyMatchChannel(matchId int64) string { return "friendly:" + strconv.FormatInt(matchId, 10) }

// Subscribe registers a new listener for channel and returns its message channel plus an
// unsubscribe function the caller must run (typically deferred) once it's done reading.
func (h *LiveMatchHub) Subscribe(channel string) (chan LiveMatchMessage, func()) {
	pubsub := h.client.Subscribe(context.Background(), liveMatchKeyPrefix+channel)
	out := make(chan LiveMatchMessage, liveMatchChannelBuffer)

	go func() {
		defer close(out)
		for redisMsg := range pubsub.Channel() {
			var msg LiveMatchMessage
			if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
				log.Errorf("error decoding live match message on %s: %s", channel, err)
				continue
			}
			select {
			case out <- msg:
			default:
			}
		}
	}()

	unsubscribe := func() {
		_ = pubsub.Close()
	}
	return out, unsubscribe
}

// HasSubscribers reports whether anyone is currently watching channel, on any process connected
// to this Redis instance. Not used to gate BroadcastMatchLive (see its comment for why a
// snapshot-then-decide check like this one would race a subscriber arriving right after it),
// but kept as a small, independently useful/testable piece of the hub's API.
func (h *LiveMatchHub) HasSubscribers(channel string) bool {
	key := liveMatchKeyPrefix + channel
	counts, err := h.client.PubSubNumSub(context.Background(), key).Result()
	if err != nil {
		log.Errorf("error checking live match subscribers for %s: %s", channel, err)
		return false
	}
	return counts[key] > 0
}

// Publish sends msg to every current subscriber of channel, on any process connected to this
// Redis instance. It also records msg into channel's history (see History) so a viewer who
// connects after msg was published - the common case for anyone checking in on a match already
// under way - can still be caught up on everything they missed instead of only seeing whatever
// happens to be published from the moment they connect onward.
func (h *LiveMatchHub) Publish(channel string, msg LiveMatchMessage) {
	ctx := context.Background()
	sKey := seqKey(channel)
	// Assigned from its own atomic counter (INCR), not the history list's own length: BroadcastMatchLive
	// publishes every message for a match from a single goroutine, so this is already strictly
	// increasing in publish order without needing to agree with a separate RPush's return value.
	if seq, err := h.client.Incr(ctx, sKey).Result(); err != nil {
		log.Errorf("error assigning live match sequence for %s: %s", channel, err)
	} else {
		msg.Seq = seq
		if err := h.client.Expire(ctx, sKey, liveMatchHistoryTTL).Err(); err != nil {
			log.Errorf("error setting live match sequence TTL for %s: %s", channel, err)
		}
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("error encoding live match message for %s: %s", channel, err)
		return
	}
	hKey := historyKey(channel)
	if err := h.client.RPush(ctx, hKey, payload).Err(); err != nil {
		log.Errorf("error recording live match history for %s: %s", channel, err)
	} else if err := h.client.Expire(ctx, hKey, liveMatchHistoryTTL).Err(); err != nil {
		log.Errorf("error setting live match history TTL for %s: %s", channel, err)
	}
	// Recorded alongside history, same TTL - see LastPublishedAt/lastPublishKey.
	if err := h.client.Set(ctx, lastPublishKey(channel), time.Now().UnixNano(), liveMatchHistoryTTL).Err(); err != nil {
		log.Errorf("error recording last-publish time for %s: %s", channel, err)
	}
	if err := h.client.Publish(ctx, liveMatchKeyPrefix+channel, payload).Err(); err != nil {
		log.Errorf("error publishing live match message for %s: %s", channel, err)
	}
}

// History returns every message published for channel so far, in order - see Publish. Callers
// should Subscribe before calling History (never the reverse): any message not yet visible in
// the returned slice is then guaranteed to still arrive on the channel Subscribe returned, so
// nothing published in between the two calls can be missed.
func (h *LiveMatchHub) History(channel string) []LiveMatchMessage {
	raw, err := h.client.LRange(context.Background(), historyKey(channel), 0, -1).Result()
	if err != nil {
		log.Errorf("error reading live match history for %s: %s", channel, err)
		return nil
	}
	messages := make([]LiveMatchMessage, 0, len(raw))
	for _, payload := range raw {
		var msg LiveMatchMessage
		if err := json.Unmarshal([]byte(payload), &msg); err != nil {
			log.Errorf("error decoding live match history message on %s: %s", channel, err)
			continue
		}
		messages = append(messages, msg)
	}
	return messages
}

// LastMessage returns the most recently published message for channel, if any - the same source
// of truth a live viewer's own connection (handlers.serveLiveMatch) ultimately reads off, used by
// GetLiveMatchesOverview's dashboard suggestion so it reports the broadcast's actual current
// state (minute, score, or "complete") instead of estimating one from elapsed real time (see
// CurrentMatchMinute), which can drift from the real broadcast whenever it stalls, lags, or has
// already finished. Reads only the list's tail (LRange ... -1 -1) rather than the whole history,
// since every other caller of that already-published data is History's own, catch-up use case.
func (h *LiveMatchHub) LastMessage(channel string) (LiveMatchMessage, bool) {
	raw, err := h.client.LRange(context.Background(), historyKey(channel), -1, -1).Result()
	if err != nil {
		log.Errorf("error reading last live match message for %s: %s", channel, err)
		return LiveMatchMessage{}, false
	}
	if len(raw) == 0 {
		return LiveMatchMessage{}, false
	}
	var msg LiveMatchMessage
	if err := json.Unmarshal([]byte(raw[0]), &msg); err != nil {
		log.Errorf("error decoding last live match message for %s: %s", channel, err)
		return LiveMatchMessage{}, false
	}
	return msg, true
}

// LastPublishedAt returns when Publish was last called for channel, if ever - see
// RecoverStaleLiveBroadcasts, which uses this (rather than LastMessage's own content) to decide
// whether a played match's broadcast has gone silent for longer than LiveMatchStaleTimeout,
// including the case where nothing has ever been published for it at all (ok is false).
func (h *LiveMatchHub) LastPublishedAt(channel string) (time.Time, bool) {
	raw, err := h.client.Get(context.Background(), lastPublishKey(channel)).Result()
	if err == redis.Nil {
		return time.Time{}, false
	}
	if err != nil {
		log.Errorf("error reading last-publish time for %s: %s", channel, err)
		return time.Time{}, false
	}
	nanos, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		log.Errorf("error parsing last-publish time for %s: %s", channel, err)
		return time.Time{}, false
	}
	return time.Unix(0, nanos), true
}
