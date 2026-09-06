package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
)

// AdminGetWorldConfigHandler returns the tier/group/teams-per-league structure used the next
// time leagues are (re)created - see common.GetWorldConfig.
func AdminGetWorldConfigHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"tiers": common.GetWorldConfig(dbCtx)})
	}
}

type setWorldConfigRequest struct {
	Tiers []common.TierConfig `json:"tiers" binding:"required"`
}

// AdminSetWorldConfigHandler saves a new tier/group/teams-per-league structure. It only takes
// effect the next time leagues are (re)created (AdminRecreateLeaguesHandler or
// AdminAddCountryHandler) - it never touches any existing league/team itself.
func AdminSetWorldConfigHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setWorldConfigRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "tiers is required"))
			return
		}
		if err := common.ValidateWorldConfig(req.Tiers); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		if err := common.SetWorldConfig(dbCtx, req.Tiers); err != nil {
			log.Errorf("error saving world config: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "world structure saved"})
	}
}

// AdminRecreateLeaguesHandler wipes and regenerates every league/team/match using the current
// world config, keeping registered logins - see RecreateLeaguesKeepingUsers.
func AdminRecreateLeaguesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Warnf("admin-triggered leagues recreate (keeping user accounts)")
		if err := RecreateLeaguesKeepingUsers(dbCtx); err != nil {
			log.Errorf("error recreating leagues: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "leagues recreated"})
	}
}

// AdminListCountriesHandler lists every country the game engine can generate a world for (see
// common.KnownCountries), marking which ones are currently active - for the commissioner's
// "add a country" UI.
func AdminListCountriesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		active, err := common.GetActiveCountries(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		activeSet := make(map[string]bool, len(active))
		for _, code := range active {
			activeSet[code] = true
		}

		type countryStatus struct {
			common.CountryInfo
			Active bool `json:"active"`
		}
		countries := make([]countryStatus, 0, len(common.KnownCountries))
		for _, info := range common.KnownCountries {
			countries = append(countries, countryStatus{CountryInfo: info, Active: activeSet[info.Code]})
		}
		c.JSON(http.StatusOK, countries)
	}
}

type addCountryRequest struct {
	Code string `json:"code" binding:"required"`
}

// AdminAddCountryHandler activates one of the known-but-inactive countries (see
// common.KnownCountries): creates its leagues and teams per the current world config, generates
// this season's fixtures for just those new leagues, and starts its cup - all without touching
// any other country's data.
func AdminAddCountryHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req addCountryRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "code is required"))
			return
		}
		code := strings.ToLower(strings.TrimSpace(req.Code))
		if !common.IsKnownCountry(code) {
			c.JSON(http.StatusBadRequest, GetError(0, "unknown country code"))
			return
		}

		active, err := common.GetActiveCountries(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		for _, existing := range active {
			if existing == code {
				c.JSON(http.StatusConflict, GetError(0, "country is already active"))
				return
			}
		}

		if err := common.InsertCountry(dbCtx, code); err != nil {
			log.Errorf("error activating country %s: %s", code, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		createCountryLeagues(dbCtx, code, common.GetWorldConfig(dbCtx))

		leagueIds, err := common.GetLeagueIdsForCountry(dbCtx, code)
		if err != nil {
			log.Errorf("error listing leagues for newly added country %s: %s", code, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := engine.GenerateMatchesForLeagues(dbCtx, leagueIds); err != nil {
			log.Errorf("error generating fixtures for newly added country %s: %s", code, err)
		}

		season, err := engine.GetCurrentSeason(dbCtx)
		if err != nil {
			log.Errorf("error reading current season to start %s's cup: %s", code, err)
		} else if err := engine.StartCupForCountry(dbCtx, code, season); err != nil {
			log.Errorf("error starting cup for newly added country %s: %s", code, err)
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "country added"})
	}
}

// AdminListLeaguesHandler lists every league with its effective match schedule (a per-league
// override, or the app-wide default) - see common.ListLeaguesForAdmin.
func AdminListLeaguesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagues, err := common.ListLeaguesForAdmin(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, leagues)
	}
}

type setLeagueScheduleRequest struct {
	WeekdaysMask int `json:"weekdaysMask"`
	Hour         int `json:"hour"`
	Minute       int `json:"minute"`
}

// AdminSetLeagueScheduleHandler sets one league's match-day/time override - see
// common.SetLeagueSchedule. Only affects fixtures generated after this call, same as the
// app-wide default (see AdminSetMatchScheduleHandler).
func AdminSetLeagueScheduleHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}

		var req setLeagueScheduleRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "weekdaysMask, hour and minute are required"))
			return
		}

		schedule := common.MatchSchedule{WeekdaysMask: req.WeekdaysMask, Hour: req.Hour, Minute: req.Minute}
		if err := common.ValidateMatchSchedule(schedule); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		if err := common.SetLeagueSchedule(dbCtx, leagueId, schedule); err != nil {
			log.Errorf("error setting schedule for league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "league schedule updated"})
	}
}

// AdminClearLeagueScheduleHandler removes a league's schedule override, reverting it to the
// app-wide default.
func AdminClearLeagueScheduleHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}
		if err := common.ClearLeagueSchedule(dbCtx, leagueId); err != nil {
			log.Errorf("error clearing schedule for league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "league schedule reset to the default"})
	}
}

type setUserAdminRequest struct {
	IsAdmin bool `json:"isAdmin"`
}

// AdminSetUserAdminHandler grants or revokes another user's admin access.
func AdminSetUserAdminHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid user id"))
			return
		}

		var req setUserAdminRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "isAdmin is required"))
			return
		}

		requestingUserId := GetUserIdFromCookie(c)
		if requestingUserId != nil && *requestingUserId == userId && !req.IsAdmin {
			c.JSON(http.StatusBadRequest, GetError(0, "you can't revoke your own admin access"))
			return
		}

		if err := common.SetUserAdmin(dbCtx, userId, req.IsAdmin); err != nil {
			log.Errorf("error setting admin=%v for user %d: %s", req.IsAdmin, userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "user updated"})
	}
}
