package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
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

// processStartTime captures roughly when this process started (package-init time, a few
// instructions after the real process start - close enough for an uptime gauge) - see
// registerProcessHealthGauges's "striker.process.uptime".
var processStartTime = time.Now()

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

	// TeamsResetCounter counts clubs reset into fresh CPU clubs after losing their owner (see
	// ResetReleasedTeam), labeled by "reason": inactivity, self_deletion, admin_deletion or erasure.
	TeamsResetCounter, _ = Meter.Int64Counter("striker.teams.reset",
		metric.WithDescription("Clubs reset after their owner left, by reason"))

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
	// "outcome" (accepted/rejected/withdrawn, and counter_accepted/counter_declined/counter_expired
	// for a countered one) - see OfferOutcomeAttribute. An "accepted" (or "counter_accepted") offer
	// also drives TransfersCompletedCounter/TransferVolumeCounter via RecordTransferCompleted.
	TransferOffersResolvedCounter, _ = Meter.Int64Counter("striker.offers.resolved",
		metric.WithDescription("Transfer offers resolved, by outcome"))

	// AiMarketActionsCounter counts what CPU teams do on the transfer market, labeled by
	// "action" (signed_free_agent, bought_listed, offer_made, offer_expired, listed_surplus,
	// listed_to_fund) - the per-day volume of CPU transfer activity is read off this series.
	AiMarketActionsCounter, _ = Meter.Int64Counter("striker.ai.market_actions",
		metric.WithDescription("Transfer market actions taken by CPU teams, by action"))

	// LoanListingsCounter counts players newly offered out on loan - see CreateLoanListing.
	LoanListingsCounter, _ = Meter.Int64Counter("striker.loans.listed",
		metric.WithDescription("Players newly listed for loan"))

	// LoansAcceptedCounter counts loan listings accepted by a borrowing team - see AcceptLoan.
	LoansAcceptedCounter, _ = Meter.Int64Counter("striker.loans.accepted",
		metric.WithDescription("Loan listings accepted"))

	// EconomyFlowCounter adds |amount| of every committed ledger row, labeled by "reason" (the
	// snake_case TRANSACTION_* name, see transactionReasonLabel) and "direction" (in/out) - the
	// flows behind striker.economy.total_budget. Only LedgerFlows.Commit touches it, after the
	// ledger transaction committed.
	EconomyFlowCounter, _ = Meter.Int64Counter("striker.economy.flow",
		metric.WithDescription("Money credited (in) and debited (out) through the ledger, by reason"))

	// EngineJobRunsCounter, EngineJobDuration and engineJobLastRun describe each scheduled engine
	// job run on this replica, labeled by "engine_job" (see engine.scheduleJob) - not "job", which an
	// OTLP-to-Prometheus backend reserves for the resource's service.name and would overwrite. A run
	// that skips because
	// another replica already claimed the slot still counts: the trigger did fire here.
	EngineJobRunsCounter, _ = Meter.Int64Counter("striker.engine.job.runs",
		metric.WithDescription("Scheduled engine job runs, by job"))

	EngineJobDuration, _ = Meter.Float64Histogram("striker.engine.job.duration",
		metric.WithDescription("Scheduled engine job duration, by job"), metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600))

	// EngineJobClaimsCounter counts once-per-period claims (ClaimEngineJobSlot), labeled by "engine_job"
	// and "outcome" (won/lost) - how the replicas share the work.
	EngineJobClaimsCounter, _ = Meter.Int64Counter("striker.engine.job.claims",
		metric.WithDescription("Once-per-period engine job claims, by job and outcome (won/lost)"))

	engineJobLastRun, _ = Meter.Float64ObservableGauge("striker.engine.job.last_run",
		metric.WithDescription("Unix time the job last finished on this replica"), metric.WithUnit("s"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			engineJobFinished.Range(func(job, finished any) bool {
				o.Observe(float64(finished.(time.Time).UnixNano())/1e9, metric.WithAttributes(EngineJobAttribute(job.(string))))
				return true
			})
			return nil
		}))

	// LiveConnections is the number of live-match websockets open on this svc instance.
	LiveConnections, _ = Meter.Int64UpDownCounter("striker.live.connections",
		metric.WithDescription("Live match websockets currently open on this instance"))

	// PushSentCounter counts web push deliveries to one subscription, labeled by "outcome"
	// (sent/failed/expired).
	PushSentCounter, _ = Meter.Int64Counter("striker.push.sent",
		metric.WithDescription("Web push deliveries, by outcome"))
)

// EngineJobAttribute labels an engine job metric with the job's stable snake_case name.
func EngineJobAttribute(job string) attribute.KeyValue {
	return attribute.String("engine_job", job)
}

// engineJobFinished holds, per job name, when it last finished on this process - what the
// observable gauge engineJobLastRun reports.
var engineJobFinished sync.Map

// RecordEngineJobRun records one finished scheduled run of job: it adds to the run counter,
// records how long it took and stamps the last-run gauge.
func RecordEngineJobRun(ctx context.Context, job string, started time.Time, finished time.Time) {
	attrs := metric.WithAttributes(EngineJobAttribute(job))
	EngineJobRunsCounter.Add(ctx, 1, attrs)
	EngineJobDuration.Record(ctx, finished.Sub(started).Seconds(), attrs)
	engineJobFinished.Store(job, finished)
}

// RecordEngineJobClaim records the outcome of one ClaimEngineJobSlot attempt for job.
func RecordEngineJobClaim(ctx context.Context, job string, won bool) {
	outcome := "lost"
	if won {
		outcome = "won"
	}
	EngineJobClaimsCounter.Add(ctx, 1, metric.WithAttributes(EngineJobAttribute(job), attribute.String("outcome", outcome)))
}

// RecordPushOutcome adds one web push delivery attempt: outcome is "sent", "expired" or "failed".
func RecordPushOutcome(ctx context.Context, outcome string) {
	PushSentCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
}

// transactionReasonLabel maps a TRANSACTION_* constant to the snake_case "reason" label of
// EconomyFlowCounter. A unit test requires every constant in fmconstants.go to have one, so a new
// reason cannot fall into "other" unnoticed.
func transactionReasonLabel(reason int) string {
	switch reason {
	case TRANSACTION_TRANSFER_IN:
		return "transfer_in"
	case TRANSACTION_TRANSFER_OUT:
		return "transfer_out"
	case TRANSACTION_WAGES:
		return "wages"
	case TRANSACTION_ADMIN_ADJUSTMENT:
		return "admin_adjustment"
	case TRANSACTION_TICKET_REVENUE:
		return "ticket_revenue"
	case TRANSACTION_PRIZE_MONEY:
		return "prize_money"
	case TRANSACTION_SPONSORSHIP:
		return "sponsorship"
	case TRANSACTION_STADIUM_UPGRADE:
		return "stadium_upgrade"
	case TRANSACTION_CONTRACT_RELEASE:
		return "contract_release"
	case TRANSACTION_TRAINER_WAGES:
		return "trainer_wages"
	case TRANSACTION_SEASON_TICKETS:
		return "season_tickets"
	case TRANSACTION_MERCHANDISING:
		return "merchandising"
	case TRANSACTION_ADMIN_MASS_INJECTION:
		return "admin_mass_injection"
	case TRANSACTION_LOAN_FEE:
		return "loan_fee"
	case TRANSACTION_LOAN_WAGE_CONTRIBUTION:
		return "loan_wage_contribution"
	case TRANSACTION_STADIUM_MAINTENANCE:
		return "stadium_maintenance"
	case TRANSACTION_TEAM_RESET:
		return "team_reset"
	default:
		return "other"
	}
}

// LedgerFlows buffers the money a database transaction moves so EconomyFlowCounter only counts it
// once that transaction has committed: declare one next to the transaction, pass it to
// insertLedgerRow for every ledger row, and call Commit right after tx.Commit() succeeds. A
// transaction that rolls back just drops the buffer. The zero value is ready to use.
type LedgerFlows struct {
	sums map[ledgerFlowKey]int64
}

type ledgerFlowKey struct {
	reason    string
	direction string
}

// Add buffers one ledger row of `amount` (negative for an expense). An amount of 0 is skipped.
func (f *LedgerFlows) Add(reason int, amount int64) {
	f.AddTimes(reason, amount, 1)
}

// AddTimes buffers `n` identical rows, as one mass adjustment against every team writes.
func (f *LedgerFlows) AddTimes(reason int, amount int64, n int64) {
	if amount == 0 || n <= 0 {
		return
	}
	direction := "in"
	if amount < 0 {
		direction = "out"
		amount = -amount
	}
	if f.sums == nil {
		f.sums = map[ledgerFlowKey]int64{}
	}
	f.sums[ledgerFlowKey{transactionReasonLabel(reason), direction}] += amount * n
}

// Commit exports everything buffered and empties the buffer.
func (f *LedgerFlows) Commit(ctx context.Context) {
	for key, sum := range f.sums {
		EconomyFlowCounter.Add(ctx, sum, metric.WithAttributes(
			attribute.String("reason", key.reason), attribute.String("direction", key.direction)))
	}
	f.sums = nil
}

// marketGaugesOnce guards RegisterMarketGauges - only engine.StartEngine calls it (see its own
// doc comment below), but --svc --engine combined mode still runs both StartSvc and StartEngine
// in one process against the same package-level Meter, so a second call within that same process
// needs to stay a safe no-op rather than a duplicate-instrument error.
var marketGaugesOnce sync.Once

// errNoDatabase is what a gauge callback reports when it runs against a DatabaseContext with no
// *sql.DB (a test that only checks registration) - otel logs and skips the tick instead of the
// callback panicking.
var errNoDatabase = errors.New("no database connection")

// queryInt64 runs a query that returns a single int64 column (a `SELECT count(*)` or
// `SELECT SUM(...)`) and returns it - exists purely so RegisterMarketGauges's callbacks stay one
// line each.
func queryInt64(dbCtx *DatabaseContext, query string) (int64, error) {
	if dbCtx == nil || dbCtx.Db == nil {
		return 0, errNoDatabase
	}
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

		if err := registerMatchBacklogGauges(dbCtx); err != nil {
			regErr = err
			return
		}
		if err := registerAiLevelGauges(dbCtx); err != nil {
			regErr = err
			return
		}
	})
	return regErr
}

// registerMatchBacklogGauges registers striker.matches.overdue and striker.matches.in_simulation.
// One CountMatchBacklog query feeds both through a single callback, so it is one round trip per
// export tick. Engine-only for the same reason as the gauges in RegisterMarketGauges (DB-wide
// values), read with max().
func registerMatchBacklogGauges(dbCtx *DatabaseContext) error {
	overdue, err := Meter.Int64ObservableGauge("striker.matches.overdue",
		metric.WithDescription("Unplayed matches with both teams set whose kickoff passed more than "+
			strconv.Itoa(int(MatchOverdueAfter/time.Minute))+" minutes ago, by competition type"))
	if err != nil {
		return err
	}
	inSimulation, err := Meter.Int64ObservableGauge("striker.matches.in_simulation",
		metric.WithDescription("Unplayed matches currently claimed by an engine replica for simulation, by competition type"))
	if err != nil {
		return err
	}
	_, err = Meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		backlog, err := CountMatchBacklog(dbCtx)
		if err != nil {
			return err
		}
		for _, b := range backlog {
			attrs := metric.WithAttributes(MatchTypeAttribute(b.Type))
			o.ObserveInt64(overdue, b.Overdue, attrs)
			o.ObserveInt64(inSimulation, b.InSimulation, attrs)
		}
		return nil
	}, overdue, inSimulation)
	return err
}

// MatchOverdueAfter is how long past its kickoff an unplayed match may wait before
// striker.matches.overdue counts it: the simulate job runs every minute and takes any match due
// within the next 15 minutes (see GetSimulableMatchesStmtSql), so a match this late means nothing
// is picking it up.
const MatchOverdueAfter = 5 * time.Minute

// CountMatchBacklogStmtSql counts, per competition type, the unplayed matches that are overdue
// (both teams set - a cup bye has no away team and is never played - and kickoff more than $1
// seconds ago) and the ones currently claimed for simulation. Every branch reads only unplayed
// rows through the partial indexes of init85_observability_indexes.sql.
const CountMatchBacklogStmtSql = "SELECT 'league', " +
	"count(*) FILTER (WHERE match_date < now() - ($1 * interval '1 second')), " +
	"count(*) FILTER (WHERE in_simulation) FROM matches WHERE played = false " +
	"UNION ALL SELECT 'cup', " +
	"count(*) FILTER (WHERE match_date < now() - ($1 * interval '1 second')), " +
	"count(*) FILTER (WHERE in_simulation) FROM cup_ties " +
	"WHERE played = false AND home_team_id IS NOT NULL AND away_team_id IS NOT NULL " +
	"UNION ALL SELECT 'friendly', " +
	"count(*) FILTER (WHERE match_date < now() - ($1 * interval '1 second')), " +
	"count(*) FILTER (WHERE in_simulation) FROM friendly_matches WHERE played = false"

// MatchBacklog is one competition type's row of CountMatchBacklogStmtSql.
type MatchBacklog struct {
	Type         string
	Overdue      int64
	InSimulation int64
}

// CountMatchBacklog returns the backlog for league, cup and friendly matches, in that order.
func CountMatchBacklog(dbCtx *DatabaseContext) ([]MatchBacklog, error) {
	if dbCtx == nil || dbCtx.Db == nil {
		return nil, errNoDatabase
	}
	rows, err := dbCtx.Db.Query(CountMatchBacklogStmtSql, int64(MatchOverdueAfter/time.Second))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	backlog := make([]MatchBacklog, 0, 3)
	for rows.Next() {
		var b MatchBacklog
		if err := rows.Scan(&b.Type, &b.Overdue, &b.InSimulation); err != nil {
			return nil, err
		}
		backlog = append(backlog, b)
	}
	return backlog, rows.Err()
}

// dbPoolMetricsOnce guards RegisterDBPoolMetrics for the same reason as marketGaugesOnce:
// --svc --engine runs StartSvc and StartEngine in one process, and both call it.
var dbPoolMetricsOnce sync.Once

// RegisterDBPoolMetrics exports this process's own database/sql pool state: connections in use
// and idle, and how often (and for how long) a caller had to wait for one. Unlike
// RegisterMarketGauges these are per process, not DB-wide, so every svc and engine replica
// registers them and the dashboard sums them by instance. One db.Stats() call feeds all three
// instruments.
func RegisterDBPoolMetrics(db *sql.DB) error {
	var regErr error
	dbPoolMetricsOnce.Do(func() {
		connections, err := Meter.Int64ObservableGauge("striker.db.connections",
			metric.WithDescription("Database connections in this process's pool, by state (in_use/idle)"))
		if err != nil {
			regErr = err
			return
		}
		waits, err := Meter.Int64ObservableCounter("striker.db.connection_waits",
			metric.WithDescription("Times a caller had to wait for a free database connection"))
		if err != nil {
			regErr = err
			return
		}
		waitTime, err := Meter.Float64ObservableCounter("striker.db.connection_wait_time",
			metric.WithDescription("Total time callers spent waiting for a free database connection"), metric.WithUnit("s"))
		if err != nil {
			regErr = err
			return
		}
		_, regErr = Meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
			stats := db.Stats()
			o.ObserveInt64(connections, int64(stats.InUse), metric.WithAttributes(attribute.String("state", "in_use")))
			o.ObserveInt64(connections, int64(stats.Idle), metric.WithAttributes(attribute.String("state", "idle")))
			o.ObserveInt64(waits, stats.WaitCount)
			o.ObserveFloat64(waitTime, stats.WaitDuration.Seconds())
			return nil
		}, connections, waits, waitTime)
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

	if err := registerProcessHealthGauges(); err != nil {
		return provider.Shutdown, err
	}

	log.Infof("OpenTelemetry metrics export enabled for service %q", serviceName)
	return provider.Shutdown, nil
}

// linuxClockTicksPerSecond is USER_HZ, the unit /proc/[pid]/stat's utime/stime fields are
// counted in - fixed at 100 by long-standing Linux/glibc convention (sysconf(_SC_CLK_TCK)),
// regardless of the kernel's actual timer frequency, on every Linux distribution this app
// actually runs on (see instanceID's own Linux assumption above). Hardcoded rather than read via
// a syscall, since in practice it's never anything else.
const linuxClockTicksPerSecond = 100

// parseProcSelfStatCPUSeconds extracts this process's cumulative user+system CPU time (in
// seconds) from the content of /proc/self/stat - a pure function so the parsing itself is
// testable without a real /proc filesystem (see processCPUSeconds, which reads the real file).
// The second field (comm, the executable name) is parenthesized and can itself contain spaces or
// closing parens, so this finds the LAST ")" rather than naively splitting the whole line on
// spaces - the same technique ps/top and Prometheus's own process collector use. utime and stime
// are fields 14 and 15 overall (see `man proc`), i.e. the 12th and 13th fields after comm.
func parseProcSelfStatCPUSeconds(stat string) (float64, error) {
	afterComm := strings.LastIndex(stat, ")")
	if afterComm == -1 {
		return 0, fmt.Errorf("unexpected /proc/self/stat format: no ')' found")
	}
	fields := strings.Fields(stat[afterComm+1:])
	const utimeIndex, stimeIndex = 11, 12
	if len(fields) <= stimeIndex {
		return 0, fmt.Errorf("unexpected /proc/self/stat format: only %d fields after comm", len(fields))
	}
	utime, err := strconv.ParseFloat(fields[utimeIndex], 64)
	if err != nil {
		return 0, fmt.Errorf("parsing utime: %w", err)
	}
	stime, err := strconv.ParseFloat(fields[stimeIndex], 64)
	if err != nil {
		return 0, fmt.Errorf("parsing stime: %w", err)
	}
	return (utime + stime) / linuxClockTicksPerSecond, nil
}

// processCPUSeconds reads this process's own cumulative CPU time from /proc/self/stat (Linux
// only, which every deployed svc/engine container is - see instanceID). ok is false on any other
// platform, or if /proc/self/stat couldn't be read or parsed, so the gauge callback below can
// just skip observing a value that tick rather than failing the whole metrics export over
// something that was never expected to work everywhere (a developer's own machine, say).
func processCPUSeconds() (seconds float64, ok bool) {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0, false
	}
	seconds, err = parseProcSelfStatCPUSeconds(string(data))
	if err != nil {
		log.Warnf("failed to parse /proc/self/stat for the process CPU time metric: %s", err)
		return 0, false
	}
	return seconds, true
}

// registerProcessHealthGauges wires up four observable instruments describing this process's
// own runtime health - goroutines, heap memory in use, uptime, and cumulative CPU time - each
// labeled by `instance` the same way every other metric here is (see InitMetrics's doc comment
// on service.instance.id), so the "Per node" dashboard row can compare them side by side across
// every svc/engine replica. Unlike RegisterMarketGauges, this runs unconditionally for every
// process (called directly from InitMetrics, not gated behind a --engine-only guard): there's no
// DB-wide value here to accidentally double-count, each replica's own process stats are
// genuinely its own.
func registerProcessHealthGauges() error {
	if _, err := Meter.Int64ObservableGauge("striker.process.goroutines",
		metric.WithDescription("Live goroutines in this process right now"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(runtime.NumGoroutine()))
			return nil
		})); err != nil {
		return err
	}

	if _, err := Meter.Int64ObservableGauge("striker.process.memory.heap_alloc",
		metric.WithDescription("Bytes of heap memory currently allocated and in use by this process"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			o.Observe(int64(stats.HeapAlloc))
			return nil
		})); err != nil {
		return err
	}

	if _, err := Meter.Float64ObservableGauge("striker.process.uptime",
		metric.WithDescription("Seconds since this process started"),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(time.Since(processStartTime).Seconds())
			return nil
		})); err != nil {
		return err
	}

	if _, err := Meter.Float64ObservableCounter("striker.process.cpu_time",
		metric.WithDescription("Cumulative CPU time (user+system) this process has consumed - "+
			"rate() this over a window to get cores used (1.0 = fully saturating one core). "+
			"Only observed on Linux (see processCPUSeconds)."),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			if seconds, ok := processCPUSeconds(); ok {
				o.Observe(seconds)
			}
			return nil
		})); err != nil {
		return err
	}

	return nil
}

// AiLevelAttribute labels a CPU level gauge with its bucket.
func AiLevelAttribute(bucket string) attribute.KeyValue {
	return attribute.String("ai_level", bucket)
}

// registerAiLevelGauges registers striker.ai.teams, striker.ai.average_budget,
// striker.ai.insolvent_teams and striker.ai.points_per_match, all fed by one GetAiLevelStats query
// per export tick through a single callback. Engine-only and DB-wide like the other gauges of
// RegisterMarketGauges, so dashboards read them with max().
func registerAiLevelGauges(dbCtx *DatabaseContext) error {
	teams, err := Meter.Int64ObservableGauge("striker.ai.teams",
		metric.WithDescription("Clubs per AI level (human = owned)"))
	if err != nil {
		return err
	}
	budget, err := Meter.Int64ObservableGauge("striker.ai.average_budget",
		metric.WithDescription("Average club budget per AI level"))
	if err != nil {
		return err
	}
	insolvent, err := Meter.Int64ObservableGauge("striker.ai.insolvent_teams",
		metric.WithDescription("Clubs below the insolvency line per AI level"))
	if err != nil {
		return err
	}
	ppm, err := Meter.Float64ObservableGauge("striker.ai.points_per_match",
		metric.WithDescription("Current-season league points per match per AI level"))
	if err != nil {
		return err
	}
	_, err = Meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		stats, err := GetAiLevelStats(dbCtx)
		if err != nil {
			return err
		}
		for _, s := range stats {
			attrs := metric.WithAttributes(AiLevelAttribute(s.Level))
			o.ObserveInt64(teams, s.Teams, attrs)
			o.ObserveInt64(budget, s.AverageBudget, attrs)
			o.ObserveInt64(insolvent, s.Insolvent, attrs)
			o.ObserveFloat64(ppm, s.PointsPerMatch, attrs)
		}
		return nil
	}, teams, budget, insolvent, ppm)
	return err
}
