package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// NationalTeamRoundHandler lists the current international round (see
// common.GetNationalTeamRound): every unplayed national-team match plus those that kicked off in
// the last day, broadcast-safe. When the caller owns a club, each match also carries how many of
// that club's players are in its saved call-up (see common.SetMyPlayersInNationalMatches).
func NationalTeamRoundHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		matches, err := common.GetNationalTeamRound(dbCtx)
		if err != nil {
			log.Errorf("error getting the national-team round: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if userId := GetUserIdFromCookie(c); userId != nil {
			if team, err := common.GetTeamByOwner(*userId, dbCtx); err == nil && team != nil {
				if err := common.SetMyPlayersInNationalMatches(dbCtx, team.Id, matches); err != nil {
					log.Errorf("error counting team %d's players in national-team call-ups: %s", team.Id, err)
				}
			}
		}
		c.JSON(http.StatusOK, matches)
	}
}

// maxNationalTeamResultsPage bounds ?page= so a crafted request cannot ask Postgres for an
// absurd OFFSET.
const maxNationalTeamResultsPage = 2000

// NationalTeamResultsHandler serves one page (?page=, from 0) of finished national-team results,
// optionally one country's (?country=) - see common.GetNationalTeamResults. 400 on a malformed
// page or a country with no national team.
func NationalTeamResultsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 0
		if raw := c.Query("page"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 || parsed > maxNationalTeamResultsPage {
				c.JSON(http.StatusBadRequest, GetError(0, "invalid page"))
				return
			}
			page = parsed
		}

		var teamId int64
		if country := c.Query("country"); country != "" {
			if !isCountryCodeShape(country) {
				c.JSON(http.StatusBadRequest, GetError(0, "invalid country"))
				return
			}
			id, err := common.GetNationalTeamId(dbCtx, country)
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusBadRequest, GetError(0, "no national team for that country"))
				return
			}
			if err != nil {
				log.Errorf("error resolving the national team of %s: %s", country, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
				return
			}
			teamId = id
		}

		results, err := common.GetNationalTeamResults(dbCtx, page, teamId)
		if err != nil {
			log.Errorf("error getting national-team results page %d: %s", page, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, results)
	}
}

// isCountryCodeShape rejects anything that does not look like a country code (2-8 lowercase
// letters, digits or dashes; the game's codes are two letters such as "es") before it reaches a
// query.
func isCountryCodeShape(code string) bool {
	if len(code) < 2 || len(code) > 8 {
		return false
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
			return false
		}
	}
	return true
}

// CurrentWorldCupHandler returns the most recently started World Cup - its groups (broadcast-safe
// standings and fixtures, see common.GetWorldCupGroups) plus, once the group stage is done, the
// id of its knockout cup, which the frontend fetches separately via the existing /v1/cups/:id
// endpoint (CupPage already knows how to render any cup's bracket). While no World Cup is
// running it also carries nextWorldCupEstimate (see common.GetNextWorldCupEstimate); worldCup is
// null when none has ever been played.
func CurrentWorldCupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		worldCup, err := common.GetLatestWorldCup(dbCtx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Errorf("error getting latest World Cup: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		groups := make([]common.WorldCupGroup, 0)
		if worldCup != nil {
			groups, err = common.GetWorldCupGroups(dbCtx, worldCup.WorldCupId)
			if err != nil {
				log.Errorf("error getting groups for World Cup %d: %s", worldCup.WorldCupId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
				return
			}
		}

		var nextEstimate *time.Time
		if worldCup == nil || worldCup.Completed {
			nextEstimate, err = common.GetNextWorldCupEstimate(dbCtx)
			if err != nil {
				log.Errorf("error estimating the next World Cup: %s", err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"worldCup":             worldCup,
			"groups":               groups,
			"nextWorldCupEstimate": nextEstimate,
		})
	}
}

// WorldCupHistoryHandler lists every World Cup with its champion, newest first - see
// common.GetWorldCupHistory.
func WorldCupHistoryHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		history, err := common.GetWorldCupHistory(dbCtx)
		if err != nil {
			log.Errorf("error getting the World Cup history: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, history)
	}
}
