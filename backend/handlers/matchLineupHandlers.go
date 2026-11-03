package handlers

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// TeamMatchesHandler lists every league match, cup tie, and friendly a team has played or has
// scheduled, oldest first - public in the same sense as TeamHistoryHandler/HeadToHeadHandler,
// since a team's fixture list is not sensitive.
func TeamMatchesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}

		matches, err := common.GetTeamMatches(dbCtx, teamId)
		if err != nil {
			log.Errorf("error getting matches for team %d: %s", teamId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, matches)
	}
}

// errMatchTeamNotFound distinguishes "caller has no team" from a real lookup failure below - see
// writeMatchLookupError. ErrNotMatchParticipant/ErrMatchAlreadyPlayed are the league-match
// equivalents of common.ErrNotFriendlyParticipant/common.ErrFriendlyAlreadyPlayed.
var errMatchTeamNotFound = errors.New("team not found for user")
var ErrNotMatchParticipant = errors.New("your team is not part of this match")
var ErrMatchAlreadyPlayed = errors.New("this match has already been played")

// loadMatchForParticipant fetches one league match and confirms the caller's team is one of its
// two teams - shared by the league lineup handlers, mirroring
// friendlyHandlers.loadFriendlyForParticipant.
func loadMatchForParticipant(dbCtx *common.DatabaseContext, userId int64, matchId int64) (*common.WeekMatch, *common.Team, error) {
	team, err := common.GetTeamByOwner(userId, dbCtx)
	if err != nil {
		return nil, nil, errMatchTeamNotFound
	}
	match, err := common.WeekMatchById(dbCtx, matchId)
	if err != nil {
		return nil, nil, err
	}
	if match == nil {
		return nil, nil, sql.ErrNoRows
	}
	if team.Id != match.Home && team.Id != match.Away {
		return nil, nil, ErrNotMatchParticipant
	}
	return match, team, nil
}

// writeMatchLookupError renders the errors loadMatchForParticipant can return - mirrors
// friendlyHandlers.writeFriendlyLookupError.
func writeMatchLookupError(c *gin.Context, matchId int64, err error) {
	switch {
	case errors.Is(err, errMatchTeamNotFound):
		c.JSON(http.StatusNotFound, GetError(0, err.Error()))
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(http.StatusNotFound, GetError(0, "match not found"))
	case errors.Is(err, ErrNotMatchParticipant):
		c.JSON(http.StatusForbidden, GetError(0, err.Error()))
	default:
		log.Errorf("error loading match %d: %s", matchId, err)
		c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
	}
}

// GetMatchLineupHandler returns the caller's team's saved lineup override for one league match
// (see common.GetMatchLineupOverride), or its current default lineup if it hasn't saved one yet -
// mirrors friendlyHandlers.GetFriendlyLineupHandler.
func GetMatchLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		matchId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid match id"))
			return
		}
		_, team, err := loadMatchForParticipant(dbCtx, *userId, matchId)
		if err != nil {
			writeMatchLookupError(c, matchId, err)
			return
		}

		lineup, hasOverride, err := common.GetMatchLineupOverride(dbCtx, matchId, team.Id)
		if err != nil {
			log.Errorf("error getting match lineup for match %d team %d: %s", matchId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if hasOverride {
			if !withOwnerOrders(c, &lineup, func() ([]common.LineupOrder, error) { return common.GetMatchLineupOrders(dbCtx, matchId, team.Id) }) {
				return
			}
			c.JSON(http.StatusOK, withProjectedReplacements(dbCtx, team.Id, &lineup, false))
			return
		}

		formation, err := common.GetTeamFormation(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting formation for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		defaultLineup, err := common.GetDefaultLineup(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting default lineup for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		defaultLineup.MatchId = matchId
		defaultLineup.Formation = formation
		if !withOwnerOrders(c, defaultLineup, func() ([]common.LineupOrder, error) { return common.GetDefaultLineupOrders(dbCtx, team.Id) }) {
			return
		}
		c.JSON(http.StatusOK, withProjectedReplacements(dbCtx, team.Id, defaultLineup, false))
	}
}

// SetMatchLineupHandler saves the caller's team's lineup override for one league match, along
// with any substitutions planned specifically for it, after the same validation SetLineupHandler
// applies to the default lineup - mirrors friendlyHandlers.SetFriendlyLineupHandler. A team that
// doesn't save an override for this match (or saves one with no substitutions of its own) keeps
// drawing from its team-wide default (see common.GetSubstitutionsForMatch).
func SetMatchLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		matchId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid match id"))
			return
		}
		var req SetLineupRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid lineup payload"))
			return
		}

		match, team, err := loadMatchForParticipant(dbCtx, *userId, matchId)
		if err != nil {
			writeMatchLookupError(c, matchId, err)
			return
		}
		if match.Played {
			c.JSON(http.StatusConflict, GetError(0, ErrMatchAlreadyPlayed.Error()))
			return
		}
		if common.IsLineupLocked(match.MatchDate) {
			c.JSON(http.StatusConflict, GetError(0, common.ErrLineupLocked.Error()))
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
		if err := validateLineup(req.Formation, req.Positions, players, unavailable, true); err != nil {
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
		if err := common.ValidateOrders(req.Positions, req.Orders, players, unavailable); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		if err := common.SetLineupForMatch(dbCtx, matchId, team.Id, req.Formation, req.Mentality, req.Positions, req.Substitutions, req.Orders); err != nil {
			log.Errorf("error setting match lineup for match %d team %d: %s", matchId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "lineup updated"})
	}
}

// CupTieDetailsHandler returns one cup tie's full profile (see common.CupTieDetails) - the cup
// equivalent of MatchDetailsHandler/FriendlyMatchHandler, public in the same sense (only requires
// being logged in, not being a participant), since a played tie's result/lineups are not
// sensitive.
func CupTieDetailsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		tieId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid tie id"))
			return
		}

		details, err := common.GetCupTieDetails(dbCtx, tieId)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, GetError(0, "cup tie not found"))
			return
		}
		if err != nil {
			log.Errorf("error getting cup tie details for tie %d: %s", tieId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, details)
	}
}

// errCupTieTeamNotFound and ErrNotCupTieParticipant/ErrCupTieAlreadyPlayed are the cup-tie
// equivalents of the league-match errors above.
var errCupTieTeamNotFound = errors.New("team not found for user")
var ErrNotCupTieParticipant = errors.New("your team is not part of this cup tie")
var ErrCupTieAlreadyPlayed = errors.New("this cup tie has already been played")

// loadCupTieForParticipant fetches one cup tie and confirms the caller's team is one of its two
// teams - a bye (either team id nil) can never have a participant, so it's rejected the same way
// as a mismatched team. Mirrors loadMatchForParticipant/loadFriendlyForParticipant.
func loadCupTieForParticipant(dbCtx *common.DatabaseContext, userId int64, tieId int64) (*common.CupTie, *common.Team, error) {
	team, err := common.GetTeamByOwner(userId, dbCtx)
	if err != nil {
		return nil, nil, errCupTieTeamNotFound
	}
	tie, err := common.GetCupTieById(dbCtx, tieId)
	if err != nil {
		return nil, nil, err
	}
	if tie.HomeTeamId == nil || tie.AwayTeamId == nil ||
		(team.Id != *tie.HomeTeamId && team.Id != *tie.AwayTeamId) {
		return nil, nil, ErrNotCupTieParticipant
	}
	return tie, team, nil
}

// writeCupTieLookupError renders the errors loadCupTieForParticipant can return - mirrors
// writeMatchLookupError/writeFriendlyLookupError.
func writeCupTieLookupError(c *gin.Context, tieId int64, err error) {
	switch {
	case errors.Is(err, errCupTieTeamNotFound):
		c.JSON(http.StatusNotFound, GetError(0, err.Error()))
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(http.StatusNotFound, GetError(0, "cup tie not found"))
	case errors.Is(err, ErrNotCupTieParticipant):
		c.JSON(http.StatusForbidden, GetError(0, err.Error()))
	default:
		log.Errorf("error loading cup tie %d: %s", tieId, err)
		c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
	}
}

// GetCupTieLineupHandler returns the caller's team's saved lineup override for one cup tie, or
// its current default lineup if it hasn't saved one yet - mirrors GetMatchLineupHandler.
func GetCupTieLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		tieId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid tie id"))
			return
		}
		_, team, err := loadCupTieForParticipant(dbCtx, *userId, tieId)
		if err != nil {
			writeCupTieLookupError(c, tieId, err)
			return
		}

		lineup, hasOverride, err := common.GetCupTieLineup(dbCtx, tieId, team.Id)
		if err != nil {
			log.Errorf("error getting cup tie lineup for tie %d team %d: %s", tieId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if hasOverride {
			if !withOwnerOrders(c, &lineup, func() ([]common.LineupOrder, error) { return common.GetCupTieLineupOrders(dbCtx, tieId, team.Id) }) {
				return
			}
			c.JSON(http.StatusOK, withProjectedReplacements(dbCtx, team.Id, &lineup, false))
			return
		}

		formation, err := common.GetTeamFormation(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting formation for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		defaultLineup, err := common.GetDefaultLineup(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting default lineup for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		defaultLineup.MatchId = tieId
		defaultLineup.Formation = formation
		if !withOwnerOrders(c, defaultLineup, func() ([]common.LineupOrder, error) { return common.GetDefaultLineupOrders(dbCtx, team.Id) }) {
			return
		}
		c.JSON(http.StatusOK, withProjectedReplacements(dbCtx, team.Id, defaultLineup, false))
	}
}

// SetCupTieLineupHandler saves the caller's team's lineup override for one cup tie, along with
// any substitutions planned specifically for it, after the same validation SetLineupHandler
// applies to the default lineup - mirrors SetMatchLineupHandler/
// friendlyHandlers.SetFriendlyLineupHandler.
func SetCupTieLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		tieId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid tie id"))
			return
		}
		var req SetLineupRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid lineup payload"))
			return
		}

		tie, team, err := loadCupTieForParticipant(dbCtx, *userId, tieId)
		if err != nil {
			writeCupTieLookupError(c, tieId, err)
			return
		}
		if tie.Played {
			c.JSON(http.StatusConflict, GetError(0, ErrCupTieAlreadyPlayed.Error()))
			return
		}
		if tie.MatchDate != nil && common.IsLineupLocked(*tie.MatchDate) {
			c.JSON(http.StatusConflict, GetError(0, common.ErrLineupLocked.Error()))
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
		if err := validateLineup(req.Formation, req.Positions, players, unavailable, true); err != nil {
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
		if err := common.ValidateOrders(req.Positions, req.Orders, players, unavailable); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		if err := common.SetCupTieLineup(dbCtx, tieId, team.Id, req.Formation, req.Mentality, req.Positions, req.Substitutions, req.Orders); err != nil {
			log.Errorf("error setting cup tie lineup for tie %d team %d: %s", tieId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "lineup updated"})
	}
}
