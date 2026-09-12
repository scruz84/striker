package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// LeagueDirectoryHandler lists every league in the game world, grouped by country, so any
// logged-in user can browse into a league they don't play in and see its standings/teams (see
// common.GetLeagueDirectory) - not just their own division.
func LeagueDirectoryHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		directory, err := common.GetLeagueDirectory(dbCtx)
		if err != nil {
			log.Errorf("error loading league directory: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, directory)
	}
}
