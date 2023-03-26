package handlers

import (
	"footmanager/fmcommon"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func UserTeam(dbCtx *fmcommon.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		log.Infof("Starting to get team details of user %d", userId)

		teamDetails, err := fmcommon.GetFullTeamDetailsByUser(*userId, true, dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, teamDetails)
		return
	}
}
