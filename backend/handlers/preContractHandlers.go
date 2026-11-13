package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// PreContractOfferRequest is MakePreContractOfferHandler's body: the player and the wage and
// years offered for next season.
type PreContractOfferRequest struct {
	PlayerId int64 `json:"playerId"`
	Wage     int   `json:"wage"`
	Years    int   `json:"years"`
	// ReleaseClause is an optional release clause for his new contract.
	ReleaseClause *int64 `json:"releaseClause,omitempty"`
}

// MakePreContractOfferHandler offers another club's player in his last season a contract for next
// season (openspec/specs/pre-contracts). The response is always a common.PreContractOfferResult:
// the player judges the terms at once, so a refusal is a normal 200 (Accepted == false).
func MakePreContractOfferHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		var req PreContractOfferRequest
		if err := c.BindJSON(&req); err != nil || req.PlayerId <= 0 {
			c.JSON(http.StatusBadRequest, GetError(0, "playerId, wage and years are required"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}
		result, err := common.MakePreContractOfferWithClause(dbCtx, req.PlayerId, team.Id, req.Wage, req.Years, req.ReleaseClause)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, result)
		case errors.Is(err, common.ErrInvalidPreContractTerms), errors.Is(err, common.ErrReleaseClauseTooLow):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPreContractNotLastSeason), errors.Is(err, common.ErrPreContractTooEarly),
			errors.Is(err, common.ErrPlayerPreContracted), errors.Is(err, common.ErrPreContractLimit),
			errors.Is(err, common.ErrPreContractAlreadyOffered), errors.Is(err, common.ErrCannotOfferOnOwnPlayer),
			errors.Is(err, common.ErrPlayerIsFreeAgent), errors.Is(err, common.ErrPlayerOnLoan),
			errors.Is(err, common.ErrTeamInsolvent), errors.Is(err, common.ErrTransferWindowClosed):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error making a pre-contract offer on player %d: %s", req.PlayerId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}

// PreContractsReceivedHandler lists the pre-contract offers, pending or signed, on the caller's
// players.
func PreContractsReceivedHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return preContractListHandler(dbCtx, common.GetPreContractOffersReceived)
}

// PreContractsSentHandler lists the pre-contract offers, pending or signed, the caller made.
func PreContractsSentHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return preContractListHandler(dbCtx, common.GetPreContractOffersSent)
}

// WithdrawPreContractHandler withdraws one of the caller's pending pre-contract offers; a signed
// one is binding and cannot be withdrawn.
func WithdrawPreContractHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		offerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid offer id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}
		if err := common.WithdrawPreContractOffer(dbCtx, offerId, team.Id); err != nil {
			if errors.Is(err, common.ErrPreContractNotFound) {
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
				return
			}
			log.Errorf("error withdrawing pre-contract offer %d: %s", offerId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, nil)
	}
}

// PreContractMarketHandler lists the players of other clubs in the last season of their contract,
// the ones the caller can offer a pre-contract (openspec/specs/pre-contracts "Pre-contract market").
func PreContractMarketHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return preContractListHandler(dbCtx, common.GetPreContractMarket)
}

func preContractListHandler[T any](dbCtx *common.DatabaseContext, list func(*common.DatabaseContext, int64) ([]T, error)) gin.HandlerFunc {
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
		entries, err := list(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error listing pre-contract data for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, entries)
	}
}
