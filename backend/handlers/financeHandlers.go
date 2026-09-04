package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

const RecentTransactionsLimit = 50

type FinancesResponse struct {
	Budget       int64                    `json:"budget"`
	Transactions []common.TeamTransaction `json:"transactions"`
	Stadium      *common.Stadium          `json:"stadium"`
}

// GetFinancesHandler returns the caller's team's budget, recent transaction history, and
// stadium/ticket pricing. Deliberately not part of the general Team/TeamDetails response (which
// any authenticated user can fetch for any team, see handlers.Team) - all of it, capacity
// included, stays private to the owner for now. Showing capacity on the public team page too
// would be a reasonable follow-up, just not built here to keep this change to one endpoint.
func GetFinancesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		budget, err := common.GetTeamBudget(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting budget for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		transactions, err := common.GetTeamTransactions(dbCtx, team.Id, RecentTransactionsLimit)
		if err != nil {
			log.Errorf("error getting transactions for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		stadium, err := common.GetStadium(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting stadium for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, FinancesResponse{Budget: budget, Transactions: transactions, Stadium: stadium})
	}
}

type setTicketPriceRequest struct {
	Price int `json:"price" binding:"required"`
}

// SetTicketPriceHandler lets the caller's team set what it charges per seat for its own home
// matches, within [common.MinTicketPrice, common.MaxTicketPrice].
func SetTicketPriceHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		var req setTicketPriceRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "price is required"))
			return
		}

		if err := common.SetTicketPrice(dbCtx, team.Id, req.Price); err != nil {
			if errors.Is(err, common.ErrInvalidTicketPrice) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting ticket price for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "ticket price updated"})
	}
}
