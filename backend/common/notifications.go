package common

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Notification is one entry in a user's notification feed (match results, promotion/
// relegation, transfers, cup results - see the NOTIFICATION_* constants). Link is an optional
// in-app route the frontend navigates to on click (see NotifyTeamOwnerWithLink) - empty for
// every notification type that existed before it (their message already says everything
// relevant), so the frontend only needs to treat it as clickable when non-empty.
type Notification struct {
	Id        int64     `json:"notificationId"`
	Type      int       `json:"notificationType"`
	Message   string    `json:"message"`
	Link      string    `json:"link,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"createdAt"`
}

const InsertNotificationStmtSql = "INSERT INTO notifications (user_id, notification_type, message) VALUES ($1, $2, $3)"
const InsertNotificationWithLinkStmtSql = "INSERT INTO notifications (user_id, notification_type, message, link) VALUES ($1, $2, $3, $4)"

const GetNotificationsStmtSql = "SELECT notification_id, notification_type, message, COALESCE(link, ''), read, created_at " +
	"FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2"

const GetUnreadNotificationCountStmtSql = "SELECT count(*) FROM notifications WHERE user_id = $1 AND read = false"

const MarkNotificationReadStmtSql = "UPDATE notifications SET read = true WHERE notification_id = $1 AND user_id = $2"

const MarkAllNotificationsReadStmtSql = "UPDATE notifications SET read = true WHERE user_id = $1 AND read = false"

// CreateNotification adds one entry to a user's feed.
func CreateNotification(dbCtx *DatabaseContext, userId int64, notificationType int, message string) error {
	_, err := dbCtx.Db.Exec(InsertNotificationStmtSql, userId, notificationType, message)
	if err == nil {
		NotificationsSentCounter.Add(context.Background(), 1, metric.WithAttributes(NotificationTypeAttribute(notificationType)))
	}
	return err
}

// CreateNotificationWithLink is CreateNotification plus an in-app route the frontend navigates to
// on click (see Notification.Link) - for a notification type whose message alone can't point a
// viewer at the right page (e.g. NOTIFICATION_PROGRESSION's squad-wide evolutions view).
func CreateNotificationWithLink(dbCtx *DatabaseContext, userId int64, notificationType int, message string, link string) error {
	_, err := dbCtx.Db.Exec(InsertNotificationWithLinkStmtSql, userId, notificationType, message, link)
	if err == nil {
		NotificationsSentCounter.Add(context.Background(), 1, metric.WithAttributes(NotificationTypeAttribute(notificationType)))
	}
	return err
}

// NotifyTeamOwner is CreateNotification for a team's current owner, best-effort: an unowned
// team (nobody has claimed it yet) or a lookup error simply results in no notification, since
// notifications are a side effect that must never block or fail the gameplay event that
// triggered them (a match result, a transfer, ...). Also delivers through email/push (see
// Notifier) if the owner has enabled either - itself best-effort, on top of the in-app entry
// which always gets written regardless of these preferences.
func NotifyTeamOwner(dbCtx *DatabaseContext, teamId int64, notificationType int, message string) error {
	userId, err := GetTeamOwner(dbCtx, teamId)
	if err != nil || userId <= 0 {
		return nil
	}
	if err := CreateNotification(dbCtx, userId, notificationType, message); err != nil {
		return err
	}
	deliverExternalNotifications(dbCtx, userId, message)
	return nil
}

// NotifyTeamOwnerWithLink is NotifyTeamOwner plus an in-app route (see CreateNotificationWithLink).
func NotifyTeamOwnerWithLink(dbCtx *DatabaseContext, teamId int64, notificationType int, message string, link string) error {
	userId, err := GetTeamOwner(dbCtx, teamId)
	if err != nil || userId <= 0 {
		return nil
	}
	if err := CreateNotificationWithLink(dbCtx, userId, notificationType, message, link); err != nil {
		return err
	}
	deliverExternalNotifications(dbCtx, userId, message)
	return nil
}

// deliverExternalNotifications is best-effort by design: a lookup failure or a Notifier error
// here must never fail NotifyTeamOwner - the in-app notification, the part that actually works,
// has already been written by the time this runs.
func deliverExternalNotifications(dbCtx *DatabaseContext, userId int64, message string) {
	prefs, err := GetNotificationPreferences(dbCtx, userId)
	if err != nil {
		return
	}
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
		if err := rows.Scan(&n.Id, &n.Type, &n.Message, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
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
