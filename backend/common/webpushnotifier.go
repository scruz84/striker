package common

import (
	"encoding/json"
	"net/http"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
	log "github.com/sirupsen/logrus"
)

// pushNotificationTTLSeconds bounds how long a push service should hold onto an undelivered
// notification (the recipient's device offline, say) before giving up on it - a day is generous
// for a match result or transfer update, which is only ever interesting for a little while.
const pushNotificationTTLSeconds = 86400

// PushPayload is the JSON body public/sw.js's push event handler decodes to build the on-screen
// notification.
type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// WebPushNotifier delivers "push" notifications for real via the Web Push protocol - VAPID
// keys identify this server to each browser's push service (see GenerateVAPIDKeys, called once
// to produce the pair this is configured with), Subscriber is the contact address VAPID requires
// in case a push service ever needs to reach the sender (a "mailto:" URL). "email" is
// deliberately untouched here: it falls back to the same log-only behavior LogNotifier already
// had, since only push has a real implementation to swap in for now.
type WebPushNotifier struct {
	DbCtx           *DatabaseContext
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	Subscriber      string
}

func (n WebPushNotifier) Send(userId int64, channel string, message string) error {
	if channel != "push" {
		log.Infof("[WebPushNotifier] would send %s notification to user %d: %s", channel, userId, message)
		return nil
	}

	subscriptions, err := GetPushSubscriptionsForUser(n.DbCtx, userId)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(PushPayload{Title: "Striker", Body: message})
	if err != nil {
		return err
	}

	for _, sub := range subscriptions {
		n.sendOne(userId, sub, payload)
	}
	return nil
}

// sendOne pushes to a single subscription, best-effort: a delivery failure for one of a user's
// several devices must never block the rest, so errors are logged here rather than propagated.
func (n WebPushNotifier) sendOne(userId int64, sub PushSubscription, payload []byte) {
	resp, err := webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		Subscriber:      n.Subscriber,
		VAPIDPublicKey:  n.VAPIDPublicKey,
		VAPIDPrivateKey: n.VAPIDPrivateKey,
		TTL:             pushNotificationTTLSeconds,
	})
	if err != nil {
		log.Errorf("error sending push notification to user %d subscription %d: %s", userId, sub.Id, err)
		return
	}
	defer resp.Body.Close()

	// A push service reports 404/410 once a subscription is gone for good (the user uninstalled,
	// cleared site data, or the browser itself expired it) - not a transient failure worth
	// retrying, so the stale row is dropped instead of accumulating forever.
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		if err := DeletePushSubscriptionById(n.DbCtx, sub.Id); err != nil {
			log.Errorf("error deleting stale push subscription %d: %s", sub.Id, err)
		}
	}
}

// ConfigureWebPushNotifier points DefaultNotifier at a real WebPushNotifier when both
// VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY are set in the environment (see GenerateVAPIDKeys for
// producing that pair), leaving DefaultNotifier as the LogNotifier stub otherwise - e.g. a dev
// environment that hasn't generated one yet. Called once at startup by both service.StartSvc and
// engine.StartEngine, since either process can be the one that triggers a notification (a
// human's action through the API, or the simulation engine's own match/season events) - each may
// also run as its own separate container (see docker-compose.yml), so neither can assume the
// other already configured this for the shared DefaultNotifier variable.
func ConfigureWebPushNotifier(dbCtx *DatabaseContext) {
	publicKey := os.Getenv("VAPID_PUBLIC_KEY")
	privateKey := os.Getenv("VAPID_PRIVATE_KEY")
	if publicKey == "" || privateKey == "" {
		log.Warn("VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY not configured - push notifications will only be logged, never actually sent")
		return
	}
	subscriber := os.Getenv("VAPID_SUBJECT")
	if subscriber == "" {
		subscriber = "mailto:admin@example.com"
	}
	DefaultNotifier = WebPushNotifier{DbCtx: dbCtx, VAPIDPublicKey: publicKey, VAPIDPrivateKey: privateKey, Subscriber: subscriber}
	log.Info("push notifications configured via Web Push")
}
