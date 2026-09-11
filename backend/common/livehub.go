package common

import (
	"context"
	"encoding/json"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
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
// progress) doesn't lose a message forwarded from Redis.
const liveMatchChannelBuffer = 8

// liveMatchKeyPrefix namespaces every live-match Redis pub/sub channel from anything else that
// might use this same Redis instance.
const liveMatchKeyPrefix = "livematch:"

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
// Redis instance.
func (h *LiveMatchHub) Publish(channel string, msg LiveMatchMessage) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("error encoding live match message for %s: %s", channel, err)
		return
	}
	if err := h.client.Publish(context.Background(), liveMatchKeyPrefix+channel, payload).Err(); err != nil {
		log.Errorf("error publishing live match message for %s: %s", channel, err)
	}
}
