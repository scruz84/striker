package common

import (
	"context"
	"testing"
)

func TestNotificationTypeLabel_KnownAndUnknownTypes(t *testing.T) {
	cases := map[int]string{
		NOTIFICATION_MATCH_RESULT: "match_result",
		NOTIFICATION_PROMOTION:    "promotion",
		NOTIFICATION_RELEGATION:   "relegation",
		NOTIFICATION_TRANSFER:     "transfer",
		NOTIFICATION_CUP:          "cup",
		NOTIFICATION_TRAINER:      "trainer",
		NOTIFICATION_FRIENDLY:     "friendly",
		9999:                      "other",
	}
	for notificationType, want := range cases {
		if got := notificationTypeLabel(notificationType); got != want {
			t.Errorf("notificationTypeLabel(%d) = %q, want %q", notificationType, got, want)
		}
	}
}

// TestInitMetrics_DisabledByDefault confirms that with OTEL_METRICS_ENABLED unset (the default
// for every environment that doesn't opt in), InitMetrics is a safe no-op: no error, and the
// returned shutdown function is itself a harmless no-op - callers can always defer it
// unconditionally without checking whether metrics were actually enabled.
func TestInitMetrics_DisabledByDefault(t *testing.T) {
	t.Setenv("OTEL_METRICS_ENABLED", "")

	shutdown, err := InitMetrics("test-service")
	if err != nil {
		t.Fatalf("expected no error when metrics are disabled, got %s", err)
	}
	if shutdown == nil {
		t.Fatal("expected a non-nil shutdown function even when disabled")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("expected the no-op shutdown to succeed, got %s", err)
	}
}

// TestRecordMatchSimulated_NeverPanics is a smoke test that every instrument-recording call
// this file makes stays safe to call with the default no-op MeterProvider - the state every
// test (and every deployment that never sets OTEL_METRICS_ENABLED) runs under.
func TestRecordMatchSimulated_NeverPanics(t *testing.T) {
	RecordMatchSimulated(context.Background(), "league", 2, 1)
	HTTPRequestsCounter.Add(context.Background(), 1)
	NotificationsSentCounter.Add(context.Background(), 1)
}
