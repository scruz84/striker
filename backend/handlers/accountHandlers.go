package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
)

// Activity is this replica's per-user throttle for last_seen_at writes and its list of users known
// to be deleted (see ActivityMiddleware).
var Activity = common.NewActivityTracker(common.ActivityWriteInterval)

// clearSessionCookie expires the session cookie with the same attributes it was set with.
func clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(common.COOKIE, "", -1, "/", "", true, true)
}

// ActivityMiddleware runs after the JWT middleware on every authenticated group. It records that
// the user is active (at most once per common.ActivityWriteInterval per replica) and refuses -
// 401 and an expired cookie - a still-valid token whose user has been deleted, so a deleted user's
// session can neither read nor recreate anything.
func ActivityMiddleware(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.Next()
			return
		}
		exists, err := Activity.Seen(*userId, func(id int64) (bool, error) {
			return common.TouchUserActivity(dbCtx, id)
		})
		if err != nil {
			log.Errorf("error recording activity of user %d: %s", *userId, err)
		}
		if !exists {
			clearSessionCookie(c)
			c.AbortWithStatusJSON(http.StatusUnauthorized, GetError(0, "account no longer exists"))
			return
		}
		c.Next()
	}
}

// GetAccountHandler returns the Account page's data.
func GetAccountHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		info, err := common.GetAccountInfo(dbCtx, *userId)
		if errors.Is(err, common.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, GetError(0, "account no longer exists"))
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, info)
	}
}

// DeleteAccountHandler is the self-service "unregister": the typed confirmation must match the
// club's name (or DELETE/BORRAR without a club). The club is released, the user and everything
// that identifies them is erased, and the session cookie is cleared.
func DeleteAccountHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		var body struct {
			Confirm string `json:"confirm"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "confirmation required"))
			return
		}
		info, err := common.GetAccountInfo(dbCtx, *userId)
		if errors.Is(err, common.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, GetError(0, "account no longer exists"))
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if !common.DeleteConfirmationMatches(body.Confirm, info.TeamName) {
			c.JSON(http.StatusBadRequest, GetError(0, "confirmation does not match"))
			return
		}
		if err := eraseAccount(dbCtx, *userId, common.ReleaseReasonSelfDeletion); err != nil {
			if errors.Is(err, common.ErrUserNotFound) {
				c.JSON(http.StatusUnauthorized, GetError(0, "account no longer exists"))
				return
			}
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		clearSessionCookie(c)
		c.Status(http.StatusNoContent)
	}
}

// AdminDeleteUserHandler erases another user's account (erasure requests received by email). An
// admin cannot delete their own account this way - they use the Account page like anyone else.
func AdminDeleteUserHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetId, err := strconv.ParseInt(c.Param(ID), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid user id"))
			return
		}
		if self := GetUserIdFromCookie(c); self != nil && *self == targetId {
			c.JSON(http.StatusForbidden, GetError(0, "use the Account page to delete your own account"))
			return
		}
		if err := eraseAccount(dbCtx, targetId, common.ReleaseReasonAdminDeletion); err != nil {
			if errors.Is(err, common.ErrUserNotFound) {
				c.JSON(http.StatusNotFound, GetError(0, "user not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// eraseAccount deletes the account and drops everything cached about its club and the user.
func eraseAccount(dbCtx *common.DatabaseContext, userId int64, reason string) error {
	teamId, err := common.DeleteAccount(dbCtx, userId, reason)
	if err != nil {
		return err
	}
	Activity.MarkDeleted(userId)
	if teamId != 0 {
		engine.InvalidateTeamCaches(dbCtx, teamId)
	}
	log.Infof("account %d deleted (%s)", userId, reason)
	return nil
}

// ExportAccountHandler downloads everything stored about the user as a JSON file.
func ExportAccountHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		export, err := common.ExportUserData(dbCtx, *userId)
		if errors.Is(err, common.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, GetError(0, "account no longer exists"))
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		payload, err := json.MarshalIndent(export, "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.Header("Content-Disposition", `attachment; filename="striker-my-data.json"`)
		c.Data(http.StatusOK, "application/json", payload)
	}
}

// AcceptTermsHandler records the user's acceptance of the current Privacy Policy and Terms and
// their confirmation that they are at least common.MinimumAge.
func AcceptTermsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		var body struct {
			Version      int  `json:"version"`
			AgeConfirmed bool `json:"ageConfirmed"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		switch err := common.AcceptTerms(dbCtx, *userId, body.Version, body.AgeConfirmed); {
		case err == nil:
		case errors.Is(err, common.ErrInvalidTermsVersion), errors.Is(err, common.ErrAgeNotConfirmed):
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		case errors.Is(err, common.ErrUserNotFound):
			c.JSON(http.StatusUnauthorized, GetError(0, "account no longer exists"))
			return
		default:
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		loggedUser, err := common.GetUserDetails(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, loggedUser)
	}
}
