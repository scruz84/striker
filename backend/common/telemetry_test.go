package common

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/metric"
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

// TestRecordTransferCompleted_NeverPanics is the same smoke test as
// TestRecordMatchSimulated_NeverPanics, for the transfer/loan market instruments added
// alongside it - see ExecuteTransfer/AcceptOffer (RecordTransferCompleted), MakeTransferOffer/
// AcceptOffer/RejectOffer/WithdrawOffer (TransferOffersMadeCounter/TransferOffersResolvedCounter),
// and CreateLoanListing/AcceptLoan (LoanListingsCounter/LoansAcceptedCounter).
func TestRecordTransferCompleted_NeverPanics(t *testing.T) {
	RecordTransferCompleted(context.Background(), 1_000_000)
	TransferOffersMadeCounter.Add(context.Background(), 1)
	TransferOffersResolvedCounter.Add(context.Background(), 1, metric.WithAttributes(OfferOutcomeAttribute("accepted")))
	LoanListingsCounter.Add(context.Background(), 1)
	LoansAcceptedCounter.Add(context.Background(), 1)
}

// TestRegisterMarketGauges_RegistersOnceWithoutTouchingDatabase confirms that registering the
// observable gauges (striker.transfer_market.listed_players, striker.loan_market.listed_players,
// striker.loans.active, striker.economy.total_budget) never errors and, thanks to
// marketGaugesOnce, is safe to call more than once - both StartSvc and StartEngine call it, and
// combined mode (--svc --engine) runs both in the same process against the same package-level
// Meter. It's registration-only: under the default no-op MeterProvider (the state this test runs
// under), an observable gauge's callback is never actually invoked, so passing a DatabaseContext
// with no real *sql.DB is safe here even though the callbacks assume a live one.
func TestRegisterMarketGauges_RegistersOnceWithoutTouchingDatabase(t *testing.T) {
	dbCtx := &DatabaseContext{}
	if err := RegisterMarketGauges(dbCtx); err != nil {
		t.Fatalf("expected no error registering market gauges, got %s", err)
	}
	if err := RegisterMarketGauges(dbCtx); err != nil {
		t.Fatalf("expected a second call to be a no-op, got %s", err)
	}
}
