package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type SetLineupRequest struct {
	Positions []common.LineupPosition `json:"positions"`
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
		c.JSON(http.StatusOK, lineup)
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

		if err := validateLineup(req.Positions, players); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		if err := common.SetDefaultLineup(dbCtx, team.Id, req.Positions); err != nil {
			log.Errorf("error setting lineup for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "lineup updated"})
	}
}

// validateLineup ensures the submitted lineup fills each of the 10 tactical positions
// (see common/fmconstants.go) exactly once with distinct players that actually belong to
// the team - without this, a manager could otherwise field an empty position, duplicate a
// player, or field an opponent's player, since the DB foreign key alone doesn't enforce
// team ownership.
func validateLineup(positions []common.LineupPosition, teamPlayers *[]*common.Player) error {
	const numPositions = 10

	if len(positions) != numPositions {
		return fmt.Errorf("lineup must have exactly %d positions, got %d", numPositions, len(positions))
	}

	rosterIds := make(map[int64]bool, len(*teamPlayers))
	for _, p := range *teamPlayers {
		rosterIds[p.Id] = true
	}

	seenPositions := make(map[int]bool, numPositions)
	seenPlayers := make(map[int64]bool, numPositions)
	for _, pos := range positions {
		if pos.Position < common.GOALKEEPER || pos.Position > common.LATERAL_STRIKER_RIGHT {
			return fmt.Errorf("invalid position %d", pos.Position)
		}
		if seenPositions[pos.Position] {
			return fmt.Errorf("position %d assigned more than once", pos.Position)
		}
		seenPositions[pos.Position] = true

		if seenPlayers[pos.PlayerId] {
			return fmt.Errorf("player %d assigned to more than one position", pos.PlayerId)
		}
		seenPlayers[pos.PlayerId] = true

		if !rosterIds[pos.PlayerId] {
			return fmt.Errorf("player %d does not belong to this team", pos.PlayerId)
		}
	}

	return nil
}
