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
		players, err := common.GetFreeAgents(dbCtx)
		if err != nil {
			log.Errorf("error listing free agents: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, players)
	}
}

// SignFreeAgentHandler signs a free agent to the caller's team on a fresh contract, at no fee.
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
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		err = common.SignFreeAgent(dbCtx, playerId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "free agent signed"})
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotFree), errors.Is(err, common.ErrTransferWindowClosed):
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

		offer, err := common.MakeContractRenewalOffer(dbCtx, playerId, team.Id, body.Wage, body.Years)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, offer)
		case errors.Is(err, common.ErrInvalidRenewalOffer):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotOwnedByTeam):
			c.JSON(http.StatusForbidden, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerOnLoan):
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
		case errors.Is(err, common.ErrInsufficientBudget), errors.Is(err, common.ErrPlayerOnLoan):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error releasing player %d, team %d: %s", playerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}
