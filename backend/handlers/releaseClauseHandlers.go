package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// PayReleaseClauseRequest is PayReleaseClauseHandler's body: the player, the personal terms
// offered to him and an optional release clause for his new contract.
type PayReleaseClauseRequest struct {
	PlayerId      int64  `json:"playerId"`
	Wage          int    `json:"wage"`
	Years         int    `json:"years"`
	ReleaseClause *int64 `json:"releaseClause,omitempty"`
}

// PayReleaseClauseHandler pays another club's player's release clause and signs him at once
// (openspec/specs/release-clauses "Paying a clause"). The response is always a
// common.ReleaseClauseResult: a refusal of the personal terms is a normal 200 (Accepted == false).
func PayReleaseClauseHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		var req PayReleaseClauseRequest
		if err := c.BindJSON(&req); err != nil || req.PlayerId <= 0 {
			c.JSON(http.StatusBadRequest, GetError(0, "playerId, wage and years are required"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}
		result, err := common.PayReleaseClause(dbCtx, req.PlayerId, team.Id, req.Wage, req.Years, req.ReleaseClause)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, result)
		case errors.Is(err, common.ErrInvalidPersonalTerms), errors.Is(err, common.ErrReleaseClauseTooLow):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrNoReleaseClause), errors.Is(err, common.ErrReleaseClauseLockedIn),
			errors.Is(err, common.ErrInsufficientBudget), errors.Is(err, common.ErrPlayerOnLoan),
			errors.Is(err, common.ErrPlayerPreContracted), errors.Is(err, common.ErrCannotOfferOnOwnPlayer),
			errors.Is(err, common.ErrPlayerIsFreeAgent), errors.Is(err, common.ErrTeamInsolvent),
			errors.Is(err, common.ErrTransferWindowClosed):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error paying the release clause of player %d: %s", req.PlayerId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}
