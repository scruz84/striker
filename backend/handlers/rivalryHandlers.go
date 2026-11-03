package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// TeamRivalriesShown is how many rivals a team page lists (openspec rivalries "Rivalry visibility").
const TeamRivalriesShown = 3

// TeamRivalriesHandler lists a club's top rivals - pairs with a level, highest score first. Built
// only from public results, so any signed-in user may read it.
func TeamRivalriesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}
		rivals, err := common.GetTopRivalries(dbCtx, teamId, TeamRivalriesShown)
		if err != nil {
			log.Errorf("error reading the rivalries of team %d: %s", teamId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, rivals)
	}
}
