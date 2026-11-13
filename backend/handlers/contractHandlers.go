package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// BrowseFreeAgentsHandler lists every unowned player, best-skilled first - anyone can sign one
// via SignFreeAgentHandler.
func BrowseFreeAgentsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		listings, err := common.GetFreeAgentListings(dbCtx)
		if err != nil {
			log.Errorf("error listing free agents: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, listings)
	}
}

// SignFreeAgentRequest is SignFreeAgentHandler's request body - the wage and contract length the
// team is offering. The player decides on the spot (see common.SignFreeAgent), same as a
// contract renewal offer; a decline is a normal 200 response (Accepted == false), not an error.
type SignFreeAgentRequest struct {
	Wage  int `json:"wage"`
	Years int `json:"years"`
	// ReleaseClause is an optional release clause for the new contract (openspec/specs/
	// release-clauses).
	ReleaseClause *int64 `json:"releaseClause,omitempty"`
}

// SignFreeAgentHandler proposes wage/contract-length terms to sign a free agent to the caller's
// team - see common.SignFreeAgent for the transfer premium a recently-released player's last
// club may be owed. The response is always a common.FreeAgentSigningResult, whether the player
// accepted or not.
func SignFreeAgentHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		playerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid player id"))
			return
		}
		var body SignFreeAgentRequest
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid signing offer"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		result, err := common.SignFreeAgentWithClause(dbCtx, playerId, team.Id, body.Wage, body.Years, body.ReleaseClause)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, result)
		case errors.Is(err, common.ErrInvalidFreeAgentOffer), errors.Is(err, common.ErrReleaseClauseTooLow):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotFree), errors.Is(err, common.ErrTransferWindowClosed),
			errors.Is(err, common.ErrInsufficientBudget), errors.Is(err, common.ErrTeamInsolvent):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error signing free agent %d for team %d: %s", playerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}

// RenewalOfferRequest is RenewContractHandler's request body - the wage and duration the team
// is proposing. The player decides on the spot (see common.MakeContractRenewalOffer); a decline
// is a normal 200 response (common.ContractRenewalOffer.Accepted == false), not an error, since
// the team is expected to see why and try again with different terms.
type RenewalOfferRequest struct {
	Wage  int `json:"wage"`
	Years int `json:"years"`
	// ReleaseClause is an optional release clause for the new contract; none clears the old one.
	ReleaseClause *int64 `json:"releaseClause,omitempty"`
}

// RenewContractHandler proposes new terms for one of the caller's own players' contracts. The
// response is always a common.ContractRenewalOffer, whether the player accepted or not.
func RenewContractHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		playerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid player id"))
			return
		}
		var body RenewalOfferRequest
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid renewal offer"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		offer, err := common.MakeContractRenewalOfferWithClause(dbCtx, playerId, team.Id, body.Wage, body.Years, body.ReleaseClause)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, offer)
		case errors.Is(err, common.ErrInvalidRenewalOffer), errors.Is(err, common.ErrReleaseClauseTooLow):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotOwnedByTeam):
			c.JSON(http.StatusForbidden, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerOnLoan), errors.Is(err, common.ErrPlayerPreContracted):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error making renewal offer for player %d, team %d: %s", playerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}

// ReleasePlayerHandler terminates one of the caller's own players' contracts early, paying the
// severance cost out of the team's budget.
func ReleasePlayerHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		playerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid player id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		err = common.ReleasePlayer(dbCtx, playerId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "player released"})
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotOwnedByTeam):
			c.JSON(http.StatusForbidden, GetError(0, err.Error()))
		case errors.Is(err, common.ErrInsufficientBudget), errors.Is(err, common.ErrPlayerOnLoan), errors.Is(err, common.ErrPlayerPreContracted):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error releasing player %d, team %d: %s", playerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}
