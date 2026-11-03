package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// The player watchlist endpoints (openspec player-watchlist): each acts on the caller's own club
// only, and no other endpoint ever reads who follows whom.

// GetWatchlistHandler returns the caller's club's followed players with their market status.
func GetWatchlistHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		team, ok := watchlistTeam(c, dbCtx)
		if !ok {
			return
		}
		entries, err := common.GetWatchlist(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error reading the watchlist of team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, entries)
	}
}

// WatchPlayerHandler follows a player: 400 for one of the caller's own players, 409 once the
// watchlist holds common.MaxWatchlistSize players, 404 for an unknown player.
func WatchPlayerHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		team, ok := watchlistTeam(c, dbCtx)
		if !ok {
			return
		}
		playerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid player id"))
			return
		}
		switch err := common.WatchPlayer(dbCtx, team.Id, playerId); {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "player followed"})
		case errors.Is(err, common.ErrCannotWatchOwnPlayer):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		case errors.Is(err, common.ErrWatchlistFull):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		default:
			log.Errorf("error following player %d for team %d: %s", playerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}

// UnwatchPlayerHandler stops following a player (a no-op for one not followed).
func UnwatchPlayerHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		team, ok := watchlistTeam(c, dbCtx)
		if !ok {
			return
		}
		playerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid player id"))
			return
		}
		if err := common.UnwatchPlayer(dbCtx, team.Id, playerId); err != nil {
			log.Errorf("error unfollowing player %d for team %d: %s", playerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "player unfollowed"})
	}
}

// watchlistTeam is the caller's club, answering 403 / 404 itself when there is none.
func watchlistTeam(c *gin.Context, dbCtx *common.DatabaseContext) (*common.Team, bool) {
	userId := GetUserIdFromCookie(c)
	if userId == nil {
		c.JSON(http.StatusForbidden, nil)
		return nil, false
	}
	team, err := common.GetTeamByOwner(*userId, dbCtx)
	if err != nil {
		c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
		return nil, false
	}
	return team, true
}
