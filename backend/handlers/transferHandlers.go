package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type ListPlayerRequest struct {
	AskingPrice int64 `json:"askingPrice"`
}

// ListPlayerHandler lists one of the caller's own players for transfer at a fixed price.
func ListPlayerHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		var req ListPlayerRequest
		if err := c.BindJSON(&req); err != nil || req.AskingPrice <= 0 {
			c.JSON(http.StatusBadRequest, GetError(0, "askingPrice must be a positive number"))
			return
		}

		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		player, err := common.GetPlayerById(dbCtx, playerId)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "player not found"))
			return
		}
		if player.TeamId != team.Id {
			c.JSON(http.StatusForbidden, GetError(0, "player does not belong to your team"))
			return
		}

		if err := common.CreateTransferListing(dbCtx, playerId, team.Id, req.AskingPrice); err != nil {
			switch {
			case errors.Is(err, common.ErrPlayerAlreadyListed),
				errors.Is(err, common.ErrPlayerOnLoan),
				errors.Is(err, common.ErrPlayerAlreadyLoanListed):
				c.JSON(http.StatusConflict, GetError(0, err.Error()))
				return
			}
			log.Errorf("error listing player %d for transfer: %s", playerId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "player listed for transfer"})
	}
}

// UnlistPlayerHandler cancels one of the caller's own active listings.
func UnlistPlayerHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		if err := common.CancelTransferListing(dbCtx, playerId, team.Id); err != nil {
			if errors.Is(err, common.ErrPlayerNotListedByTeam) {
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
				return
			}
			log.Errorf("error unlisting player %d: %s", playerId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "listing cancelled"})
	}
}

// BrowseTransfersHandler lists every active transfer listing.
func BrowseTransfersHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		listings, err := common.GetActiveTransferListings(dbCtx)
		if err != nil {
			log.Errorf("error getting transfer listings: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, listings)
	}
}

// BuyTransferHandler buys a listed player at its asking price for the caller's team.
func BuyTransferHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		listingId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid listing id"))
			return
		}

		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		err = common.ExecuteTransfer(dbCtx, listingId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "transfer completed"})
		case errors.Is(err, common.ErrListingNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrCannotBuyOwnPlayer), errors.Is(err, common.ErrInsufficientBudget):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		default:
			log.Errorf("error executing transfer of listing %d to team %d: %s", listingId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}
