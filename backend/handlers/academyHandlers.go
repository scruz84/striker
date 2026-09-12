package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// AcademyOverview is what GetAcademyHandler returns: the caller's current cantera pool, plus
// whether a promotion is available right now (see common.CanPromoteFromAcademy) and, if not,
// when the weekly limit resets.
type AcademyOverview struct {
	Players         []*common.Player `json:"players"`
	CanPromote      bool             `json:"canPromote"`
	NextAvailableAt *int64           `json:"nextAvailableAt,omitempty"`
}

// GetAcademyHandler lists the caller's team's current youth academy ("cantera") prospects.
func GetAcademyHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		players, err := common.GetCantera(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error listing academy for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		canPromote, nextAvailableAt, err := common.CanPromoteFromAcademy(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error checking academy promotion eligibility for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		overview := AcademyOverview{Players: players, CanPromote: canPromote}
		if !canPromote {
			ms := nextAvailableAt.UnixMilli()
			overview.NextAvailableAt = &ms
		}
		c.JSON(http.StatusOK, overview)
	}
}

// PromoteFromAcademyHandler promotes one of the caller's academy prospects to the senior squad,
// granting them their first professional contract - at most once per real calendar week per team
// (see common.PromoteFromAcademy).
func PromoteFromAcademyHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		playerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid player id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		err = common.PromoteFromAcademy(dbCtx, playerId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "player promoted from academy"})
		case errors.Is(err, common.ErrPlayerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrPlayerNotInAcademy):
			c.JSON(http.StatusForbidden, GetError(0, err.Error()))
		case errors.Is(err, common.ErrAcademyPromotionAlreadyUsedThisWeek):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error promoting player %d from academy for team %d: %s", playerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}
