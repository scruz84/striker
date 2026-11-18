package common

// PushSubscription is one browser/device's Web Push subscription - see
// resources/sql/history/init43_push_subscriptions.sql. A user can have several (one per device/browser
// they've enabled push on).
type PushSubscription struct {
	Id       int64  `json:"id"`
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

const UpsertPushSubscriptionStmtSql = "INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES ($1, $2, $3, $4) " +
	"ON CONFLICT (endpoint) DO UPDATE SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth"

// SavePushSubscription records (or, keyed on the endpoint's own uniqueness, replaces) one
// browser/device's push subscription for userId - re-subscribing the same device/browser (e.g.
// its keys rotated) upserts in place rather than accumulating a duplicate row. userId is
// reassigned on conflict too, so a shared/reused browser profile that switches accounts sends
// push to whoever's actually logged in now, not whoever subscribed it first.
func SavePushSubscription(dbCtx *DatabaseContext, userId int64, endpoint string, p256dh string, auth string) error {
	_, err := dbCtx.Db.Exec(UpsertPushSubscriptionStmtSql, userId, endpoint, p256dh, auth)
	return err
}

const DeletePushSubscriptionByEndpointStmtSql = "DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2"

// DeletePushSubscriptionByEndpoint removes one of userId's own subscriptions - called when the
// user disables push, or the browser itself unsubscribes (see PushManager.unsubscribe on the
// frontend). Scoped to userId so one user can never delete another's subscription just by
// guessing/replaying an endpoint URL.
func DeletePushSubscriptionByEndpoint(dbCtx *DatabaseContext, userId int64, endpoint string) error {
	_, err := dbCtx.Db.Exec(DeletePushSubscriptionByEndpointStmtSql, userId, endpoint)
	return err
}

const DeletePushSubscriptionByIdStmtSql = "DELETE FROM push_subscriptions WHERE subscription_id = $1"

// DeletePushSubscriptionById removes a subscription outright, regardless of owner - used by
// WebPushNotifier once a push service reports one as permanently gone (a 404/410 response), which
// carries no user_id of its own to scope a delete by.
func DeletePushSubscriptionById(dbCtx *DatabaseContext, subscriptionId int64) error {
	_, err := dbCtx.Db.Exec(DeletePushSubscriptionByIdStmtSql, subscriptionId)
	return err
}

const GetPushSubscriptionsForUserStmtSql = "SELECT subscription_id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1"

// GetPushSubscriptionsForUser lists every device/browser userId has enabled push on.
func GetPushSubscriptionsForUser(dbCtx *DatabaseContext, userId int64) ([]PushSubscription, error) {
	rows, err := dbCtx.Db.Query(GetPushSubscriptionsForUserStmtSql, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscriptions := make([]PushSubscription, 0)
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.Id, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, s)
	}
	return subscriptions, rows.Err()
}
