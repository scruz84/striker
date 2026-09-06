package handlers

import (
	"errors"
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

type renameTeamRequest struct {
	Name string `json:"name"`
}

// RenameTeamHandler lets the caller rename their own team - but only once ever, see
// common.RenameTeam. There's no admin override to reset that flag; it's a one-shot deal by
// design (the default RandomTeamName is meant to be good enough to keep, with one do-over for
// whoever wants something different).
func RenameTeamHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		var req renameTeamRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "name is required"))
			return
		}

		if err := common.RenameTeam(dbCtx, team.Id, *userId, req.Name); err != nil {
			if errors.Is(err, common.ErrInvalidTeamName) || errors.Is(err, common.ErrTeamNameAlreadySet) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error renaming team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "team renamed"})
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
