package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

// TopScorersLimit bounds how many players a league's leaderboard shows.
const TopScorersLimit = 10

// RecentLeagueTransfersLimit bounds how many transfers the league details/dashboard panel shows.
const RecentLeagueTransfersLimit = 10

// TopScorersHandler returns a league's top goalscorers for the current season - see
// common.GetTopScorers. Public, same reasoning as division standings: league stats aren't
// private.
func TopScorersHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}

		season, err := common.GetCurrentSeasonForLeague(dbCtx, leagueId)
		if err != nil {
			log.Errorf("error getting current season for top scorers: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		scorers, err := common.GetTopScorers(dbCtx, leagueId, season, TopScorersLimit)
		if err != nil {
			log.Errorf("error getting top scorers for league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, scorers)
	}
}

// parseLeagueWeek reads the :id and :week params of the matchday endpoint. Split out so the
// validation is unit-testable without a database.
func parseLeagueWeek(leagueParam, weekParam string) (leagueId int64, week int, err error) {
	leagueId, err = common.StringToInt64(leagueParam)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid league id")
	}
	week, err = strconv.Atoi(weekParam)
	if err != nil || week < 1 {
		return 0, 0, fmt.Errorf("invalid week")
	}
	return leagueId, week, nil
}

// LeagueWeekMatchesHandler returns one matchday's matches of a league's current season (or of the
// season in the optional `season` query parameter, at least 1), in the same shape as
// DivisionDetails' previousWeekMatches - so the league page can browse any matchday.
// Public, same reasoning as TopScorersHandler. A week past the season's last one is an empty list.
func LeagueWeekMatchesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, week, err := parseLeagueWeek(c.Param(ID), c.Param("week"))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		var season int
		if param, ok := c.GetQuery("season"); ok {
			if season, err = parseSeason(param); err != nil {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
		} else if season, err = common.GetCurrentSeasonForLeague(dbCtx, leagueId); err != nil {
			log.Errorf("error getting current season for league week: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		matches, err := engine.GetLeagueWeekMatches(dbCtx, leagueId, season, week)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, matches)
	}
}

// LeagueLiveScoresHandler returns the live scores of every match of a league inside its
// live-broadcast window - see common.GetLeagueLiveScores. Scores are the broadcast's own, never the
// stored result, so (like the other league stats) it is open to any signed-in user. A league with
// nothing in play is an empty list, never null.
func LeagueLiveScoresHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}

		scores, err := common.GetLeagueLiveScores(dbCtx, leagueId)
		if err != nil {
			log.Errorf("error getting live scores for league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, scores)
	}
}

// RecentLeagueTransfersHandler returns a league's most recent completed transfers - see
// common.GetRecentLeagueTransfers. Public, same reasoning as TopScorersHandler.
func RecentLeagueTransfersHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}

		transfers, err := common.GetRecentLeagueTransfers(dbCtx, leagueId, RecentLeagueTransfersLimit)
		if err != nil {
			log.Errorf("error getting recent transfers for league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, transfers)
	}
}

// LeagueLeadersLimit bounds how many players a league leaders list shows.
const LeagueLeadersLimit = 10

// LeagueLeadersHandler returns a league season's leaders in one category - goals, rating, minutes
// or cards (?category=, goals by default) - for the current season or a past one (?season=). See
// common.GetLeagueLeaders. Public, like the top scorers.
func LeagueLeadersHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}
		category := c.DefaultQuery("category", common.LeadersGoals)
		if !common.IsLeadersCategory(category) {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid category"))
			return
		}
		season, ok := optionalSeasonParam(c)
		if !ok {
			return
		}
		if season == 0 {
			if season, err = common.GetCurrentSeasonForLeague(dbCtx, leagueId); err != nil {
				log.Errorf("error getting current season for league %d leaders: %s", leagueId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
				return
			}
		}
		cutoff := common.LiveBroadcastCutoff()
		leaders, err := common.GetLeagueLeaders(dbCtx, leagueId, season, category, LeagueLeadersLimit, &cutoff)
		if err != nil {
			log.Errorf("error getting %s leaders for league %d: %s", category, leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, gin.H{"season": season, "category": category, "leaders": leaders})
	}
}

// TeamSeasonStatsHandler returns a team's squad stats for one season (?season=, its newest season
// with any by default) and the seasons it has any for. See common.GetTeamSeasonStats. Public, like
// the match reports they are totalled from.
func TeamSeasonStatsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}
		season, ok := optionalSeasonParam(c)
		if !ok {
			return
		}
		stats, err := common.GetTeamSeasonStats(dbCtx, teamId, season)
		if err != nil {
			log.Errorf("error getting season stats for team %d: %s", teamId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, stats)
	}
}

// optionalSeasonParam reads the ?season= query parameter: 0 when absent, a positive season
// otherwise. An invalid value answers 400 and returns false.
func optionalSeasonParam(c *gin.Context) (int, bool) {
	raw := c.Query("season")
	if raw == "" {
		return 0, true
	}
	season, err := strconv.Atoi(raw)
	if err != nil || season < 1 {
		c.JSON(http.StatusBadRequest, GetError(0, "invalid season"))
		return 0, false
	}
	return season, true
}

// TeamRecordsHandler returns a club's records (see common.GetClubRecords). Public.
func TeamRecordsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid team id"))
			return
		}
		records, err := common.GetClubRecords(dbCtx, teamId)
		if err != nil {
			log.Errorf("error getting records for team %d: %s", teamId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, records)
	}
}

// LeagueRecordsHandler returns a league's champions and records (see common.GetLeagueRecords).
// Public.
func LeagueRecordsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}
		records, err := common.GetLeagueRecords(dbCtx, leagueId)
		if err != nil {
			log.Errorf("error getting records for league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, records)
	}
}

// LeagueBestXiHandler returns a league's team of the week (?week=, its latest matchday with one by
// default) or of the season (?week=0), for the current season or a past one (?season=). See
// common.GetBestXi. Public.
func LeagueBestXiHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		leagueId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid league id"))
			return
		}
		season, ok := optionalSeasonParam(c)
		if !ok {
			return
		}
		if season == 0 {
			if season, err = common.GetCurrentSeasonForLeague(dbCtx, leagueId); err != nil {
				log.Errorf("error getting current season for league %d best XI: %s", leagueId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
				return
			}
		}
		week := -1
		if raw := c.Query("week"); raw != "" {
			if week, err = strconv.Atoi(raw); err != nil || week < 0 {
				c.JSON(http.StatusBadRequest, GetError(0, "invalid week"))
				return
			}
		}
		xi, err := common.GetBestXi(dbCtx, leagueId, season, week)
		if err != nil {
			log.Errorf("error getting the best XI of league %d: %s", leagueId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, xi)
	}
}
