package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
)

// PlayerHistoryLimit bounds how many past transactions a player's page shows.
const PlayerHistoryLimit = 20

// PlayerRecentMatchesLimit bounds how many of the player's team's recent league matches show on
// their profile page.
const PlayerRecentMatchesLimit = 5

// PlayerRecentCallUpsLimit bounds how many of the player's recent national-team call-ups show on
// their profile page.
const PlayerRecentCallUpsLimit = 10

// PlayerDetailsHandler returns a single player's full profile - base attributes, current team,
// career stats (appearances/goals/cards), current status (on loan, listed, unavailable), and
// transfer history. See common.PlayerDetails.
func PlayerDetailsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		playerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid player id"))
			return
		}

		player, err := common.GetPlayerById(dbCtx, playerId)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, GetError(0, "player not found"))
			return
		}
		if err != nil {
			log.Errorf("error fetching player %d: %s", playerId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		// Best-effort: powers the transfer market's "most viewed" stat (issue #72), never worth
		// failing the page load over.
		if err := common.IncrementPlayerViewCount(dbCtx, playerId); err != nil {
			log.Errorf("error incrementing view count for player %d: %s", playerId, err)
		}

		details := common.PlayerDetails{
			Player:       *player,
			BestPosition: common.BestCorePosition(*player),
			AverageSkill: common.AverageSkill(*player),
		}

		if player.TeamId != 0 {
			if team, err := common.GetTeamById(player.TeamId, dbCtx); err != nil {
				log.Errorf("error fetching team %d for player %d: %s", player.TeamId, playerId, err)
			} else {
				details.TeamName = team.TeamName
				details.DivisionId = team.DivisionId
				details.DivisionName = team.DivisionName
				details.Country = team.Country
			}
		}

		if stats, err := common.GetPlayerCareerStats(dbCtx, playerId, player.TeamId); err != nil {
			log.Errorf("error computing career stats for player %d: %s", playerId, err)
		} else {
			details.PlayerCareerStats = stats
		}

		if status, err := common.GetPlayerStatus(dbCtx, playerId); err != nil {
			log.Errorf("error computing status for player %d: %s", playerId, err)
		} else {
			details.PlayerStatus = status
		}

		if evolution, err := common.GetLatestPlayerEvolution(dbCtx, playerId); err != nil {
			log.Errorf("error loading latest evolution for player %d: %s", playerId, err)
		} else {
			details.LatestEvolution = evolution
		}

		if clubHistory, err := common.GetPlayerClubHistory(dbCtx, playerId); err != nil {
			log.Errorf("error loading club history for player %d: %s", playerId, err)
		} else {
			details.ClubHistory = clubHistory
		}

		if recentMatches, err := common.GetPlayerRecentMatches(dbCtx, playerId, player.TeamId, PlayerRecentMatchesLimit); err != nil {
			log.Errorf("error loading recent matches for player %d: %s", playerId, err)
		} else {
			details.RecentMatches = recentMatches
		}

		if callUps, err := common.GetRecentNationalTeamCallUps(dbCtx, playerId, PlayerRecentCallUpsLimit); err != nil {
			log.Errorf("error loading national-team call-ups for player %d: %s", playerId, err)
		} else {
			details.RecentCallUps = callUps
		}

		// A rostered player's contract length and financial transaction history are the owning
		// team's business, not any visitor's - see common.Player.RedactFinancials. A free agent
		// (TeamId == 0) has no current team to protect that information for, so it stays visible
		// to everyone, same as on the transfer market's free agent listing. Wage is the one
		// exception: it stays visible even on someone else's rostered player (restored right after
		// RedactFinancials zeroes it), giving a visitor a rough sense of what signing this player's
		// team would cost to compete with, same reasoning as AverageSkill above.
		isOwner := true
		if player.TeamId != 0 {
			userId := GetUserIdFromCookie(c)
			teamOwner, _ := engine.GetUserOfTeamId(dbCtx, player.TeamId)
			isOwner = userId != nil && teamOwner >= 0 && *userId == teamOwner
		}
		if !isOwner {
			wage := details.Wage
			details.RedactFinancials()
			details.Wage = wage
			details.History = []common.TeamTransaction{}
		} else if history, err := common.GetPlayerTransactions(dbCtx, playerId, PlayerHistoryLimit); err != nil {
			log.Errorf("error loading transaction history for player %d: %s", playerId, err)
		} else {
			details.History = history
		}

		// A rostered player's skill ratings are likewise only the owning team's business, unless
		// this specific player has been put on the transfer market (details.PlayerStatus was
		// already populated above, so TransferListingAskingPrice reflects that without another
		// query) - a prospective buyer needs to see what they'd be signing. Free agents are exempt
		// too, but RedactSkills already handles that case itself.
		if !isOwner {
			transferListed := details.TransferListingAskingPrice != nil
			details.RedactSkills(transferListed)
			if player.TeamId != 0 && !transferListed {
				details.BestPosition = 0
				details.LatestEvolution = nil
			}
		}

		c.JSON(http.StatusOK, details)
	}
}

// TeamLatestEvolutionsHandler lists the caller's own squad's progression results (see
// engine.updatePlayerSkillsAfterMatch). With a logIds query param (the comma-separated
// player_progression_log ids a NOTIFICATION_PROGRESSION notification's link carries - see
// engine.updateTeamPlayersAfterMatch), it returns exactly those rows: the player(s) who evolved
// after that one specific match, scoped to the caller's own team so one manager can't pull up
// another team's players just by guessing ids (see common.GetPlayerEvolutionsForTeamByLogIds).
// With no logIds (direct navigation, no particular match in mind), it falls back to every current
// squad player's own most recent progression run (see common.GetLatestPlayerEvolutionsForTeam) -
// a browsing view, not a report of any one match. Always scoped to the caller's own team; there's
// no equivalent for browsing another team's evolutions.
func TeamLatestEvolutionsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		if logIdsParam := c.Query("logIds"); logIdsParam != "" {
			logIds, err := parseInt64List(logIdsParam)
			if err != nil {
				c.JSON(http.StatusBadRequest, GetError(0, "invalid logIds"))
				return
			}
			evolutions, err := common.GetPlayerEvolutionsForTeamByLogIds(dbCtx, team.Id, logIds)
			if err != nil {
				log.Errorf("error loading evolutions for team %d log ids: %s", team.Id, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
				return
			}
			c.JSON(http.StatusOK, evolutions)
			return
		}

		evolutions, err := common.GetLatestPlayerEvolutionsForTeam(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error loading latest evolutions for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, evolutions)
	}
}

// parseInt64List parses a comma-separated list of integers (e.g. a logIds query param) into
// int64s, rejecting the whole list if any entry doesn't parse - a malformed id here is a bad
// request, not a reason to silently drop it and return someone else's rows.
func parseInt64List(s string) ([]int64, error) {
	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
