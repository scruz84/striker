package common

import (
	"context"
	"os"
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

// TestInstanceID_MatchesHostname confirms the service.instance.id resource attribute (see
// InitMetrics) tracks the process's actual hostname - the property that makes it unique per
// replica once svc/engine are scaled to several nodes, with no extra configuration needed.
func TestInstanceID_MatchesHostname(t *testing.T) {
	want, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname() failed: %s", err)
	}
	if got := instanceID(); got != want {
		t.Errorf("instanceID() = %q, want %q", got, want)
	}
}

// TestRegisterMarketGauges_RegistersOnceWithoutTouchingDatabase confirms that registering the
// observable gauges (striker.transfer_market.listed_players, striker.loan_market.listed_players,
// striker.loans.active, striker.economy.total_budget) never errors and, thanks to
// marketGaugesOnce, is safe to call more than once - only engine.StartEngine calls it, but
// combined mode (--svc --engine) still runs StartSvc and StartEngine in the same process against
// the same package-level Meter. It's registration-only: under the default no-op MeterProvider
// (the state this test runs under), an observable gauge's callback is never actually invoked, so
// passing a DatabaseContext with no real *sql.DB is safe here even though the callbacks assume a
// live one.
func TestRegisterMarketGauges_RegistersOnceWithoutTouchingDatabase(t *testing.T) {
	dbCtx := &DatabaseContext{}
	if err := RegisterMarketGauges(dbCtx); err != nil {
		t.Fatalf("expected no error registering market gauges, got %s", err)
	}
	if err := RegisterMarketGauges(dbCtx); err != nil {
		t.Fatalf("expected a second call to be a no-op, got %s", err)
	}
}

// TestParseProcSelfStatCPUSeconds_ParsesUtimeAndStime uses a hand-written line in the real
// /proc/[pid]/stat format (see `man proc`) - comm (the second field) deliberately contains a
// space to exercise finding the LAST ")" rather than naively splitting on spaces. utime=1500,
// stime=250 clock ticks -> (1500+250)/100 = 17.5s.
func TestParseProcSelfStatCPUSeconds_ParsesUtimeAndStime(t *testing.T) {
	stat := "1234 (my process name) S 1 1234 1234 0 -1 4194304 100 0 0 0 1500 250 0 0 20 0 4 0 12345 0 0 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"

	got, err := parseProcSelfStatCPUSeconds(stat)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if want := 17.5; got != want {
		t.Errorf("parseProcSelfStatCPUSeconds() = %v, want %v", got, want)
	}
}

func TestParseProcSelfStatCPUSeconds_RejectsMalformedInput(t *testing.T) {
	cases := []string{
		"",
		"no closing paren here",
		"1234 (ok) S 1 1234", // too few fields after comm
	}
	for _, stat := range cases {
		if _, err := parseProcSelfStatCPUSeconds(stat); err == nil {
			t.Errorf("parseProcSelfStatCPUSeconds(%q) expected an error, got none", stat)
		}
	}
}

// TestProcessCPUSeconds_ReadsRealProcSelfStat confirms this test process's own real
// /proc/self/stat parses successfully - a sanity check that the field-counting in
// parseProcSelfStatCPUSeconds matches the real kernel's layout, not just the hand-written sample
// above. Every svc/engine deployment runs on Linux (see processCPUSeconds/instanceID), which is
// also what CI runs this test on.
func TestProcessCPUSeconds_ReadsRealProcSelfStat(t *testing.T) {
	seconds, ok := processCPUSeconds()

	if !ok {
		t.Fatal("expected processCPUSeconds() to succeed")
	}
	if seconds < 0 {
		t.Errorf("processCPUSeconds() = %v, want >= 0", seconds)
	}
}

// TestRegisterProcessHealthGauges_NeverErrors is the same registration-only smoke test as
// TestRegisterMarketGauges_RegistersOnceWithoutTouchingDatabase: under the default no-op
// MeterProvider (the state this test runs under), creating an observable gauge never touches its
// callback, so this is safe without a real OTLP exporter - including the CPU gauge's callback,
// even though it reads the real /proc/self/stat.
func TestRegisterProcessHealthGauges_NeverErrors(t *testing.T) {
	if err := registerProcessHealthGauges(); err != nil {
		t.Fatalf("expected no error registering process health gauges, got %s", err)
	}
}
