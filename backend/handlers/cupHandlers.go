package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type CupDetails struct {
	Cup  common.Cup      `json:"cup"`
	Ties []common.CupTie `json:"ties"`
}

// CupDetailsHandler returns one cup's bracket (every tie, across every round played so far).
func CupDetailsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		cupId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid cup id"))
			return
		}

		cup, err := common.GetCup(dbCtx, cupId)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "cup not found"))
			return
		}

		ties, err := common.GetCupTies(dbCtx, cupId)
		if err != nil {
			log.Errorf("error getting ties for cup %d: %s", cupId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, CupDetails{Cup: *cup, Ties: ties})
	}
}

type LatestCupResponse struct {
	CupId int64 `json:"cupId"`
}

// LatestCupHandler points callers at the most recently created cup for a country, so the
// frontend doesn't need to know a specific cup id up front.
func LatestCupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		country := c.Param("country")
		cupId, err := common.GetLatestCupForCountry(dbCtx, country)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "no cup found for country"))
			return
		}
		c.JSON(http.StatusOK, LatestCupResponse{CupId: cupId})
	}
}

// CupTieEventsHandler returns one tie's events (goals/cards/injuries) - the tie's score and
// participants are already available from CupDetailsHandler's bracket view.
func CupTieEventsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		tieId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid tie id"))
			return
		}

		events, err := common.GetCupTieEvents(dbCtx, tieId)
		if err != nil {
			log.Errorf("error getting events for cup tie %d: %s", tieId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, events)
	}
}
