package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// BrowseTrainersHandler lists every unemployed trainer candidate on the open market, best-rated
// first - anyone can hire one via HireTrainerHandler.
func BrowseTrainersHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		trainers, err := common.GetAvailableTrainers(dbCtx)
		if err != nil {
			log.Errorf("error listing available trainers: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, trainers)
	}
}

// TeamTrainerHandler returns the caller's team's current head coach, or null if they haven't
// hired one.
func TeamTrainerHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		trainer, err := common.GetTeamTrainer(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting trainer for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, trainer)
	}
}

// HireTrainerHandler assigns an available trainer to the caller's team, at no signing fee -
// only the wage the team takes on is the cost. Fails if the team already employs one.
func HireTrainerHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		trainerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid trainer id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		err = common.HireTrainer(dbCtx, trainerId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "trainer hired"})
		case errors.Is(err, common.ErrTrainerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrTrainerNotAvailable), errors.Is(err, common.ErrTeamAlreadyHasTrainer):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error hiring trainer %d for team %d: %s", trainerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}

// FireTrainerHandler releases one of the caller's team's own trainer back to the open market
// immediately, at no cost.
func FireTrainerHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		trainerId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid trainer id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		err = common.FireTrainer(dbCtx, trainerId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "trainer fired"})
		case errors.Is(err, common.ErrTrainerNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrTrainerNotOwnedByTeam):
			c.JSON(http.StatusForbidden, GetError(0, err.Error()))
		default:
			log.Errorf("error firing trainer %d for team %d: %s", trainerId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}
