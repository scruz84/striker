package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// TeamHistoryHandler returns a team's season-by-season history (final league position,
// promotion/relegation, cup result) - the "trophy cabinet." Public, like the rest of a team's
// details (see handlers.Team) - a team's past results aren't private the way its budget is.
func TeamHistoryHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}

		history, err := common.GetSeasonHistory(dbCtx, teamId)
		if err != nil {
			log.Errorf("error getting season history for team %d: %s", teamId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, history)
	}
}

// HeadToHeadHandler returns the aggregate record and recent meetings between two teams. Public,
// same reasoning as TeamHistoryHandler above.
func HeadToHeadHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamAId, err := common.StringToInt64(c.Param("teamA"))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}
		teamBId, err := common.StringToInt64(c.Param("teamB"))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}

		headToHead, err := common.GetHeadToHead(dbCtx, teamAId, teamBId)
		if err != nil {
			log.Errorf("error getting head-to-head for teams %d/%d: %s", teamAId, teamBId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, headToHead)
	}
}
