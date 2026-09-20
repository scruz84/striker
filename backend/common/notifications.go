package common

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
)

// Notification is one entry in a user's notification feed (match results, promotion/
// relegation, transfers, cup results - see the NOTIFICATION_* constants). Link is an optional
// in-app route the frontend navigates to on click (see NotifyTeamOwnerWithLink) - empty for
// every notification type that existed before it (their message already says everything
// relevant), so the frontend only needs to treat it as clickable when non-empty. MessageKey/
// MessageParams (empty for a notification created before they existed) let the frontend render
// the message in the viewer's own chosen language via i18next - see NotificationMessage; Message
// itself stays the English text, used as the frontend's fallback when MessageKey is empty and as
// the last resort for push/email delivery (see deliverExternalNotifications, which renders into
// the recipient's stored language - common.GetUserLanguage - when one is set and recognized).
type Notification struct {
	Id            int64          `json:"notificationId"`
	Type          int            `json:"notificationType"`
	Message       string         `json:"message"`
	MessageKey    string         `json:"messageKey,omitempty"`
	MessageParams map[string]any `json:"messageParams,omitempty"`
	Link          string         `json:"link,omitempty"`
	Read          bool           `json:"read"`
	CreatedAt     time.Time      `json:"createdAt"`
}

// NotificationMessage is what every notification-generating call site builds, instead of a bare
// string: Text is the English copy (used for push/email, and as the frontend's fallback if Key
// is ever empty), Key is the i18n key the notification bell actually renders with, and Params
// are that key's interpolation values. Splitting these out (rather than just storing Text) is
// what lets a Spanish-language viewer see cup/transfer/match notifications in Spanish instead of
// always English.
type NotificationMessage struct {
	Text   string
	Key    string
	Params map[string]any
}

const InsertNotificationStmtSql = "INSERT INTO notifications (user_id, notification_type, message, message_key, message_params) " +
	"VALUES ($1, $2, $3, $4, $5)"
const InsertNotificationWithLinkStmtSql = "INSERT INTO notifications (user_id, notification_type, message, message_key, message_params, link) " +
	"VALUES ($1, $2, $3, $4, $5, $6)"

const GetNotificationsStmtSql = "SELECT notification_id, notification_type, message, COALESCE(message_key, ''), message_params, COALESCE(link, ''), read, created_at " +
	"FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2"

const GetUnreadNotificationCountStmtSql = "SELECT count(*) FROM notifications WHERE user_id = $1 AND read = false"

const MarkNotificationReadStmtSql = "UPDATE notifications SET read = true WHERE notification_id = $1 AND user_id = $2"

const MarkAllNotificationsReadStmtSql = "UPDATE notifications SET read = true WHERE user_id = $1 AND read = false"

const DeleteNotificationStmtSql = "DELETE FROM notifications WHERE notification_id = $1 AND user_id = $2"

const DeleteAllNotificationsStmtSql = "DELETE FROM notifications WHERE user_id = $1"

// CreateNotification adds one entry to a user's feed.
func CreateNotification(dbCtx *DatabaseContext, userId int64, notificationType int, msg NotificationMessage) error {
	paramsJson, err := marshalNotificationParams(msg.Params)
	if err != nil {
		return err
	}
	_, err = dbCtx.Db.Exec(InsertNotificationStmtSql, userId, notificationType, msg.Text, nullIfEmpty(msg.Key), paramsJson)
	if err == nil {
		NotificationsSentCounter.Add(context.Background(), 1, metric.WithAttributes(NotificationTypeAttribute(notificationType)))
	}
	return err
}

// CreateNotificationWithLink is CreateNotification plus an in-app route the frontend navigates to
// on click (see Notification.Link) - for a notification type whose message alone can't point a
// viewer at the right page (e.g. NOTIFICATION_PROGRESSION's squad-wide evolutions view).
func CreateNotificationWithLink(dbCtx *DatabaseContext, userId int64, notificationType int, msg NotificationMessage, link string) error {
	paramsJson, err := marshalNotificationParams(msg.Params)
	if err != nil {
		return err
	}
	_, err = dbCtx.Db.Exec(InsertNotificationWithLinkStmtSql, userId, notificationType, msg.Text, nullIfEmpty(msg.Key), paramsJson, link)
	if err == nil {
		NotificationsSentCounter.Add(context.Background(), 1, metric.WithAttributes(NotificationTypeAttribute(notificationType)))
	}
	return err
}

// marshalNotificationParams JSON-encodes a message's interpolation params for the
// message_params jsonb column - nil (not an empty-object literal) for a message with none, so
// the column stays genuinely NULL rather than "{}".
func marshalNotificationParams(params map[string]any) ([]byte, error) {
	if len(params) == 0 {
		return nil, nil
	}
	return json.Marshal(params)
}

func nullIfEmpty(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// NotifyTeamOwner is CreateNotification for a team's current owner, best-effort: an unowned
// team (nobody has claimed it yet) or a lookup error simply results in no notification, since
// notifications are a side effect that must never block or fail the gameplay event that
// triggered them (a match result, a transfer, ...). Also delivers through email/push (see
// Notifier) if the owner has enabled either - itself best-effort, on top of the in-app entry
// which always gets written regardless of these preferences.
func NotifyTeamOwner(dbCtx *DatabaseContext, teamId int64, notificationType int, msg NotificationMessage) error {
	userId, err := GetTeamOwner(dbCtx, teamId)
	if err != nil || userId <= 0 {
		return nil
	}
	if err := CreateNotification(dbCtx, userId, notificationType, msg); err != nil {
		return err
	}
	deliverExternalNotifications(dbCtx, userId, msg)
	return nil
}

// NotifyTeamOwnerWithLink is NotifyTeamOwner plus an in-app route (see CreateNotificationWithLink).
func NotifyTeamOwnerWithLink(dbCtx *DatabaseContext, teamId int64, notificationType int, msg NotificationMessage, link string) error {
	userId, err := GetTeamOwner(dbCtx, teamId)
	if err != nil || userId <= 0 {
		return nil
	}
	if err := CreateNotificationWithLink(dbCtx, userId, notificationType, msg, link); err != nil {
		return err
	}
	deliverExternalNotifications(dbCtx, userId, msg)
	return nil
}

// deliverExternalNotifications is best-effort by design: a lookup failure or a Notifier error
// here must never fail NotifyTeamOwner - the in-app notification, the part that actually works,
// has already been written by the time this runs. Renders into the recipient's stored language
// (see GetUserLanguage/renderNotificationMessage) when one is set and recognized, falling back to
// msg.Text (English) otherwise.
func deliverExternalNotifications(dbCtx *DatabaseContext, userId int64, msg NotificationMessage) {
	prefs, err := GetNotificationPreferences(dbCtx, userId)
	if err != nil {
		return
	}
	if !prefs.EmailEnabled && !prefs.PushEnabled {
		return
	}
	language, err := GetUserLanguage(dbCtx, userId)
	if err != nil {
		language = ""
	}
	message := renderNotificationMessage(msg, language)
	if prefs.EmailEnabled {
		_ = DefaultNotifier.Send(userId, "email", message)
	}
	if prefs.PushEnabled {
		_ = DefaultNotifier.Send(userId, "push", message)
	}
}

func GetNotifications(dbCtx *DatabaseContext, userId int64, limit int) ([]Notification, error) {
	rows, err := dbCtx.Db.Query(GetNotificationsStmtSql, userId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := make([]Notification, 0)
	for rows.Next() {
		var n Notification
		var paramsJson []byte
		if err := rows.Scan(&n.Id, &n.Type, &n.Message, &n.MessageKey, &paramsJson, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		if len(paramsJson) > 0 {
			if err := json.Unmarshal(paramsJson, &n.MessageParams); err != nil {
				log.Errorf("error unmarshalling params for notification %d: %s", n.Id, err)
			}
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func GetUnreadNotificationCount(dbCtx *DatabaseContext, userId int64) (int, error) {
	var count int
	err := dbCtx.Db.QueryRow(GetUnreadNotificationCountStmtSql, userId).Scan(&count)
	return count, err
}

func MarkNotificationRead(dbCtx *DatabaseContext, notificationId int64, userId int64) error {
	_, err := dbCtx.Db.Exec(MarkNotificationReadStmtSql, notificationId, userId)
	return err
}

func MarkAllNotificationsRead(dbCtx *DatabaseContext, userId int64) error {
	_, err := dbCtx.Db.Exec(MarkAllNotificationsReadStmtSql, userId)
	return err
}

// DeleteNotification removes one of userId's own notifications - scoped by user_id the same way
// MarkNotificationRead is, so one user can never delete another's row just by guessing an id.
func DeleteNotification(dbCtx *DatabaseContext, notificationId int64, userId int64) error {
	_, err := dbCtx.Db.Exec(DeleteNotificationStmtSql, notificationId, userId)
	return err
}

// DeleteAllNotifications clears userId's entire notification feed, read and unread alike.
func DeleteAllNotifications(dbCtx *DatabaseContext, userId int64) error {
	_, err := dbCtx.Db.Exec(DeleteAllNotificationsStmtSql, userId)
	return err
}
