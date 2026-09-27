package handlers

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

var liveMatchUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// The frontend is served from a different origin than the API in dev and in the Docker
	// Compose setup (nginx vs the API container), same as the REST API's CORS configuration -
	// this mirrors that rather than restricting the handshake to a single origin.
	CheckOrigin: func(r *http.Request) bool { return true },
}

const liveMatchWriteTimeout = 10 * time.Second

// liveMatchMaxConnectionLifetime caps how long serveLiveMatch will hold a connection open
// waiting for messages that may never come - comfortably above the ~105-real-minute broadcast
// itself (see engine.liveMatchMinuteDelay/engine.liveMatchHalftimeDelay) plus scheduler lag, so it never cuts off a genuinely
// in-progress match, but still bounds the damage of a channel that's stuck for any reason (a
// downed Redis, a bug, a match that somehow never gets a "complete" published) - without this, a
// stuck connection's goroutines and file descriptor stay held indefinitely, and enough of those
// accumulating under real traffic can degrade the rest of the service, not just this one match's
// viewers. The client-side reconnect/give-up logic (useLiveMatch) treats a close here exactly
// like any other dropped connection, so this is a safe, ordinary case for it to handle.
const liveMatchMaxConnectionLifetime = 3 * time.Hour

// LiveMatchHandler streams a league match's events live over a WebSocket - see
// engine.BroadcastMatchLive, which is what actually paces the messages out once the match
// finishes simulating.
func LiveMatchHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid match id"))
			return
		}

		match, err := common.WeekMatchById(dbCtx, matchId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, GetError(0, "match not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		serveLiveMatch(c, dbCtx, common.LeagueMatchChannel(matchId), match.Played, match.HomeGoals, match.AwayGoals)
	}
}

// LiveCupTieHandler is the cup-tie equivalent of LiveMatchHandler - see
// common.CupTieChannel for why cup ties need their own id namespace in the hub.
func LiveCupTieHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		tieId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid cup tie id"))
			return
		}

		tie, err := common.GetCupTieById(dbCtx, tieId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, GetError(0, "cup tie not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		serveLiveMatch(c, dbCtx, common.CupTieChannel(tieId), tie.Played, tie.HomeGoals, tie.AwayGoals)
	}
}

// LiveFriendlyHandler is the friendly-match equivalent of LiveMatchHandler - see
// common.FriendlyMatchChannel, which engine.simulateFriendly already broadcasts to.
func LiveFriendlyHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid match id"))
			return
		}

		match, err := common.GetFriendlyMatchById(dbCtx, matchId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, GetError(0, "friendly not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		serveLiveMatch(c, dbCtx, common.FriendlyMatchChannel(matchId), match.Played, match.HomeGoals, match.AwayGoals)
	}
}

// serveLiveMatch upgrades the connection and streams a match/tie's live broadcast (see
// engine.BroadcastMatchLive) to it, catching the client up on anything already published before
// it connected rather than deciding purely from the played flag whether there's anything left to
// show it - `played` flips true in the DB the instant the match is simulated, well before its
// ~90-real-minute broadcast finishes pacing out to viewers, so a client connecting any time after
// that (the common case - see issue #37) would otherwise be told the match is already "complete"
// with no events at all, even while the broadcast is still very much in progress for everyone
// else watching.
func serveLiveMatch(c *gin.Context, dbCtx *common.DatabaseContext, channel string, played bool, homeGoals int, awayGoals int) {
	conn, err := liveMatchUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("error upgrading live match websocket for %s: %s", channel, err)
		return
	}
	defer conn.Close()

	// Subscribed before reading history so nothing published in between the two calls can be
	// missed (see LiveMatchHub.History's own comment on this ordering requirement).
	ch, unsubscribe := dbCtx.LiveHub.Subscribe(channel)
	defer unsubscribe()

	history := dbCtx.LiveHub.History(channel)
	lastMinute := -1
	for _, msg := range history {
		if msg.Type == "complete" {
			_ = conn.WriteJSON(msg)
			return
		}
		_ = conn.SetWriteDeadline(time.Now().Add(liveMatchWriteTimeout))
		if err := conn.WriteJSON(msg); err != nil {
			return
		}
		if minute := messageMinute(msg); minute > lastMinute {
			lastMinute = minute
		}
	}

	// No broadcast history at all and the match is already played: either its broadcast finished
	// and its history has since expired out of Redis (see liveMatchHistoryTTL), or it was
	// simulated before this ever broadcasts (shouldn't normally happen, but fail safe) - either
	// way there's nothing left to stream, so report the final score directly rather than waiting
	// forever on a channel nobody will ever publish to again.
	if len(history) == 0 && played {
		_ = conn.WriteJSON(common.LiveMatchMessage{Type: "complete", HomeGoals: homeGoals, AwayGoals: awayGoals})
		return
	}

	// The client never sends application messages on this connection - this goroutine only
	// exists to notice when it disconnects (a read error, including the close frame a browser
	// sends when the tab/page goes away) and unblock the write loop below via done.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	maxLifetime := time.NewTimer(liveMatchMaxConnectionLifetime)
	defer maxLifetime.Stop()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// Already delivered as part of the history catch-up above - skip the duplicate that
			// arrives here whenever it was published in the small window between subscribing and
			// reading history.
			if minute := messageMinute(msg); minute >= 0 && minute <= lastMinute {
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(liveMatchWriteTimeout))
			if err := conn.WriteJSON(msg); err != nil {
				return
			}
			if msg.Type == "complete" {
				return
			}
		case <-done:
			return
		case <-maxLifetime.C:
			log.Warnf("live match connection for %s exceeded max lifetime without completing - closing", channel)
			return
		}
	}
}

// messageMinute returns msg's match minute, or -1 for a message type that doesn't carry one (eg.
// "complete") - used by serveLiveMatch to avoid re-delivering a message live that was already
// sent as part of the history catch-up.
func messageMinute(msg common.LiveMatchMessage) int {
	switch {
	case msg.Instant != nil:
		return msg.Instant.Minute
	case msg.Event != nil:
		return msg.Event.Minute
	default:
		return -1
	}
}
