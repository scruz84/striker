package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
)

// TeamHistoryHandler returns a team's season-by-season history (final league position,
// promotion/relegation, cup result) - the "trophy cabinet." Public, like the rest of a team's
// details (see handlers.Team) - a team's past results aren't private the way its budget is.
func TeamHistoryHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}

		history, err := common.GetSeasonHistory(dbCtx, teamId)
		if err != nil {
			log.Errorf("error getting season history for team %d: %s", teamId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, history)
	}
}

// HeadToHeadHandler returns the aggregate record and recent meetings between two teams. Public,
// same reasoning as TeamHistoryHandler above.
func HeadToHeadHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamAId, err := common.StringToInt64(c.Param("teamA"))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}
		teamBId, err := common.StringToInt64(c.Param("teamB"))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}

		headToHead, err := common.GetHeadToHead(dbCtx, teamAId, teamBId)
		if err != nil {
			log.Errorf("error getting head-to-head for teams %d/%d: %s", teamAId, teamBId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, headToHead)
	}
}

// parseSeason reads a season number (a path or query parameter): an integer of at least 1.
func parseSeason(param string) (int, error) {
	season, err := strconv.Atoi(param)
	if err != nil || season < 1 {
		return 0, fmt.Errorf("invalid season")
	}
	return season, nil
}

// LeagueSeasonsHandler returns a league's current season and its past seasons, newest first, for
// the league page's season picker - see common.GetLeagueSeasons. Public, like the standings.
func LeagueSeasonsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}
		seasons, err := common.GetLeagueSeasons(dbCtx, leagueId)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, GetError(0, "league not found"))
			return
		}
		if err != nil {
			log.Errorf("error getting the seasons of league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, seasons)
	}
}

// LeagueSeasonTableHandler returns a league's final table for a past season - see
// engine.GetLeagueSeasonTable. A season with no history rows, the current one included, is 404.
func LeagueSeasonTableHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}
		season, err := parseSeason(c.Param("season"))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		table, err := engine.GetLeagueSeasonTable(dbCtx, leagueId, season)
		if err != nil {
			log.Errorf("error getting season %d of league %d: %s", season, leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if table == nil {
			c.JSON(http.StatusNotFound, GetError(0, "no table for that season"))
			return
		}
		c.JSON(http.StatusOK, table)
	}
}
