package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// TopScorersLimit bounds how many players a league's leaderboard shows.
const TopScorersLimit = 10

// RecentLeagueTransfersLimit bounds how many transfers the league details/dashboard panel shows.
const RecentLeagueTransfersLimit = 10

// TopScorersHandler returns a league's top goalscorers for the current season - see
// common.GetTopScorers. Public, same reasoning as division standings: league stats aren't
// private.
func TopScorersHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}

		season, err := engine.GetCurrentSeason(dbCtx)
		if err != nil {
			log.Errorf("error getting current season for top scorers: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		scorers, err := common.GetTopScorers(dbCtx, leagueId, season, TopScorersLimit)
		if err != nil {
			log.Errorf("error getting top scorers for league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, scorers)
	}
}

// RecentLeagueTransfersHandler returns a league's most recent completed transfers - see
// common.GetRecentLeagueTransfers. Public, same reasoning as TopScorersHandler.
func RecentLeagueTransfersHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}

		transfers, err := common.GetRecentLeagueTransfers(dbCtx, leagueId, RecentLeagueTransfersLimit)
		if err != nil {
			log.Errorf("error getting recent transfers for league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, transfers)
	}
}
