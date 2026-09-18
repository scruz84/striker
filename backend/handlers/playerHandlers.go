package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
)

// PlayerHistoryLimit bounds how many past transactions a player's page shows.
const PlayerHistoryLimit = 20

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

		details := common.PlayerDetails{Player: *player, BestPosition: common.BestCorePosition(*player)}

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

		if stats, err := common.GetPlayerCareerStats(dbCtx, playerId); err != nil {
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

		// A rostered player's wage, contract, and financial transaction history are the owning
		// team's business, not any visitor's - see common.Player.RedactFinancials. A free agent
		// (TeamId == 0) has no current team to protect that information for, so it stays visible
		// to everyone, same as on the transfer market's free agent listing.
		isOwner := true
		if player.TeamId != 0 {
			userId := GetUserIdFromCookie(c)
			teamOwner, _ := engine.GetUserOfTeamId(dbCtx, player.TeamId)
			isOwner = userId != nil && teamOwner >= 0 && *userId == teamOwner
		}
		if !isOwner {
			details.RedactFinancials()
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

// TeamLatestEvolutionsHandler lists the caller's own squad's most recent season-end progression
// results (see common.GetLatestPlayerEvolutionsForTeam) - the page a NOTIFICATION_PROGRESSION
// notification links to. Always scoped to the caller's own team; there's no equivalent for
// browsing another team's evolutions.
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
		evolutions, err := common.GetLatestPlayerEvolutionsForTeam(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error loading latest evolutions for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, evolutions)
	}
}
