package common

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Meter is the app's single OpenTelemetry meter, used to create every instrument below. It's
// always safe to use, whether or not metrics export is actually enabled: before InitMetrics
// installs a real MeterProvider (or when it's never called, or OTEL_METRICS_ENABLED is unset),
// otel's default global MeterProvider is a documented no-op - every instrument created from it,
// and every Add/Record call on those instruments, is a harmless no-op. That's why the
// instrumentation calls scattered through common/engine/handlers never need an "is this even
// enabled" check of their own.
var Meter = otel.Meter("github.com/scruz84/striker")

// Instruments for the handful of events judged most worth watching in production - see
// InitMetrics's doc comment for how to actually export them. Errors from instrument creation are
// ignored (never non-nil for these fixed, hardcoded names/options - the API can only fail on
// malformed instrument configuration, not at runtime) so these can be declared as simple package
// vars rather than requiring init-time error handling everywhere they're used.
var (
	// HTTPRequestsCounter counts every HTTP request the API service handles, labeled by method,
	// route (the matched Gin route pattern, not the raw path - so /v1/teams/player/:id/ stays
	// one series regardless of which player id was requested), and response status code.
	HTTPRequestsCounter, _ = Meter.Int64Counter("striker.http.requests",
		metric.WithDescription("HTTP requests handled by the API service"))

	// HTTPRequestDuration is how long each request took to handle, in milliseconds - same
	// labels as HTTPRequestsCounter, so the two can be correlated (request count and p95 latency
	// per route).
	HTTPRequestDuration, _ = Meter.Float64Histogram("striker.http.request.duration",
		metric.WithDescription("HTTP request duration"), metric.WithUnit("ms"))

	// MatchesSimulatedCounter counts every match the engine finishes simulating, labeled by
	// "type" (league/cup/friendly) - the core of what actually makes the game world move.
	MatchesSimulatedCounter, _ = Meter.Int64Counter("striker.matches.simulated",
		metric.WithDescription("Matches simulated, by competition type"))

	// GoalsScoredCounter tallies goals across every simulated match (both teams combined),
	// same "type" label as MatchesSimulatedCounter - lets a dashboard show goals-per-match as a
	// derived rate without the engine computing it itself.
	GoalsScoredCounter, _ = Meter.Int64Counter("striker.goals.scored",
		metric.WithDescription("Goals scored, by competition type"))

	// SeasonRolloversCounter counts each time a season actually completes and rolls over
	// (promotion/relegation, aging, new fixtures) - a genuinely rare, high-significance event
	// worth being able to see happen in a metrics backend, not just in logs.
	SeasonRolloversCounter, _ = Meter.Int64Counter("striker.season.rollovers",
		metric.WithDescription("Season rollovers completed"))

	// TransfersCompletedCounter counts completed transfer-market purchases (listed player
	// bought at asking price) - free agent signings are tracked separately, see
	// FreeAgentSigningsCounter.
	TransfersCompletedCounter, _ = Meter.Int64Counter("striker.transfers.completed",
		metric.WithDescription("Transfer market purchases completed"))

	// TransferVolumeCounter sums the money that has changed hands via completed transfers -
	// the in-game economy's single biggest number, and one no other counter here captures.
	TransferVolumeCounter, _ = Meter.Int64Counter("striker.transfers.volume",
		metric.WithDescription("Total transfer fees paid across completed transfers"))

	// FreeAgentSigningsCounter counts free agent signings (no fee, just a new contract).
	FreeAgentSigningsCounter, _ = Meter.Int64Counter("striker.free_agents.signed",
		metric.WithDescription("Free agent signings"))

	// LoginsCounter counts successful logins, labeled by "provider" (google/dev) - the
	// simplest possible signal of how much the game is actually being played.
	LoginsCounter, _ = Meter.Int64Counter("striker.logins",
		metric.WithDescription("Successful logins, by auth provider"))

	// UserRegistrationsCounter counts brand-new accounts (first-ever sign-in) - distinct from
	// LoginsCounter, which also counts every returning login.
	UserRegistrationsCounter, _ = Meter.Int64Counter("striker.users.registered",
		metric.WithDescription("New user registrations"))

	// NotificationsSentCounter counts every in-app notification written, labeled by
	// "type" (see the NOTIFICATION_* constants) - a proxy for how much is actually happening
	// across every user's game at once (results, transfers, promotions, ...).
	NotificationsSentCounter, _ = Meter.Int64Counter("striker.notifications.sent",
		metric.WithDescription("Notifications written, by notification type"))

	// AcademyPromotionsCounter counts youth academy ("cantera") prospects promoted to the
	// senior squad - see PromoteFromAcademy in academy.go.
	AcademyPromotionsCounter, _ = Meter.Int64Counter("striker.academy.promotions",
		metric.WithDescription("Youth academy prospects promoted to the senior squad"))

	// TransferOffersMadeCounter counts every scouting offer actually created (the player already
	// judged the personal terms acceptable) - see MakeTransferOffer. Distinct from
	// TransfersCompletedCounter, which only fires once an offer or listing is actually accepted.
	TransferOffersMadeCounter, _ = Meter.Int64Counter("striker.offers.made",
		metric.WithDescription("Transfer offers made"))

	// TransferOffersResolvedCounter counts every offer that leaves the pending state, labeled by
	// "outcome" (accepted/rejected/withdrawn) - see OfferOutcomeAttribute. An "accepted" offer
	// also drives TransfersCompletedCounter/TransferVolumeCounter via RecordTransferCompleted.
	TransferOffersResolvedCounter, _ = Meter.Int64Counter("striker.offers.resolved",
		metric.WithDescription("Transfer offers resolved, by outcome"))

	// LoanListingsCounter counts players newly offered out on loan - see CreateLoanListing.
	LoanListingsCounter, _ = Meter.Int64Counter("striker.loans.listed",
		metric.WithDescription("Players newly listed for loan"))

	// LoansAcceptedCounter counts loan listings accepted by a borrowing team - see AcceptLoan.
	LoansAcceptedCounter, _ = Meter.Int64Counter("striker.loans.accepted",
		metric.WithDescription("Loan listings accepted"))
)

// marketGaugesOnce guards RegisterMarketGauges - only engine.StartEngine calls it (see its own
// doc comment below), but --svc --engine combined mode still runs both StartSvc and StartEngine
// in one process against the same package-level Meter, so a second call within that same process
// needs to stay a safe no-op rather than a duplicate-instrument error.
var marketGaugesOnce sync.Once

// queryInt64 runs a query that returns a single int64 column (a `SELECT count(*)` or
// `SELECT SUM(...)`) and returns it - exists purely so RegisterMarketGauges's callbacks stay one
// line each.
func queryInt64(dbCtx *DatabaseContext, query string) (int64, error) {
	var result int64
	err := dbCtx.Db.QueryRow(query).Scan(&result)
	return result, err
}

// RegisterMarketGauges wires up the observable gauges that reflect point-in-time economy state
// (as opposed to the counters above, which only track events as they happen) - "how many players
// are on the transfer market right now" or "how much money is in the economy right now" can't be
// recovered from a counter of listings created or fees paid, since listings disappear (bought,
// cancelled) and transfers just move money between teams rather than create or destroy it. Each
// gauge's callback re-runs a plain aggregate query against dbCtx on every export tick (see
// InitMetrics's periodic reader), which is cheap and always correct, rather than trying to track
// a running total that could drift from the real table contents if an update path were ever
// missed. Deliberately called only from engine.StartEngine, not service.StartSvc - a normal
// deployment always runs both as separate containers (see docker-compose.yml), each with its own
// service.name resource attribute (see InitMetrics), so calling this from both would export every
// one of these gauges twice under two different series for the exact same DB-wide value (they're
// plain aggregate queries, not scoped to whichever process happens to run the callback) - visible
// in a dashboard as the same panel's value appearing duplicated. The counters above don't have
// this problem since each one only fires from the process whose code path actually produced that
// event.
//
// The same duplication shows up again, one level down, once the engine itself is scaled to
// several replicas (see InitMetrics's doc comment on service.instance.id): every replica still
// runs this same DB-wide query and exports the same correct value, now just under N different
// `instance` labels instead of one. Harmless for a "stat" panel reading the latest value (they
// all agree), but summing across instances would multiply the true number by the replica count -
// the dashboard's economy panels use max(...) rather than sum(...) for exactly this reason.
func RegisterMarketGauges(dbCtx *DatabaseContext) error {
	var regErr error
	marketGaugesOnce.Do(func() {
		_, err := Meter.Int64ObservableGauge("striker.transfer_market.listed_players",
			metric.WithDescription("Players currently listed for transfer"),
			metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
				count, err := queryInt64(dbCtx, "SELECT count(*) FROM transfer_listings")
				if err != nil {
					return err
				}
				o.Observe(count)
				return nil
			}))
		if err != nil {
			regErr = err
			return
		}

		_, err = Meter.Int64ObservableGauge("striker.loan_market.listed_players",
			metric.WithDescription("Players currently listed for loan"),
			metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
				count, err := queryInt64(dbCtx, "SELECT count(*) FROM loan_listings")
				if err != nil {
					return err
				}
				o.Observe(count)
				return nil
			}))
		if err != nil {
			regErr = err
			return
		}

		_, err = Meter.Int64ObservableGauge("striker.loans.active",
			metric.WithDescription("Players currently out on an active loan"),
			metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
				count, err := queryInt64(dbCtx, "SELECT count(*) FROM player_loans")
				if err != nil {
					return err
				}
				o.Observe(count)
				return nil
			}))
		if err != nil {
			regErr = err
			return
		}

		// TotalTeamBudget is every team's budget summed together - the total amount of money
		// that exists in the game's economy right now. It moves only when money is actually
		// created or destroyed (season prize payouts, sponsor deals, stadium/wage upkeep), never
		// by a transfer or transfer offer, which just moves an existing amount from one team's
		// budget to another's - so this is a genuinely different signal than
		// striker.transfers.volume, which tracks flow rather than the standing total. Can go
		// negative in aggregate if enough teams are in debt (see ErrTeamInsolvent/MaxDebt), so
		// this is a plain SELECT sum, not a count.
		_, err = Meter.Int64ObservableGauge("striker.economy.total_budget",
			metric.WithDescription("Sum of every team's budget - total money currently in the game economy"),
			metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
				total, err := queryInt64(dbCtx, "SELECT COALESCE(SUM(budget), 0) FROM teams")
				if err != nil {
					return err
				}
				o.Observe(total)
				return nil
			}))
		if err != nil {
			regErr = err
			return
		}
	})
	return regErr
}

// notificationTypeLabel maps a NOTIFICATION_* constant to a short, stable metric label - the
// numeric constant itself would work as an attribute value too, but a name reads better on a
// dashboard and doesn't require cross-referencing the source to decode.
func notificationTypeLabel(notificationType int) string {
	switch notificationType {
	case NOTIFICATION_MATCH_RESULT:
		return "match_result"
	case NOTIFICATION_PROMOTION:
		return "promotion"
	case NOTIFICATION_RELEGATION:
		return "relegation"
	case NOTIFICATION_TRANSFER:
		return "transfer"
	case NOTIFICATION_CUP:
		return "cup"
	case NOTIFICATION_TRAINER:
		return "trainer"
	case NOTIFICATION_FRIENDLY:
		return "friendly"
	default:
		return "other"
	}
}

// MatchTypeAttribute and the other Attribute helpers below exist purely so every call site
// spells its labels the same way (metric.WithAttributes(common.MatchTypeAttribute("cup"))
// rather than each one hand-rolling attribute.String("type", "cup")).
func MatchTypeAttribute(matchType string) attribute.KeyValue {
	return attribute.String("type", matchType)
}

func NotificationTypeAttribute(notificationType int) attribute.KeyValue {
	return attribute.String("type", notificationTypeLabel(notificationType))
}

func LoginProviderAttribute(provider string) attribute.KeyValue {
	return attribute.String("provider", provider)
}

// OfferOutcomeAttribute labels a resolved transfer offer - see TransferOffersResolvedCounter.
func OfferOutcomeAttribute(outcome string) attribute.KeyValue {
	return attribute.String("outcome", outcome)
}

// RecordMatchSimulated updates MatchesSimulatedCounter and GoalsScoredCounter for one
// just-finished match - matchType is "league", "cup", or "friendly" (see each call site).
func RecordMatchSimulated(ctx context.Context, matchType string, homeGoals int, awayGoals int) {
	attrs := metric.WithAttributes(MatchTypeAttribute(matchType))
	MatchesSimulatedCounter.Add(ctx, 1, attrs)
	GoalsScoredCounter.Add(ctx, int64(homeGoals+awayGoals), attrs)
}

// RecordTransferCompleted updates TransfersCompletedCounter and TransferVolumeCounter for one
// just-completed transfer, whether it came from a listing bought outright (ExecuteTransfer) or a
// negotiated offer accepted (AcceptOffer) - both are "a transfer completed" from the economy's
// point of view, so both feed the same two instruments.
func RecordTransferCompleted(ctx context.Context, feeAmount int64) {
	TransfersCompletedCounter.Add(ctx, 1)
	TransferVolumeCounter.Add(ctx, feeAmount)
}

// otelMetricsEnabledEnvVar and otelServiceNameEnvVar are read directly (rather than through
// this file's own env-parsing helpers) since InitMetrics runs once at process startup, before
// anything else needs these values.
const otelMetricsEnabledEnvVar = "OTEL_METRICS_ENABLED"
const otelServiceNameEnvVar = "OTEL_SERVICE_NAME"

// instanceID identifies this specific process for the service.instance.id resource attribute
// (see InitMetrics) - the hostname, which on every deployment shape that actually runs several
// svc/engine replicas (Docker Compose's --scale, a Kubernetes Deployment/StatefulSet) is already
// unique per container/pod with no extra configuration needed. Falls back to the empty string on
// the (practically never happening on Linux/the container images this app ships in) error case,
// which still exports fine - just without a way to tell replicas apart.
func instanceID() string {
	name, err := os.Hostname()
	if err != nil {
		log.Warnf("failed to read hostname for the service.instance.id metric attribute: %s", err)
		return ""
	}
	return name
}

// InitMetrics turns on real OpenTelemetry metrics export for this process, if asked to.
//
// Off by default: unless OTEL_METRICS_ENABLED=true is set, this is a no-op and every
// instrument above keeps using otel's default no-op MeterProvider - Add/Record calls
// throughout the codebase cost a few no-op function calls and nothing is ever exported.
//
// When enabled, it builds an OTLP/HTTP metrics exporter and installs it as the global
// MeterProvider, exporting on a periodic timer (the SDK default is every 60s). The exporter
// itself follows the standard OpenTelemetry environment variables - most importantly
// OTEL_EXPORTER_OTLP_ENDPOINT (or the metrics-specific OTEL_EXPORTER_OTLP_METRICS_ENDPOINT),
// e.g. "http://localhost:4318" for a local otel-collector - see
// https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/ for the full list
// (headers, compression, timeout, TLS). serviceName picks the `service.name` resource
// attribute reported on every exported metric, unless OTEL_SERVICE_NAME overrides it - pass
// something that distinguishes the API service from the simulation engine (see cmd/striker.go)
// since both processes can be exporting to the same collector at once.
//
// Also sets the `service.instance.id` resource attribute to this process's hostname (see
// instanceID) - svc and engine can each be scaled to several replicas (Docker Compose's --scale,
// a Kubernetes Deployment), and every one of them exports under the same service.name. A
// Prometheus-compatible backend that ingests OTLP natively (Prometheus >=2.47's OTLP receiver,
// Mimir, Grafana Cloud) turns service.name/service.instance.id into the usual `job`/`instance`
// labels on every series, so per-node load (is striker.http.requests or
// striker.matches.simulated actually spread evenly across replicas, or is one of them doing all
// the work?) can be read straight off any panel with `by (instance)` - see the "Per node"
// dashboard row in backend/resources/grafana/striker-dashboard.json. The one place this needs
// care is RegisterMarketGauges's observable gauges: they query the same DB-wide aggregate from
// every engine replica, so summing them by instance overcounts - see that function's doc comment.
//
// Returns a shutdown function that flushes any buffered metrics and stops the exporter -
// callers should defer it (or call it from a signal handler) so a normal exit doesn't lose the
// last, not-yet-exported interval of data. The returned function is always safe to call, even
// when metrics were never enabled.
func InitMetrics(serviceName string) (shutdown func(context.Context) error, err error) {
	noop := func(context.Context) error { return nil }

	enabled, _ := strconv.ParseBool(os.Getenv(otelMetricsEnabledEnvVar))
	if !enabled {
		log.Infof("OpenTelemetry metrics export disabled (set %s=true to enable)", otelMetricsEnabledEnvVar)
		return noop, nil
	}

	if name := os.Getenv(otelServiceNameEnvVar); name != "" {
		serviceName = name
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return noop, err
	}

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		attribute.String("service.name", serviceName),
		attribute.String("service.instance.id", instanceID()),
	))
	if err != nil {
		return noop, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)
	// otel's global meter provider is a deferred/delegating implementation: every instrument
	// already created from Meter (all the package vars above, created at package-init time,
	// long before this function ever runs) automatically starts forwarding to this real
	// provider the moment SetMeterProvider is called - nothing needs to be recreated.
	otel.SetMeterProvider(provider)

	log.Infof("OpenTelemetry metrics export enabled for service %q", serviceName)
	return provider.Shutdown, nil
}
