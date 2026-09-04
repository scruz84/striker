package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type SetLineupRequest struct {
	Formation int                     `json:"formation"`
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

		unavailable, err := common.GetUnavailablePlayerIds(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting unavailable players for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		if err := validateLineup(req.Formation, req.Positions, players, unavailable); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		if err := common.SetTeamFormation(dbCtx, team.Id, req.Formation); err != nil {
			log.Errorf("error setting formation for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
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

// validateLineup ensures the submitted lineup matches the chosen formation's required headcount
// at every tactical position (see common.FormationPositionCounts - always 11 players total,
// goalkeeper included) with distinct players that actually belong to the team - without this, a
// manager could otherwise field an empty/understaffed position, duplicate a player, or field an
// opponent's player, since the DB foreign key alone doesn't enforce team ownership. It also
// rejects any player currently suspended or injured (see common.GetUnavailablePlayerIds) - the
// simulation itself would exclude them anyway (see engine.excludeUnavailablePlayers), but
// rejecting it here gives the manager a clear reason instead of a lineup that quietly plays
// short.
func validateLineup(formation int, positions []common.LineupPosition, teamPlayers *[]*common.Player, unavailable map[int64]bool) error {
	if !common.IsValidFormation(formation) {
		return fmt.Errorf("invalid formation %d", formation)
	}
	wantCounts := common.FormationPositionCounts(formation)

	wantTotal := 0
	for _, count := range wantCounts {
		wantTotal += count
	}
	if len(positions) != wantTotal {
		return fmt.Errorf("lineup must have exactly %d positions for this formation, got %d", wantTotal, len(positions))
	}

	rosterIds := make(map[int64]bool, len(*teamPlayers))
	for _, p := range *teamPlayers {
		rosterIds[p.Id] = true
	}

	gotCounts := make(map[int]int, len(wantCounts))
	seenPlayers := make(map[int64]bool, len(positions))
	for _, pos := range positions {
		if pos.Position < common.GOALKEEPER || pos.Position > common.LATERAL_STRIKER_RIGHT {
			return fmt.Errorf("invalid position %d", pos.Position)
		}
		gotCounts[pos.Position]++

		if seenPlayers[pos.PlayerId] {
			return fmt.Errorf("player %d assigned to more than one position", pos.PlayerId)
		}
		seenPlayers[pos.PlayerId] = true

		if !rosterIds[pos.PlayerId] {
			return fmt.Errorf("player %d does not belong to this team", pos.PlayerId)
		}

		if unavailable[pos.PlayerId] {
			return fmt.Errorf("player %d is suspended or injured and unavailable", pos.PlayerId)
		}
	}

	for position, want := range wantCounts {
		if gotCounts[position] != want {
			return fmt.Errorf("formation requires %d players at position %d, got %d", want, position, gotCounts[position])
		}
	}

	return nil
}
