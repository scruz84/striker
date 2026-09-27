package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// LiveMatchesOverviewHandler returns how many matches are currently inside their live-broadcast
// window across the whole world, plus one suggestion (prioritized toward the top divisions) to
// jump straight into watching - see common.GetLiveMatchesOverview. Public, same reasoning as
// division standings/top scorers: none of this is specific to the caller's own team.
func LiveMatchesOverviewHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		overview, err := common.GetLiveMatchesOverview(dbCtx)
		if err != nil {
			log.Errorf("error getting live matches overview: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, overview)
	}
}
