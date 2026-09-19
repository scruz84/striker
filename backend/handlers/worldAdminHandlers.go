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

type addDivisionRequest struct {
	Tier        int  `json:"tier" binding:"required"`
	WithMatches bool `json:"withMatches"`
}

// AdminAddDivisionHandler adds one new tier's division(s) to a country that's already active -
// the gap issue #68 describes: today, editing world_config to add a tier (see
// AdminSetWorldConfigHandler) does nothing for any existing country until either a brand new
// country is activated (AdminAddCountryHandler, which creates every tier from scratch) or every
// league/team is wiped and recreated (AdminRecreateLeaguesHandler). This adds just the missing
// tier to an already-running country, without touching anything else, in either of two ways:
//   - WithMatches=true: generate this season's fixtures for the new division right away, then
//     immediately force-simulate as many of its own weeks as the rest of the country has already
//     completed (see common.GetElapsedWeeksForCountry, engine.SimulateMatchesNow) - so it
//     finishes the season in step with everyone else and takes part in this season's
//     promotion/relegation.
//   - WithMatches=false: create the teams only. It sits out this season entirely (no fixtures at
//     all) and is picked up automatically at the next season rollover, the same moment every
//     other league gets its fresh fixtures generated (see engine.AdvanceSeasonIfComplete calling
//     GenerateMatches) - engine.ComputePromotionRelegation already reads tiers straight from the
//     leagues table, so no further plumbing is needed for it to join the promotion/relegation
//     pyramid from that season on.
func AdminAddDivisionHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := strings.ToLower(strings.TrimSpace(c.Param("code")))
		if !common.IsKnownCountry(code) {
			c.JSON(http.StatusBadRequest, GetError(0, "unknown country code"))
			return
		}

		var req addDivisionRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "tier is required"))
			return
		}

		active, err := common.GetActiveCountries(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		isActive := false
		for _, existing := range active {
			if existing == code {
				isActive = true
				break
			}
		}
		if !isActive {
			c.JSON(http.StatusBadRequest, GetError(0, "country is not active yet"))
			return
		}

		var tierConfig *common.TierConfig
		for _, tc := range common.GetWorldConfig(dbCtx) {
			if tc.Tier == req.Tier {
				found := tc
				tierConfig = &found
				break
			}
		}
		if tierConfig == nil {
			c.JSON(http.StatusBadRequest, GetError(0, "tier not found in the current world structure - add it there first"))
			return
		}

		existingLeagues, err := common.GetLeagueIdsForCountryTier(dbCtx, code, req.Tier)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if len(existingLeagues) > 0 {
			c.JSON(http.StatusConflict, GetError(0, "this division already exists for this country"))
			return
		}

		leagueIds := createLeague(dbCtx, code, tierConfig.Name, tierConfig.Tier, tierConfig.Groups)
		createTeamsForTier(dbCtx, code, leagueIds, tierConfig.Tier, tierConfig.TeamsPerLeague)

		if req.WithMatches {
			addDivisionMatchesNow(dbCtx, code, req.Tier, leagueIds)
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "division added"})
	}
}

// addDivisionMatchesNow generates the new division's fixtures and force-simulates whichever of
// its own weeks the rest of country has already completed this season - see
// AdminAddDivisionHandler's WithMatches=true case.
func addDivisionMatchesNow(dbCtx *common.DatabaseContext, country string, tier int, leagueIds []int64) {
	if err := engine.GenerateMatchesForLeagues(dbCtx, leagueIds); err != nil {
		log.Errorf("error generating fixtures for new division (country %s tier %d): %s", country, tier, err)
		return
	}

	season, err := engine.GetCurrentSeason(dbCtx)
	if err != nil {
		log.Errorf("error reading current season to backfill new division (country %s tier %d): %s", country, tier, err)
		return
	}

	elapsedWeeks, err := common.GetElapsedWeeksForCountry(dbCtx, country, season)
	if err != nil {
		log.Errorf("error computing elapsed weeks for country %s: %s", country, err)
		return
	}
	if elapsedWeeks <= 0 {
		return
	}

	matches, err := common.GetMatchesForLeaguesUpToWeek(dbCtx, leagueIds, season, elapsedWeeks)
	if err != nil {
		log.Errorf("error loading backfill matches for new division (country %s tier %d): %s", country, tier, err)
		return
	}
	log.Infof("backfilling %d week(s) (%d match(es)) for new division (country %s tier %d)",
		elapsedWeeks, len(*matches), country, tier)
	engine.SimulateMatchesNow(dbCtx, *matches)
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
// common.SetLeagueSchedule - and immediately re-dates that league's own unplayed matches onto it
// (see engine.RescheduleUnplayedMatches), rather than only affecting fixtures generated at the
// next season rollover (issue #67).
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
		if err := engine.RescheduleUnplayedMatches(dbCtx, leagueId, schedule); err != nil {
			log.Errorf("error rescheduling unplayed matches for league %d: %s", leagueId, err)
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "league schedule updated"})
	}
}

// AdminClearLeagueScheduleHandler removes a league's schedule override, reverting it to the
// app-wide default, and immediately re-dates that league's unplayed matches onto that default
// (issue #67).
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
		if err := engine.RescheduleUnplayedMatches(dbCtx, leagueId, common.GetMatchSchedule(dbCtx)); err != nil {
			log.Errorf("error rescheduling unplayed matches for league %d: %s", leagueId, err)
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
