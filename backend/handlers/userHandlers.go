package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func UserTeam(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		log.Infof("Starting to get team details of user %d", userId)

		teamDetails, err := common.GetFullTeamDetailsByUser(*userId, true, dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, teamDetails)
		return
	}
}

func Team(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamIdStr := c.Param(ID)
		log.Infof("Starting to get team details from %s", teamIdStr)
		teamId, err := common.StringToInt64(teamIdStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		userId := GetUserIdFromCookie(c)
		teamOwner, _ := engine.GetUserOfTeamId(dbCtx, teamId)
		teamDetails, err := engine.GetTeamDetails(dbCtx, teamOwner >= 0 && *userId == teamOwner, teamId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, teamDetails)
		return
	}
}
