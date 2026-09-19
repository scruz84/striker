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
	// Insolvent mirrors common.IsTeamSolvent - true once the budget has crossed -common.MaxDebt,
	// at which point buying, signing, and borrowing on loan are all blocked (see ErrTeamInsolvent).
	// Computed here rather than duplicating MaxDebt as a frontend constant, so the two can never
	// drift out of sync.
	Insolvent bool `json:"insolvent"`
	// MaxDebt mirrors common.MaxDebt - exposed so the frontend's cash-flow forecast can compute
	// how far budget can still fall (to -MaxDebt) without hardcoding the ceiling a second time.
	MaxDebt int64 `json:"maxDebt"`
	// SponsorDeal is one of the common.SponsorDeal* constants - see SetSponsorDealHandler.
	SponsorDeal common.SponsorDeal `json:"sponsorDeal"`
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

		sponsorDeal, err := common.GetSponsorDeal(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting sponsor deal for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, FinancesResponse{
			Budget: budget, Transactions: transactions, Stadium: stadium,
			Insolvent:   budget < -common.MaxDebt,
			MaxDebt:     common.MaxDebt,
			SponsorDeal: sponsorDeal,
		})
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

type setSponsorDealRequest struct {
	Deal int `json:"deal"`
}

// SetSponsorDealHandler lets the caller's team switch between the sponsor deals (see
// common.SponsorDeal) - the tradeoff between a higher guaranteed per-match payment and a bigger
// bonus for winning/drawing takes effect from the next match either side plays.
func SetSponsorDealHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		var req setSponsorDealRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "deal is required"))
			return
		}

		if err := common.SetSponsorDeal(dbCtx, team.Id, common.SponsorDeal(req.Deal)); err != nil {
			if errors.Is(err, common.ErrInvalidSponsorDeal) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting sponsor deal for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "sponsor deal updated"})
	}
}

// UpgradeStadiumHandler spends the caller's team's budget to schedule an expansion of its own
// stadium by a fixed increment - see common.UpgradeStadium for the cost formula and construction
// delay.
func UpgradeStadiumHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		stadium, err := common.UpgradeStadium(dbCtx, team.Id)
		if err != nil {
			if errors.Is(err, common.ErrInsufficientBudget) || errors.Is(err, common.ErrStadiumUpgradeInProgress) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error upgrading stadium for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, stadium)
	}
}
