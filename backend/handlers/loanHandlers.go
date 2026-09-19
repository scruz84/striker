package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type ListPlayerForLoanRequest struct {
	DurationSeasons int `json:"durationSeasons"`
}

// ListPlayerForLoanHandler offers one of the caller's own players out on loan for a fixed
// number of seasons (see common.MinLoanDurationSeasons/MaxLoanDurationSeasons) - free, no fee,
// auto-returned once the duration elapses.
func ListPlayerForLoanHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		var req ListPlayerForLoanRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "durationSeasons is required"))
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

		if err := common.CreateLoanListing(dbCtx, playerId, team.Id, req.DurationSeasons); err != nil {
			switch {
			case errors.Is(err, common.ErrInvalidLoanDuration),
				errors.Is(err, common.ErrPlayerOnLoan),
				errors.Is(err, common.ErrPlayerAlreadyListed),
				errors.Is(err, common.ErrPlayerAlreadyLoanListed),
				errors.Is(err, common.ErrTransferWindowClosed):
				c.JSON(http.StatusConflict, GetError(0, err.Error()))
				return
			}
			log.Errorf("error listing player %d for loan: %s", playerId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "player listed for loan"})
	}
}

// UnlistPlayerForLoanHandler cancels one of the caller's own active loan listings.
func UnlistPlayerForLoanHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		if err := common.CancelLoanListing(dbCtx, playerId, team.Id); err != nil {
			if errors.Is(err, common.ErrPlayerNotLoanListedByTeam) {
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
				return
			}
			log.Errorf("error unlisting player %d from loan: %s", playerId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "loan listing cancelled"})
	}
}

// BrowseLoansHandler lists every active loan listing.
func BrowseLoansHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		listings, err := common.GetActiveLoanListings(dbCtx)
		if err != nil {
			log.Errorf("error getting loan listings: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, listings)
	}
}

// AcceptLoanHandler brings a listed player onto the caller's team for the listing's duration.
func AcceptLoanHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		err = common.AcceptLoan(dbCtx, listingId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "loan accepted"})
		case errors.Is(err, common.ErrLoanListingNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrCannotAcceptOwnLoan), errors.Is(err, common.ErrTeamInsolvent):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		case errors.Is(err, common.ErrTransferWindowClosed):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error accepting loan listing %d for team %d: %s", listingId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}
