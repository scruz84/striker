package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

const RecentTransactionsLimit = 50

type FinancesResponse struct {
	Budget       int64                    `json:"budget"`
	Transactions []common.TeamTransaction `json:"transactions"`
}

// GetFinancesHandler returns the caller's team's budget and recent transaction history.
// Deliberately not part of the general Team/TeamDetails response (which any authenticated
// user can fetch for any team, see handlers.Team) - budget stays private to the owner.
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

		c.JSON(http.StatusOK, FinancesResponse{Budget: budget, Transactions: transactions})
	}
}
