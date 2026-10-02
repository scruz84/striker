package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// SearchHandler backs the frontend's search page - look up teams, players, coaches, and stadiums
// by (partial) name in one call. A blank/missing query returns empty result lists rather than an
// error, so the page can call it eagerly as the user types.
func SearchHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := strings.TrimSpace(c.Query("q"))
		if query == "" {
			c.JSON(http.StatusOK, common.SearchResults{
				Teams:    []common.TeamSearchResult{},
				Players:  []common.PlayerSearchResult{},
				Coaches:  []common.CoachSearchResult{},
				Stadiums: []common.StadiumSearchResult{},
			})
			return
		}

		results, err := common.Search(dbCtx, query)
		if err != nil {
			log.Errorf("error searching for %q: %s", query, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, results)
	}
}
