package common

import (
	"database/sql"
	"errors"
)

// NotificationPreferences controls whether a user also gets a notification through an external
// channel (email/push - see Notifier) on top of the in-app feed, which always receives every
// notification regardless of these settings. Both default off: since delivery is log-only in
// this environment (see LogNotifier), defaulting to on would suggest the user is getting real
// emails/pushes they never actually receive.
type NotificationPreferences struct {
	EmailEnabled bool `json:"emailEnabled"`
	PushEnabled  bool `json:"pushEnabled"`
}

const GetNotificationPreferencesStmtSql = "SELECT email_enabled, push_enabled FROM notification_preferences WHERE user_id = $1"
const UpsertNotificationPreferencesStmtSql = `
INSERT INTO notification_preferences (user_id, email_enabled, push_enabled) VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE SET email_enabled = EXCLUDED.email_enabled, push_enabled = EXCLUDED.push_enabled, updated_at = now()`

// GetNotificationPreferences returns userId's preferences, or the (false, false) zero value if
// they've never set any - there's no row until a user actually saves a preference.
func GetNotificationPreferences(dbCtx *DatabaseContext, userId int64) (*NotificationPreferences, error) {
	var p NotificationPreferences
	err := dbCtx.Db.QueryRow(GetNotificationPreferencesStmtSql, userId).Scan(&p.EmailEnabled, &p.PushEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return &NotificationPreferences{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func SetNotificationPreferences(dbCtx *DatabaseContext, userId int64, prefs NotificationPreferences) error {
	_, err := dbCtx.Db.Exec(UpsertNotificationPreferencesStmtSql, userId, prefs.EmailEnabled, prefs.PushEnabled)
	return err
}
