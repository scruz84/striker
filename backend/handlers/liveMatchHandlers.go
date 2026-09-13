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

// serveLiveMatch upgrades the connection and either immediately reports the final score (the
// match/tie was already played by the time the client connected) or streams live messages from
// channel until the broadcast completes or the client disconnects.
func serveLiveMatch(c *gin.Context, dbCtx *common.DatabaseContext, channel string, played bool, homeGoals int, awayGoals int) {
	conn, err := liveMatchUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("error upgrading live match websocket for %s: %s", channel, err)
		return
	}
	defer conn.Close()

	if played {
		_ = conn.WriteJSON(common.LiveMatchMessage{Type: "complete", HomeGoals: homeGoals, AwayGoals: awayGoals})
		return
	}

	ch, unsubscribe := dbCtx.LiveHub.Subscribe(channel)
	defer unsubscribe()

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

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
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
		}
	}
}
