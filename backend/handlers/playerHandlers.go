package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
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

		if history, err := common.GetPlayerTransactions(dbCtx, playerId, PlayerHistoryLimit); err != nil {
			log.Errorf("error loading transaction history for player %d: %s", playerId, err)
		} else {
			details.History = history
		}

		c.JSON(http.StatusOK, details)
	}
}
