package common

import (
	"context"
	"os"
	"strconv"
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
)

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

// RecordMatchSimulated updates MatchesSimulatedCounter and GoalsScoredCounter for one
// just-finished match - matchType is "league", "cup", or "friendly" (see each call site).
func RecordMatchSimulated(ctx context.Context, matchType string, homeGoals int, awayGoals int) {
	attrs := metric.WithAttributes(MatchTypeAttribute(matchType))
	MatchesSimulatedCounter.Add(ctx, 1, attrs)
	GoalsScoredCounter.Add(ctx, int64(homeGoals+awayGoals), attrs)
}

// otelMetricsEnabledEnvVar and otelServiceNameEnvVar are read directly (rather than through
// this file's own env-parsing helpers) since InitMetrics runs once at process startup, before
// anything else needs these values.
const otelMetricsEnabledEnvVar = "OTEL_METRICS_ENABLED"
const otelServiceNameEnvVar = "OTEL_SERVICE_NAME"

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
