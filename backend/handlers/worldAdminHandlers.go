package handlers

import (
	"errors"
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
	// ConfirmDissolve must be true to save a structure that dissolves CPU teams (see
	// refuseUnconfirmedDissolve).
	ConfirmDissolve bool `json:"confirmDissolve"`
}

// dissolveConfirmationResponse is the 409 a structure save answers when it would dissolve CPU
// teams and was not confirmed: the teams each country would lose at its next rollover.
type dissolveConfirmationResponse struct {
	Message  string         `json:"message"`
	Dissolve map[string]int `json:"dissolve"`
}

// checkStructureChange validates tiers against the countries it applies to and, unless confirm is
// set, refuses a change that would dissolve CPU teams with a 409 listing them (see
// openspec/specs/league-pyramid "Confirming a structure that dissolves teams"). It writes the
// response and returns false when the save must not go ahead.
func checkStructureChange(c *gin.Context, tiers []common.TierConfig, counts map[string]common.CountryTeams, confirm bool) bool {
	dissolve, err := common.PlanStructureChange(tiers, counts)
	if err != nil {
		c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
		return false
	}
	if len(dissolve) > 0 && !confirm {
		c.JSON(http.StatusConflict, dissolveConfirmationResponse{
			Message:  "this structure dissolves CPU teams at each country's next rollover; send it again with confirmDissolve to save it",
			Dissolve: dissolve,
		})
		return false
	}
	return true
}

// countryTeamCounts returns the club team counts of the active countries, keeping only those
// keep accepts.
func countryTeamCounts(c *gin.Context, dbCtx *common.DatabaseContext, keep func(country string) bool) (map[string]common.CountryTeams, bool) {
	all, err := common.CountTeamsPerActiveCountry(dbCtx)
	if err != nil {
		log.Errorf("error counting teams per country for a structure check: %s", err)
		c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		return nil, false
	}
	counts := make(map[string]common.CountryTeams, len(all))
	for country, n := range all {
		if keep(country) {
			counts[country] = n
		}
	}
	return counts, true
}

// AdminSetWorldConfigHandler saves a new global tier/group/teams-per-league structure. It never
// touches any existing league/team itself: new worlds and countries are created with it, and every
// active country without its own structure is restructured into it at its own next season
// rollover (see engine.restructureIfNeeded). A structure that breaks the pyramid rule, or has
// fewer places than a country's user teams, is rejected; one that dissolves CPU teams needs
// confirmDissolve (see openspec/specs/league-pyramid).
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
		own, err := common.GetCountryStructures(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		counts, ok := countryTeamCounts(c, dbCtx, func(country string) bool { return own[country] == nil })
		if !ok || !checkStructureChange(c, req.Tiers, counts, req.ConfirmDissolve) {
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

// activeCountryParam reads the :code of an active country, answering 404 for any other.
func activeCountryParam(c *gin.Context, dbCtx *common.DatabaseContext) (string, bool) {
	code := strings.ToLower(c.Param("code"))
	active, err := common.GetActiveCountries(dbCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		return "", false
	}
	for _, a := range active {
		if a == code {
			return code, true
		}
	}
	c.JSON(http.StatusNotFound, GetError(0, "unknown or inactive country"))
	return "", false
}

// AdminGetCountryStructureHandler returns an active country's own structure, or null tiers when it
// follows the global one.
func AdminGetCountryStructureHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		code, ok := activeCountryParam(c, dbCtx)
		if !ok {
			return
		}
		tiers, err := common.GetCountryStructure(dbCtx, code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, gin.H{"tiers": tiers})
	}
}

// AdminSetCountryStructureHandler gives an active country its own structure, checked like the
// global one but against that country only (openspec/specs/league-pyramid "Per-country
// structure"). It takes effect at the country's next rollover.
func AdminSetCountryStructureHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		code, ok := activeCountryParam(c, dbCtx)
		if !ok {
			return
		}
		var req setWorldConfigRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "tiers is required"))
			return
		}
		if err := common.ValidateWorldConfig(req.Tiers); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		counts, ok := countryTeamCounts(c, dbCtx, func(country string) bool { return country == code })
		if !ok || !checkStructureChange(c, req.Tiers, counts, req.ConfirmDissolve) {
			return
		}
		if err := common.SetCountryStructure(dbCtx, code, req.Tiers); err != nil {
			log.Errorf("error saving %s's own structure: %s", code, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "country structure saved"})
	}
}

// AdminClearCountryStructureHandler puts an active country back on the global structure. Going
// back can dissolve CPU teams too, so it takes the same confirmation, as ?confirmDissolve=true.
func AdminClearCountryStructureHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		code, ok := activeCountryParam(c, dbCtx)
		if !ok {
			return
		}
		counts, ok := countryTeamCounts(c, dbCtx, func(country string) bool { return country == code })
		if !ok || !checkStructureChange(c, common.GetWorldConfig(dbCtx), counts, c.Query("confirmDissolve") == "true") {
			return
		}
		if err := common.SetCountryStructure(dbCtx, code, nil); err != nil {
			log.Errorf("error clearing %s's own structure: %s", code, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "country back on the global structure"})
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
			// LeagueKickoffMinute is an active country's tier-1 league kickoff, its local minute of
			// day (see common.GetCountryLeagueKickoff); omitted for an inactive country.
			LeagueKickoffMinute *int `json:"leagueKickoffMinute,omitempty"`
			// Structure is an active country's own structure (see common.GetCountryStructure);
			// omitted when it follows the global one.
			Structure []common.TierConfig `json:"structure,omitempty"`
		}
		own, err := common.GetCountryStructures(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		countries := make([]countryStatus, 0, len(common.KnownCountries))
		for _, info := range common.KnownCountries {
			status := countryStatus{CountryInfo: info, Active: activeSet[info.Code]}
			if status.Active {
				kickoff := common.GetCountryLeagueKickoff(dbCtx, info.Code)
				status.LeagueKickoffMinute = &kickoff
				status.Structure = own[info.Code]
			}
			countries = append(countries, status)
		}
		c.JSON(http.StatusOK, countries)
	}
}

type addCountryRequest struct {
	Code string `json:"code" binding:"required"`
	// KickoffHour/KickoffMinute is the time of day this country's matches are played, in the
	// country's own local time (see common.CountryTimezone) - not the server's/viewer's. Both nil
	// defaults to common.DefaultKickoffHour/DefaultKickoffMinute (14:00 local).
	KickoffHour   *int `json:"kickoffHour"`
	KickoffMinute *int `json:"kickoffMinute"`
}

// AdminAddCountryHandler activates one of the known-but-inactive countries (see
// common.KnownCountries): creates its leagues and teams per the current world config, sets every
// new league's match schedule to kick off at the requested (or default 14:00) local time of day
// (see common.LocalTimeToServerSchedule), generates this season's fixtures for just those new
// leagues, and starts its cup - all without touching any other country's data.
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

		kickoffHour, kickoffMinute := common.DefaultKickoffHour, common.DefaultKickoffMinute
		if req.KickoffHour != nil {
			kickoffHour = *req.KickoffHour
		}
		if req.KickoffMinute != nil {
			kickoffMinute = *req.KickoffMinute
		}
		if kickoffHour < 0 || kickoffHour > 23 || kickoffMinute < 0 || kickoffMinute > 59 {
			c.JSON(http.StatusBadRequest, GetError(0, "kickoff time must be between 00:00 and 23:59"))
			return
		}
		if err := common.ValidateCountryLeagueKickoff(kickoffHour*60+kickoffMinute, len(common.GetWorldConfig(dbCtx))); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
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

		// Gives the newly active country a national team right away rather than waiting for the
		// next service restart's BackfillNationalTeams sweep - see common.CreateNationalTeam.
		if _, err := common.CreateNationalTeam(dbCtx, code); err != nil {
			log.Errorf("error creating national team for newly added country %s: %s", code, err)
		}

		leagueIds, err := common.GetLeagueIdsForCountry(dbCtx, code)
		if err != nil {
			log.Errorf("error listing leagues for newly added country %s: %s", code, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		// The country's leagues follow its league kickoff, tier 1 at the requested local time and
		// every lower tier 15 minutes later (see common.EffectiveLeagueSchedule) - no per-league
		// override is written.
		if err := common.SetCountryLeagueKickoff(dbCtx, code, kickoffHour*60+kickoffMinute, len(common.GetWorldConfig(dbCtx))); err != nil {
			log.Errorf("error storing the league kickoff of newly added country %s: %s", code, err)
		}

		if err := engine.GenerateMatchesForLeagues(dbCtx, leagueIds); err != nil {
			log.Errorf("error generating fixtures for newly added country %s: %s", code, err)
		}

		season, err := engine.GetCurrentSeason(dbCtx, code)
		if err != nil {
			log.Errorf("error reading current season to start %s's cup: %s", code, err)
		} else if err := engine.StartCupForCountry(dbCtx, code, season); err != nil {
			log.Errorf("error starting cup for newly added country %s: %s", code, err)
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "country added"})
	}
}

type setCountryKickoffRequest struct {
	Hour   *int `json:"hour"`
	Minute *int `json:"minute"`
}

// AdminSetCountryKickoffHandler changes an active country's league kickoff - tier 1's local time,
// every lower tier 15 minutes later (see openspec/specs/match-scheduling "Country league kickoff
// setting") - and immediately re-dates the country's unplayed matches of leagues without an
// override onto it (see engine.RescheduleUnplayedMatchesForCountry). The time must leave room for
// the deepest tier, the country's own or the saved structure's, before midnight.
func AdminSetCountryKickoffHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := strings.ToLower(strings.TrimSpace(c.Param("code")))
		var req setCountryKickoffRequest
		if err := c.BindJSON(&req); err != nil || req.Hour == nil || req.Minute == nil {
			c.JSON(http.StatusBadRequest, GetError(0, "hour and minute are required"))
			return
		}
		if !isActiveCountry(dbCtx, code) {
			c.JSON(http.StatusBadRequest, GetError(0, "country is not active"))
			return
		}
		shape, err := common.GetCountryPyramidShape(dbCtx, code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		tierCount := max(shape.TierCount(), len(common.EffectiveStructure(dbCtx, code)))
		if err := common.SetCountryLeagueKickoff(dbCtx, code, *req.Hour*60+*req.Minute, tierCount); err != nil {
			if errors.Is(err, common.ErrInvalidLeagueKickoff) || errors.Is(err, common.ErrLeagueKickoffTooLate) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting %s's league kickoff: %s", code, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if _, err := engine.RescheduleUnplayedMatchesForCountry(dbCtx, code); err != nil {
			log.Errorf("error rescheduling %s's matches after a kickoff change: %s", code, err)
		}
		c.JSON(http.StatusOK, common.HttpResponse{Message: "country kickoff updated"})
	}
}

func isActiveCountry(dbCtx *common.DatabaseContext, code string) bool {
	active, err := common.GetActiveCountries(dbCtx)
	if err != nil {
		return false
	}
	for _, existing := range active {
		if existing == code {
			return true
		}
	}
	return false
}

// AdminRescheduleCountryHandler re-dates every unplayed league match of one active country's
// current season onto its leagues' effective schedules, starting from the next matching slot
// after now - see engine.RescheduleUnplayedMatchesForCountry. For fixtures that ended up dated
// further out than they should be; it doesn't change any schedule setting and doesn't touch
// other countries (unlike re-saving the app-wide default schedule, which re-dates every league
// without its own override in every country).
func AdminRescheduleCountryHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := strings.ToLower(strings.TrimSpace(c.Param("code")))

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
			c.JSON(http.StatusBadRequest, GetError(0, "country is not active"))
			return
		}

		leagues, err := engine.RescheduleUnplayedMatchesForCountry(dbCtx, code)
		if err != nil {
			log.Errorf("error rescheduling unplayed matches for country %s: %s", code, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		log.Infof("admin-triggered reschedule of %s's unplayed matches (%d leagues)", code, leagues)
		c.JSON(http.StatusOK, common.HttpResponse{Message: "country fixtures rescheduled"})
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
//     all) and is picked up automatically at this country's next season rollover, the same
//     moment every other of its leagues gets its fresh fixtures generated (see
//     engine.advanceCountrySeasonIfComplete calling GenerateMatchesForLeagues) -
//     engine.ComputePromotionRelegation already reads tiers straight from the
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
		for _, tc := range common.EffectiveStructure(dbCtx, code) {
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

	season, err := engine.GetCurrentSeason(dbCtx, country)
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

// AdminClearLeagueScheduleHandler removes a league's schedule override, so it follows its
// country's league kickoff for its tier again (see common.EffectiveLeagueSchedule), and
// immediately re-dates that league's unplayed matches onto it (issue #67).
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
		if err := engine.RescheduleUnplayedMatchesForLeagues(dbCtx, []int64{leagueId}); err != nil {
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
