package common

import "time"

// Notification is one entry in a user's notification feed (match results, promotion/
// relegation, transfers, cup results - see the NOTIFICATION_* constants).
type Notification struct {
	Id        int64     `json:"id"`
	Type      int       `json:"type"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"createdAt"`
}

const InsertNotificationStmtSql = "INSERT INTO notifications (user_id, notification_type, message) VALUES ($1, $2, $3)"

const GetNotificationsStmtSql = "SELECT notification_id, notification_type, message, read, created_at " +
	"FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2"

const GetUnreadNotificationCountStmtSql = "SELECT count(*) FROM notifications WHERE user_id = $1 AND read = false"

const MarkNotificationReadStmtSql = "UPDATE notifications SET read = true WHERE notification_id = $1 AND user_id = $2"

const MarkAllNotificationsReadStmtSql = "UPDATE notifications SET read = true WHERE user_id = $1 AND read = false"

// CreateNotification adds one entry to a user's feed.
func CreateNotification(dbCtx *DatabaseContext, userId int64, notificationType int, message string) error {
	_, err := dbCtx.Db.Exec(InsertNotificationStmtSql, userId, notificationType, message)
	return err
}

// NotifyTeamOwner is CreateNotification for a team's current owner, best-effort: an unowned
// team (nobody has claimed it yet) or a lookup error simply results in no notification, since
// notifications are a side effect that must never block or fail the gameplay event that
// triggered them (a match result, a transfer, ...).
func NotifyTeamOwner(dbCtx *DatabaseContext, teamId int64, notificationType int, message string) error {
	userId, err := GetTeamOwner(dbCtx, teamId)
	if err != nil || userId <= 0 {
		return nil
	}
	return CreateNotification(dbCtx, userId, notificationType, message)
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
		if err := rows.Scan(&n.Id, &n.Type, &n.Message, &n.Read, &n.CreatedAt); err != nil {
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
