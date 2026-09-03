package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// AdminOverviewHandler returns a snapshot of the game's overall state for the commissioner
// dashboard - counts, not full listings, so it stays cheap regardless of how many
// users/teams/matches exist.
func AdminOverviewHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		season, err := engine.GetCurrentSeason(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		matchesTotal, matchesUnplayed, err := common.GetSeasonMatchCounts(dbCtx, season)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		userCount, err := common.CountUsers(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		teamCount, err := common.CountTeams(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		leagueCount, err := common.CountLeagues(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		activeCups, err := common.GetActiveCups(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.AdminOverview{
			Season:          season,
			UserCount:       userCount,
			TeamCount:       teamCount,
			LeagueCount:     leagueCount,
			MatchesTotal:    matchesTotal,
			MatchesUnplayed: matchesUnplayed,
			ActiveCups:      len(activeCups),
		})
	}
}

// AdminListUsersHandler lists every registered user with their team and admin status.
func AdminListUsersHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := common.GetAllUsersWithTeams(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

// AdminSimulateNowHandler force-runs the same match and cup tie simulation pass the scheduler
// already runs every minute (see engine/scheduler.go) - a manual override/testing convenience
// rather than something normal operation depends on.
func AdminSimulateNowHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("admin-triggered manual simulation pass")
		engine.SimulateMatches(dbCtx)
		engine.SimulateCupTies(dbCtx)
		engine.AdvanceCups(dbCtx)
		c.JSON(http.StatusOK, common.HttpResponse{Message: "simulation pass triggered"})
	}
}

// AdminAdvanceSeasonHandler force-checks whether the current season is complete and, if so,
// rolls it over immediately instead of waiting for the scheduler's next tick (every 5 minutes -
// see engine/scheduler.go). It's a no-op if matches remain unplayed.
func AdminAdvanceSeasonHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("admin-triggered season advance check")
		engine.AdvanceSeasonIfComplete(dbCtx)
		c.JSON(http.StatusOK, common.HttpResponse{Message: "season advance check triggered"})
	}
}

type adjustBudgetRequest struct {
	Amount      int64  `json:"amount" binding:"required"`
	Description string `json:"description" binding:"required"`
}

// AdminAdjustBudgetHandler applies a manual correction to a team's budget - positive to grant
// funds, negative to deduct them - logged to that team's ledger like any other transaction so
// the adjustment isn't invisible to the team's owner.
func AdminAdjustBudgetHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}

		var req adjustBudgetRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "amount and description are required"))
			return
		}

		if err := common.AdjustTeamBudget(dbCtx, teamId, req.Amount, common.TRANSACTION_ADMIN_ADJUSTMENT, req.Description); err != nil {
			log.Errorf("error applying admin budget adjustment to team %d: %s", teamId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "budget adjusted"})
	}
}
