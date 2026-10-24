package common

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
	"go.opentelemetry.io/otel/attribute"

	"github.com/scruz84/striker/common/metrictest"
)

// pushTestSubscription returns a subscription pointing at endpoint with valid keys (the library
// encrypts the payload for them before posting), and a notifier with a freshly generated VAPID pair.
func pushTestSubscription(t *testing.T, endpoint string) (PushSubscription, WebPushNotifier) {
	t.Helper()
	_, browserPublic, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generating browser key: %s", err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatalf("generating auth secret: %s", err)
	}
	vapidPrivate, vapidPublic, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generating VAPID keys: %s", err)
	}
	return PushSubscription{Endpoint: endpoint, P256dh: browserPublic, Auth: base64.RawURLEncoding.EncodeToString(auth)},
		WebPushNotifier{VAPIDPublicKey: vapidPublic, VAPIDPrivateKey: vapidPrivate, Subscriber: "mailto:test@example.com"}
}

func pushOutcome(t *testing.T, outcome string) float64 {
	t.Helper()
	return metrictest.Sum(t, "striker.push.sent", attribute.String("outcome", outcome))
}

// TestSendOne_CountsSentAndFailed: a 201 from the push service is "sent", a 503 is "failed" and
// so is a transport error (no server listening).
func TestSendOne_CountsSentAndFailed(t *testing.T) {
	metrictest.Install()
	var status atomic.Int32
	status.Store(http.StatusCreated)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(int(status.Load())) }))
	defer server.Close()
	sub, notifier := pushTestSubscription(t, server.URL)

	sent, failed := pushOutcome(t, "sent"), pushOutcome(t, "failed")

	notifier.sendOne(1, sub, []byte(`{}`))
	if got := pushOutcome(t, "sent") - sent; got != 1 {
		t.Errorf("sent grew by %v after a 201, want 1", got)
	}

	status.Store(http.StatusServiceUnavailable)
	notifier.sendOne(1, sub, []byte(`{}`))
	if got := pushOutcome(t, "failed") - failed; got != 1 {
		t.Errorf("failed grew by %v after a 503, want 1", got)
	}

	server.Close()
	notifier.sendOne(1, sub, []byte(`{}`))
	if got := pushOutcome(t, "failed") - failed; got != 2 {
		t.Errorf("failed grew by %v after a 503 and a transport error, want 2", got)
	}
}

// TestSendOne_ExpiredSubscriptionIsCountedAndDeleted: a 410 is "expired" and the stored
// subscription is dropped.
func TestSendOne_ExpiredSubscriptionIsCountedAndDeleted(t *testing.T) {
	if os.Getenv("STRIKER_TEST_DB_DSN") == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, dbCtx := openTransfersTestDB(t)
	metrictest.Install()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusGone) }))
	defer server.Close()
	sub, notifier := pushTestSubscription(t, server.URL)
	notifier.DbCtx = dbCtx

	var userId int64
	if err := db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ('Push Expired', 'test', 'push-expired@test.local') ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email RETURNING user_id`).Scan(&userId); err != nil {
		t.Fatalf("seeding user: %s", err)
	}
	if err := db.QueryRow(`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO UPDATE SET user_id = EXCLUDED.user_id RETURNING subscription_id`, userId, sub.Endpoint, sub.P256dh, sub.Auth).Scan(&sub.Id); err != nil {
		t.Fatalf("seeding subscription: %s", err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM users WHERE user_id = $1`, userId) })

	expired := pushOutcome(t, "expired")
	notifier.sendOne(userId, sub, []byte(`{}`))

	if got := pushOutcome(t, "expired") - expired; got != 1 {
		t.Errorf("expired grew by %v after a 410, want 1", got)
	}
	var remaining int
	if err := db.QueryRow(`SELECT count(*) FROM push_subscriptions WHERE subscription_id = $1`, sub.Id).Scan(&remaining); err != nil {
		t.Fatalf("counting subscription: %s", err)
	}
	if remaining != 0 {
		t.Errorf("the expired subscription was not deleted")
	}
}
