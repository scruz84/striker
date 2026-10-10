package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

const nationalTeamFriendliesLimit = 10

// UpcomingNationalTeamFriendliesHandler lists every national team's upcoming friendly - the
// global, any-country counterpart to UpcomingFriendliesHandler, since no logged-in manager's own
// team is ever one side of a national-team fixture.
func UpcomingNationalTeamFriendliesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		matches, err := common.GetUpcomingNationalTeamFriendlies(dbCtx, nationalTeamFriendliesLimit)
		if err != nil {
			log.Errorf("error getting upcoming national-team friendlies: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, matches)
	}
}

// RecentNationalTeamFriendliesHandler is UpcomingNationalTeamFriendliesHandler's played
// counterpart.
func RecentNationalTeamFriendliesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		matches, err := common.GetRecentNationalTeamFriendlies(dbCtx, nationalTeamFriendliesLimit)
		if err != nil {
			log.Errorf("error getting recent national-team friendlies: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, matches)
	}
}

// CurrentWorldCupHandler returns the most recently started World Cup - its groups (standings and
// fixtures) plus, once the group stage is done, the id of its knockout cup (see
// common.GetLatestWorldCup), which the frontend fetches separately via the existing
// /v1/cups/:id endpoint (CupPage already knows how to render any cup's bracket). 404 if no World
// Cup has ever been started.
func CurrentWorldCupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		worldCup, err := common.GetLatestWorldCup(dbCtx)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, GetError(0, "no World Cup found"))
			return
		}
		if err != nil {
			log.Errorf("error getting latest World Cup: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		groups, err := common.GetWorldCupGroups(dbCtx, worldCup.WorldCupId)
		if err != nil {
			log.Errorf("error getting groups for World Cup %d: %s", worldCup.WorldCupId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"worldCup": worldCup,
			"groups":   groups,
		})
	}
}
