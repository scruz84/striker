package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type ChallengeToFriendlyRequest struct {
	OpponentTeamId int64 `json:"opponentTeamId"`
}

// ChallengeToFriendlyHandler proposes a friendly match to another team on behalf of the
// caller's team. If the opponent is unowned, common.ChallengeToFriendly accepts on their
// behalf immediately; otherwise it waits for the opponent's owner to accept/reject.
func ChallengeToFriendlyHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req ChallengeToFriendlyRequest
		if err := c.BindJSON(&req); err != nil || req.OpponentTeamId <= 0 {
			c.JSON(http.StatusBadRequest, GetError(0, "opponentTeamId is required"))
			return
		}

		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		if err := common.ChallengeToFriendly(dbCtx, team.Id, req.OpponentTeamId); err != nil {
			switch {
			case errors.Is(err, common.ErrCannotChallengeOwnTeam),
				errors.Is(err, common.ErrFriendlyChallengeAlreadyExists),
				errors.Is(err, common.ErrFriendlyLimitReached):
				c.JSON(http.StatusConflict, GetError(0, err.Error()))
				return
			}
			log.Errorf("error challenging team %d to a friendly from team %d: %s", req.OpponentTeamId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly challenge sent"})
	}
}

// FriendlyChallengesReceivedHandler lists every pending friendly challenge made to the caller's team.
func FriendlyChallengesReceivedHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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
		challenges, err := common.GetFriendlyChallengesReceived(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting friendly challenges received for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, challenges)
	}
}

// FriendlyChallengesSentHandler lists every pending friendly challenge the caller's team has made.
func FriendlyChallengesSentHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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
		challenges, err := common.GetFriendlyChallengesSent(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting friendly challenges sent for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, challenges)
	}
}

// AcceptFriendlyChallengeHandler agrees to play a pending challenge made to the caller's team.
func AcceptFriendlyChallengeHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		challengeId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid challenge id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		err = common.AcceptFriendlyChallenge(dbCtx, challengeId, team.Id)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly challenge accepted"})
		case errors.Is(err, common.ErrFriendlyChallengeNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrNotChallengeRecipient):
			c.JSON(http.StatusForbidden, GetError(0, err.Error()))
		case errors.Is(err, common.ErrFriendlyLimitReached):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error accepting friendly challenge %d for team %d: %s", challengeId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}

// RejectFriendlyChallengeHandler declines a pending challenge made to the caller's team.
func RejectFriendlyChallengeHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		challengeId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid challenge id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		if err := common.RejectFriendlyChallenge(dbCtx, challengeId, team.Id); err != nil {
			switch {
			case errors.Is(err, common.ErrFriendlyChallengeNotFound):
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
			case errors.Is(err, common.ErrNotChallengeRecipient):
				c.JSON(http.StatusForbidden, GetError(0, err.Error()))
			default:
				log.Errorf("error rejecting friendly challenge %d: %s", challengeId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			}
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly challenge rejected"})
	}
}

// WithdrawFriendlyChallengeHandler cancels one of the caller's own pending friendly challenges.
func WithdrawFriendlyChallengeHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		challengeId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid challenge id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		if err := common.WithdrawFriendlyChallenge(dbCtx, challengeId, team.Id); err != nil {
			switch {
			case errors.Is(err, common.ErrFriendlyChallengeNotFound):
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
			case errors.Is(err, common.ErrNotChallengeSender):
				c.JSON(http.StatusForbidden, GetError(0, err.Error()))
			default:
				log.Errorf("error withdrawing friendly challenge %d: %s", challengeId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			}
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly challenge withdrawn"})
	}
}

// UpcomingFriendliesHandler lists the caller's team's accepted-but-not-yet-played friendlies.
func UpcomingFriendliesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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
		matches, err := common.GetUpcomingFriendlies(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting upcoming friendlies for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, matches)
	}
}

const recentFriendliesLimit = 10

// RecentFriendliesHandler lists the caller's team's most recently played friendlies.
func RecentFriendliesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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
		matches, err := common.GetRecentFriendlies(dbCtx, team.Id, recentFriendliesLimit)
		if err != nil {
			log.Errorf("error getting recent friendlies for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, matches)
	}
}

type setMaxFriendliesPerWeekRequest struct {
	MaxFriendliesPerWeek int `json:"maxFriendliesPerWeek"`
}

// AdminSetMaxFriendliesPerWeekHandler updates the admin-configured cap on friendly matches a
// team may accept per real calendar week (see common.SetMaxFriendliesPerWeek).
func AdminSetMaxFriendliesPerWeekHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setMaxFriendliesPerWeekRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "maxFriendliesPerWeek is required"))
			return
		}

		if err := common.SetMaxFriendliesPerWeek(dbCtx, req.MaxFriendliesPerWeek); err != nil {
			if errors.Is(err, common.ErrInvalidFriendlyLimit) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting max friendlies per week: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "max friendlies per week updated"})
	}
}
