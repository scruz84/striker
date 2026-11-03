package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type SetLineupRequest struct {
	Formation     int                         `json:"formation"`
	Positions     []common.LineupPosition     `json:"positions"`
	Substitutions []common.LineupSubstitution `json:"substitutions"`
	// Mentality is -1 (defensive), 0 (balanced) or 1 (attacking); a payload without it saves
	// Balanced.
	Mentality int `json:"mentality"`
	// Orders are the lineup's conditional match orders (at most 3; see common.ValidateOrders). The
	// friendly default lineup carries none, so its handler ignores them.
	Orders []common.LineupOrder `json:"orders"`
	// LineupRoles are the optional captain, penalty taker and free-kick taker (captainId,
	// penaltyTakerId, freeKickTakerId; missing = automatic) - see common.ValidateRoles.
	common.LineupRoles
}

// GetLineupHandler returns the caller's team's default lineup.
func GetLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		lineup, err := common.GetDefaultLineup(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting default lineup for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if !withOwnerOrders(c, lineup, func() ([]common.LineupOrder, error) { return common.GetDefaultLineupOrders(dbCtx, team.Id) }) {
			return
		}
		c.JSON(http.StatusOK, withProjectedReplacements(dbCtx, team.Id, lineup, false))
	}
}

// SetLineupHandler replaces the caller's team's default lineup after validating it.
func SetLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req SetLineupRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid lineup payload"))
			return
		}

		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		players, err := common.GetPlayersFromTeamId(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting players for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		unavailable, err := common.GetUnavailablePlayerIds(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting unavailable players for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		if err := validateLineup(req.Formation, req.Positions, players, unavailable, false); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		if err := validateSubstitutions(req.Positions, req.Substitutions, players, unavailable); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		if err := common.ValidateMentality(req.Mentality); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		if err := common.ValidateRoles(req.Positions, req.LineupRoles, players); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		if err := common.ValidateOrders(req.Positions, req.Orders, players, unavailable); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		if err := common.SetTeamFormation(dbCtx, team.Id, req.Formation); err != nil {
			log.Errorf("error setting formation for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetTeamMentality(dbCtx, team.Id, req.Mentality); err != nil {
			log.Errorf("error setting mentality for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetDefaultLineupRoles(dbCtx, team.Id, false, req.LineupRoles); err != nil {
			log.Errorf("error setting lineup roles for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetDefaultLineup(dbCtx, team.Id, req.Positions); err != nil {
			log.Errorf("error setting lineup for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetDefaultLineupSubstitutions(dbCtx, team.Id, req.Substitutions); err != nil {
			log.Errorf("error setting substitutions for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetDefaultLineupOrders(dbCtx, team.Id, req.Orders); err != nil {
			log.Errorf("error setting conditional orders for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "lineup updated"})
	}
}

// GetFriendlyDefaultLineupHandler returns the caller's team's standing friendly-only default
// lineup (see common.GetDefaultFriendlyLineup) - separate from GetLineupHandler's league/cup
// default so a manager can review/edit each independently.
func GetFriendlyDefaultLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		lineup, err := common.GetDefaultFriendlyLineup(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting default friendly lineup for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, withProjectedReplacements(dbCtx, team.Id, lineup, true))
	}
}

// SetFriendlyDefaultLineupHandler replaces the caller's team's standing friendly-only default
// lineup after the same formation/roster/availability validation SetLineupHandler applies to the
// league/cup default. Scheduled substitutions are the friendly default's own (see
// common.SetDefaultFriendlyLineupSubstitutions), validated with friendly availability rules (only
// an injury bars a player from coming on) and independent of the league/cup set; a missing or
// empty list clears them.
func SetFriendlyDefaultLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req SetLineupRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid lineup payload"))
			return
		}

		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		players, err := common.GetPlayersFromTeamId(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting players for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		unavailable, err := common.GetUnavailablePlayerIds(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting unavailable players for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		if err := validateLineup(req.Formation, req.Positions, players, unavailable, false); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		unavailableForFriendly, err := common.GetUnavailablePlayerIdsForFriendly(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting friendly-unavailable players for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := validateSubstitutions(req.Positions, req.Substitutions, players, unavailableForFriendly); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		if err := common.ValidateMentality(req.Mentality); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		if err := common.ValidateRoles(req.Positions, req.LineupRoles, players); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		if err := common.SetTeamFriendlyFormation(dbCtx, team.Id, req.Formation); err != nil {
			log.Errorf("error setting friendly formation for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetTeamFriendlyMentality(dbCtx, team.Id, req.Mentality); err != nil {
			log.Errorf("error setting friendly mentality for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetDefaultLineupRoles(dbCtx, team.Id, true, req.LineupRoles); err != nil {
			log.Errorf("error setting lineup roles for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetDefaultFriendlyLineup(dbCtx, team.Id, req.Positions); err != nil {
			log.Errorf("error setting friendly lineup for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.SetDefaultFriendlyLineupSubstitutions(dbCtx, team.Id, req.Substitutions); err != nil {
			log.Errorf("error setting friendly substitutions for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "lineup updated"})
	}
}

// withProjectedReplacements fills lineup.ProjectedReplacements for a team owner reading the lineup
// of an upcoming match (or his default): for each unavailable starter, the player the pre-kickoff
// refill would put in his slot if the match kicked off now (see common.ProjectKickoffReplacements).
// forFriendly applies the friendly rules (injuries only - a suspension doesn't bar a friendly).
// It is a courtesy on top of the lineup, so a failure to compute it is logged and the lineup is
// returned as saved.
func withProjectedReplacements(dbCtx *common.DatabaseContext, teamId int64, lineup *common.Lineup, forFriendly bool) *common.Lineup {
	if lineup == nil || len(lineup.Positions) == 0 {
		return lineup
	}
	var unavailable map[int64]bool
	var err error
	if forFriendly {
		unavailable, err = common.GetUnavailablePlayerIdsForFriendly(dbCtx, teamId)
	} else {
		unavailable, err = common.GetUnavailablePlayerIds(dbCtx, teamId)
	}
	if err != nil {
		log.Errorf("error getting unavailable players for team %d projection: %s", teamId, err)
		return lineup
	}
	if len(unavailable) == 0 {
		return lineup
	}
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting players for team %d projection: %s", teamId, err)
		return lineup
	}
	lineup.ProjectedReplacements = common.ProjectKickoffReplacements(lineup, *players, unavailable)
	return lineup
}

// validateLineup and validateSubstitutions apply the shared lineup rules (see
// common.ValidateLineup/common.ValidateSubstitutions, which the engine also runs on a CPU team's
// kickoff pick).
func validateLineup(formation int, positions []common.LineupPosition, teamPlayers *[]*common.Player, unavailable map[int64]bool, enforceAvailability bool) error {
	return common.ValidateLineup(formation, positions, teamPlayers, unavailable, enforceAvailability)
}

func validateSubstitutions(positions []common.LineupPosition, subs []common.LineupSubstitution, teamPlayers *[]*common.Player, unavailable map[int64]bool) error {
	return common.ValidateSubstitutions(positions, subs, teamPlayers, unavailable)
}

// withOwnerOrders fills lineup.Orders with the conditional orders load returns, for the lineup
// GET handlers a team owner opens - a team's orders are never shown to anyone else (see
// common.Lineup.Orders). A failed read answers the request with a 500 and returns false.
func withOwnerOrders(c *gin.Context, lineup *common.Lineup, load func() ([]common.LineupOrder, error)) bool {
	orders, err := load()
	if err != nil {
		log.Errorf("error getting conditional orders for team %d: %s", lineup.TeamId, err)
		c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		return false
	}
	lineup.Orders = orders
	return true
}
