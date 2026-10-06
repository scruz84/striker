package handlers

import (
	"errors"
	"fmt"
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
		stuckMatches, err := common.CountStuckSimulations(dbCtx)
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

		schedule := common.GetMatchSchedule(dbCtx)
		maxFriendliesPerWeek, err := common.GetMaxFriendliesPerWeek(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		friendlyScheduleWeekdays := common.GetFriendlyScheduleWeekdays(dbCtx)
		promotionRelegationConfig := common.GetPromotionRelegationConfig(dbCtx)
		freeAgentInjectionsPerWeek, err := common.GetFreeAgentInjectionsPerWeek(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		offerNegotiationWindowMinutes, err := common.GetOfferNegotiationWindowMinutes(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.AdminOverview{
			Season:                        season,
			UserCount:                     userCount,
			TeamCount:                     teamCount,
			LeagueCount:                   leagueCount,
			MatchesTotal:                  matchesTotal,
			MatchesUnplayed:               matchesUnplayed,
			StuckMatches:                  stuckMatches,
			ActiveCups:                    len(activeCups),
			TransferWindowOpen:            common.IsTransferWindowOpen(dbCtx),
			MatchScheduleWeekdaysMask:     schedule.WeekdaysMask,
			MatchScheduleHour:             schedule.Hour,
			MatchScheduleMinute:           schedule.Minute,
			MaxFriendliesPerWeek:          maxFriendliesPerWeek,
			FriendlyScheduleWeekdays:      friendlyScheduleWeekdays,
			PromotionSlotsPerGroup:        promotionRelegationConfig.PromotionSlotsPerGroup,
			RelegationSlotsPerGroup:       promotionRelegationConfig.RelegationSlotsPerGroup,
			FreeAgentInjectionsPerWeek:    freeAgentInjectionsPerWeek,
			OfferNegotiationWindowMinutes: offerNegotiationWindowMinutes,
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

type finishStuckMatchesResponse struct {
	LeagueMatches int `json:"leagueMatches"`
	Friendlies    int `json:"friendlies"`
}

// AdminFinishStuckMatchesHandler re-simulates every league match and friendly left stuck at
// in_simulation = true (played = false) - rows a simulation pass claimed but never finished
// simulating, most likely because the server crashed mid-run. The normal scheduler never
// revisits these on its own (see engine.FinishStuckSimulations), so this is the only way to
// recover them short of a direct database fix.
func AdminFinishStuckMatchesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("admin-triggered stuck match recovery")
		result := engine.FinishStuckSimulations(dbCtx)
		c.JSON(http.StatusOK, finishStuckMatchesResponse{
			LeagueMatches: result.LeagueMatches,
			Friendlies:    result.Friendlies,
		})
	}
}

// AdminPayWagesNowHandler force-charges every team's current wage bill on demand - wages are
// tied to league match cadence now (see engine.PayMatchdayWages), not a recurring scheduler, so
// this is purely a manual testing convenience for seeing a wage payment happen without waiting
// for a league's next round to simulate.
func AdminPayWagesNowHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("admin-triggered manual wage payment")
		engine.PayAllTeamsWagesNow(dbCtx)
		c.JSON(http.StatusOK, common.HttpResponse{Message: "wages paid"})
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

// AdminInjectFreeAgentsNowHandler force-runs the same free agent injection the scheduler already
// runs every real calendar week (see engine/scheduler.go), using the currently admin-configured
// count (see common.GetFreeAgentInjectionsPerWeek) - a manual override/testing convenience, same
// as AdminPayWagesNowHandler.
func AdminInjectFreeAgentsNowHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		count, err := common.GetFreeAgentInjectionsPerWeek(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		log.Infof("admin-triggered manual free agent injection (%d players)", count)
		if err := common.InjectFreeAgents(dbCtx, count); err != nil {
			log.Errorf("error injecting free agents: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "free agents injected"})
	}
}

type setFreeAgentInjectionsPerWeekRequest struct {
	FreeAgentInjectionsPerWeek int `json:"freeAgentInjectionsPerWeek"`
}

// AdminSetFreeAgentInjectionsPerWeekHandler updates the admin-configured count of brand-new
// players injected into the free agent pool each real calendar week (see
// common.SetFreeAgentInjectionsPerWeek).
func AdminSetFreeAgentInjectionsPerWeekHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setFreeAgentInjectionsPerWeekRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "freeAgentInjectionsPerWeek is required"))
			return
		}

		if err := common.SetFreeAgentInjectionsPerWeek(dbCtx, req.FreeAgentInjectionsPerWeek); err != nil {
			if errors.Is(err, common.ErrInvalidFreeAgentInjectionCount) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting free agent injections per week: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "free agent injections per week updated"})
	}
}

type setOfferNegotiationWindowRequest struct {
	OfferNegotiationWindowMinutes int `json:"offerNegotiationWindowMinutes"`
}

// AdminSetOfferNegotiationWindowHandler updates the admin-configured minimum time (in minutes) a
// received transfer offer must sit pending before it can be accepted (see
// common.SetOfferNegotiationWindowMinutes).
func AdminSetOfferNegotiationWindowHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setOfferNegotiationWindowRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "offerNegotiationWindowMinutes is required"))
			return
		}

		if err := common.SetOfferNegotiationWindowMinutes(dbCtx, req.OfferNegotiationWindowMinutes); err != nil {
			if errors.Is(err, common.ErrInvalidOfferNegotiationWindow) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting offer negotiation window: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "offer negotiation window updated"})
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

type injectAllTeamsBudgetRequest struct {
	Amount      int64  `json:"amount" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type injectAllTeamsBudgetResponse struct {
	Message       string `json:"message"`
	TeamsAffected int64  `json:"teamsAffected"`
}

// AdminInjectAllTeamsBudgetHandler applies the same budget adjustment to every existing team at
// once - positive to grant funds league-wide, negative to deduct from everyone - logged to each
// team's own ledger like any other transaction so the injection isn't invisible to any owner.
func AdminInjectAllTeamsBudgetHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req injectAllTeamsBudgetRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "amount and description are required"))
			return
		}

		affected, err := common.AdjustAllTeamsBudget(dbCtx, req.Amount, common.TRANSACTION_ADMIN_MASS_INJECTION, req.Description)
		if err != nil {
			log.Errorf("error applying mass budget injection to all teams: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, injectAllTeamsBudgetResponse{
			Message:       fmt.Sprintf("budget adjusted for %d teams", affected),
			TeamsAffected: affected,
		})
	}
}

type setTeamAiLevelRequest struct {
	AiLevel int `json:"aiLevel" binding:"required,min=1,max=3"`
}

// AdminSetTeamAiLevelHandler hand-tunes one CPU team's AI difficulty (see
// common.AiLevelCaretaker/Manager/Director) - has no effect once a team has a real owner, since
// every AI action is gated on owner_id being unset regardless of this value.
func AdminSetTeamAiLevelHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}

		var req setTeamAiLevelRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "aiLevel is required and must be 1-3"))
			return
		}

		if err := common.SetTeamAiLevel(dbCtx, teamId, req.AiLevel); err != nil {
			log.Errorf("error setting AI level for team %d: %s", teamId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		engine.InvalidateTeamDetailsCache(dbCtx, teamId)

		c.JSON(http.StatusOK, common.HttpResponse{Message: "AI level updated"})
	}
}

type setTransferWindowRequest struct {
	Open bool `json:"open"`
}

// AdminSetTransferWindowHandler opens or closes the global transfer/loan market - see
// common.IsTransferWindowOpen for how every listing/buy/accept endpoint checks it.
func AdminSetTransferWindowHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setTransferWindowRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "open is required"))
			return
		}

		if err := common.SetTransferWindowOpen(dbCtx, req.Open); err != nil {
			log.Errorf("error setting transfer window open=%v: %s", req.Open, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "transfer window updated"})
	}
}

type setMatchScheduleRequest struct {
	WeekdaysMask int `json:"weekdaysMask"`
	Hour         int `json:"hour"`
	Minute       int `json:"minute"`
}

// AdminSetMatchScheduleHandler updates the recurring weekday/time-of-day schedule used to date
// newly generated league fixtures (see engine.ComputeFixtureDates) - e.g. "Saturdays at 14:00"
// or "every day at 16:00" - and immediately re-dates the unplayed matches of every league that
// doesn't have its own per-league override (see engine.RescheduleUnplayedMatchesForLeagues),
// rather than only affecting fixtures generated at the next season rollover (issue #67). A
// league with its own override is left alone, same as it would be for newly generated fixtures.
func AdminSetMatchScheduleHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setMatchScheduleRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "weekdaysMask, hour and minute are required"))
			return
		}

		schedule := common.MatchSchedule{WeekdaysMask: req.WeekdaysMask, Hour: req.Hour, Minute: req.Minute}
		if err := common.SetMatchSchedule(dbCtx, schedule); err != nil {
			if errors.Is(err, common.ErrInvalidMatchSchedule) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting match schedule: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		leagueIds, err := common.GetAllLeagueIds(dbCtx)
		if err != nil {
			log.Errorf("error listing leagues to reschedule after a default schedule change: %s", err)
		} else {
			overrides, err := common.GetLeagueScheduleOverrides(dbCtx)
			if err != nil {
				log.Errorf("error loading league schedule overrides to reschedule after a default schedule change: %s", err)
				overrides = nil
			}
			leaguesWithoutOverride := make([]int64, 0, len(leagueIds))
			for _, leagueId := range leagueIds {
				if _, hasOverride := overrides[leagueId]; !hasOverride {
					leaguesWithoutOverride = append(leaguesWithoutOverride, leagueId)
				}
			}
			engine.RescheduleUnplayedMatchesForLeagues(dbCtx, leaguesWithoutOverride, schedule)
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "match schedule updated"})
	}
}

type setPromotionRelegationRequest struct {
	PromotionSlotsPerGroup  int `json:"promotionSlotsPerGroup"`
	RelegationSlotsPerGroup int `json:"relegationSlotsPerGroup"`
}

// AdminSetPromotionRelegationHandler updates how many teams move between adjacent tiers at the
// next season rollover (see engine.ComputePromotionRelegation,
// common.SetPromotionRelegationConfig). Only affects a rollover that hasn't happened yet - one
// already applied doesn't get undone.
func AdminSetPromotionRelegationHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setPromotionRelegationRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "promotionSlotsPerGroup and relegationSlotsPerGroup are required"))
			return
		}

		config := common.PromotionRelegationConfig{
			PromotionSlotsPerGroup:  req.PromotionSlotsPerGroup,
			RelegationSlotsPerGroup: req.RelegationSlotsPerGroup,
		}
		if err := common.SetPromotionRelegationConfig(dbCtx, config); err != nil {
			if errors.Is(err, common.ErrInvalidPromotionRelegationConfig) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting promotion/relegation config: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "promotion/relegation slots updated"})
	}
}
