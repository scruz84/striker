package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type MakeOfferRequest struct {
	PlayerId int64 `json:"playerId"`
	Amount   int64 `json:"amount"`
	Wage     int   `json:"wage"`
	Years    int   `json:"years"`
}

// MakeOfferHandler proposes a price plus personal terms (wage/contract length) for any player,
// listed or not, on behalf of the caller's team - the "scouting" counterpart to the flat-price
// transfer market. The response is always a common.TransferOfferResult: the player judges the
// personal terms immediately (see common.MakeTransferOffer), so a decline is a normal 200
// (Accepted == false), not an error.
func MakeOfferHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req MakeOfferRequest
		if err := c.BindJSON(&req); err != nil || req.PlayerId <= 0 {
			c.JSON(http.StatusBadRequest, GetError(0, "playerId, amount, wage and years are required"))
			return
		}

		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		result, err := common.MakeTransferOffer(dbCtx, req.PlayerId, team.Id, req.Amount, req.Wage, req.Years)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, result)
			return
		case errors.Is(err, common.ErrInvalidOfferAmount),
			errors.Is(err, common.ErrInvalidPersonalTerms),
			errors.Is(err, common.ErrCannotOfferOnOwnPlayer),
			errors.Is(err, common.ErrPlayerIsFreeAgent),
			errors.Is(err, common.ErrOfferAlreadyExists),
			errors.Is(err, common.ErrPlayerOnLoan),
			errors.Is(err, common.ErrTeamInsolvent),
			errors.Is(err, common.ErrTransferWindowClosed):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
			return
		}
		log.Errorf("error making offer on player %d: %s", req.PlayerId, err)
		c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
	}
}

// OffersReceivedHandler lists every pending offer made on the caller's own players.
func OffersReceivedHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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
		offers, err := common.GetOffersReceived(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting offers received for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, offers)
	}
}

// OffersSentHandler lists every pending offer the caller's team has made.
func OffersSentHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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
		offers, err := common.GetOffersSent(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting offers sent for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, offers)
	}
}

// WithdrawOfferHandler cancels one of the caller's own pending offers.
func WithdrawOfferHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		if err := common.WithdrawOffer(dbCtx, offerId, team.Id); err != nil {
			switch {
			case errors.Is(err, common.ErrOfferNotFound):
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
			case errors.Is(err, common.ErrNotOfferSender):
				c.JSON(http.StatusForbidden, GetError(0, err.Error()))
			default:
				log.Errorf("error withdrawing offer %d: %s", offerId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			}
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "offer withdrawn"})
	}
}

// RejectOfferHandler declines an offer made on one of the caller's own players.
func RejectOfferHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		if err := common.RejectOffer(dbCtx, offerId, team.Id); err != nil {
			switch {
			case errors.Is(err, common.ErrOfferNotFound):
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
			case errors.Is(err, common.ErrNotOfferRecipient):
				c.JSON(http.StatusForbidden, GetError(0, err.Error()))
			default:
				log.Errorf("error rejecting offer %d: %s", offerId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			}
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "offer rejected"})
	}
}

// AcceptOfferHandler accepts an offer made on one of the caller's own players, executing the
// transfer at the offer's amount.
func AcceptOfferHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		err = common.AcceptOffer(dbCtx, offerId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "offer accepted"})
		case errors.Is(err, common.ErrOfferNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrNotOfferRecipient):
			c.JSON(http.StatusForbidden, GetError(0, err.Error()))
		case errors.Is(err, common.ErrOfferNoLongerValid),
			errors.Is(err, common.ErrInsufficientBudget),
			errors.Is(err, common.ErrTransferWindowClosed):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error accepting offer %d for team %d: %s", offerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}
