package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

const RecentNotificationsLimit = 50

type NotificationsResponse struct {
	Unread        int                   `json:"unread"`
	Notifications []common.Notification `json:"notifications"`
}

// GetNotificationsHandler returns the caller's recent notifications and unread count.
func GetNotificationsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		notifications, err := common.GetNotifications(dbCtx, *userId, RecentNotificationsLimit)
		if err != nil {
			log.Errorf("error getting notifications for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		unread, err := common.GetUnreadNotificationCount(dbCtx, *userId)
		if err != nil {
			log.Errorf("error getting unread notification count for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, NotificationsResponse{Unread: unread, Notifications: notifications})
	}
}

// MarkNotificationReadHandler marks one of the caller's own notifications as read.
func MarkNotificationReadHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		notificationId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid notification id"))
			return
		}

		if err := common.MarkNotificationRead(dbCtx, notificationId, *userId); err != nil {
			log.Errorf("error marking notification %d read: %s", notificationId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "notification marked as read"})
	}
}

// MarkAllNotificationsReadHandler marks every one of the caller's unread notifications as read.
func MarkAllNotificationsReadHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		if err := common.MarkAllNotificationsRead(dbCtx, *userId); err != nil {
			log.Errorf("error marking all notifications read for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "all notifications marked as read"})
	}
}

// GetNotificationPreferencesHandler returns the caller's email/push notification preferences -
// both default false if they've never set any (see common.GetNotificationPreferences).
func GetNotificationPreferencesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		prefs, err := common.GetNotificationPreferences(dbCtx, *userId)
		if err != nil {
			log.Errorf("error getting notification preferences for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, prefs)
	}
}

// SetNotificationPreferencesHandler updates the caller's email/push notification preferences.
func SetNotificationPreferencesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var prefs common.NotificationPreferences
		if err := c.BindJSON(&prefs); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid preferences payload"))
			return
		}

		if err := common.SetNotificationPreferences(dbCtx, *userId, prefs); err != nil {
			log.Errorf("error setting notification preferences for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "notification preferences updated"})
	}
}
