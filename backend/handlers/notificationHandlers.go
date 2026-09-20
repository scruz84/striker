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

// DeleteNotificationHandler removes one of the caller's own notifications.
func DeleteNotificationHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
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

		if err := common.DeleteNotification(dbCtx, notificationId, *userId); err != nil {
			log.Errorf("error deleting notification %d: %s", notificationId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "notification deleted"})
	}
}

// DeleteAllNotificationsHandler clears the caller's entire notification feed.
func DeleteAllNotificationsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		if err := common.DeleteAllNotifications(dbCtx, *userId); err != nil {
			log.Errorf("error deleting all notifications for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "all notifications deleted"})
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

// supportedLanguages mirrors frontend src/i18n/index.ts's SUPPORTED_LANGUAGES.
var supportedLanguages = map[string]bool{"en": true, "es": true}

type SetLanguageRequest struct {
	Language string `json:"language"`
}

// SetUserLanguageHandler records the caller's chosen UI language so external notification
// delivery (email/push) can render in that language too - see common.deliverExternalNotifications.
func SetUserLanguageHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req SetLanguageRequest
		if err := c.BindJSON(&req); err != nil || !supportedLanguages[req.Language] {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid language"))
			return
		}

		if err := common.SetUserLanguage(dbCtx, *userId, req.Language); err != nil {
			log.Errorf("error setting language for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "language updated"})
	}
}

// PushSubscriptionRequest mirrors the PushSubscription object the browser's own
// PushManager.subscribe() resolves to - see frontend src/pushNotifications.ts.
type PushSubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// SubscribeToPushHandler records (or replaces, if this exact endpoint was already subscribed -
// see common.SavePushSubscription) one of the caller's devices/browsers as a Web Push target.
func SubscribeToPushHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req PushSubscriptionRequest
		if err := c.BindJSON(&req); err != nil || req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid push subscription payload"))
			return
		}

		if err := common.SavePushSubscription(dbCtx, *userId, req.Endpoint, req.Keys.P256dh, req.Keys.Auth); err != nil {
			log.Errorf("error saving push subscription for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "push subscription saved"})
	}
}

// UnsubscribeFromPushHandler drops one of the caller's own subscriptions - called when the user
// disables push (see NotificationBell), or the browser itself unsubscribed.
func UnsubscribeFromPushHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req struct {
			Endpoint string `json:"endpoint"`
		}
		if err := c.BindJSON(&req); err != nil || req.Endpoint == "" {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid endpoint"))
			return
		}

		if err := common.DeletePushSubscriptionByEndpoint(dbCtx, *userId, req.Endpoint); err != nil {
			log.Errorf("error deleting push subscription for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "push subscription removed"})
	}
}
