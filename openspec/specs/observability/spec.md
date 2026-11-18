# observability Specification

## Purpose

States which OpenTelemetry metrics the svc and engine processes export, under which names and
labels and from which process, and what the Grafana dashboard in the repo shows. An operator
should be able to tell from it whether the API, the engine replicas and the in-game economy are
healthy.

## Requirements

### Requirement: Metrics export is opt-in and free when off
Metrics SHALL be exported over OTLP only when `OTEL_METRICS_ENABLED=true`. Every series carries the resource attributes `service.name` and `service.instance.id` (the hostname). When export is off, every instrument MUST be a no-op, and no gauge callback may query the database.

#### Scenario: Export disabled
- **WHEN** a replica starts without `OTEL_METRICS_ENABLED`
- **THEN** no metric is exported and no gauge query runs against Postgres

### Requirement: HTTP request metrics
The svc process SHALL count every request in `striker.http.requests` and record its duration in ms in `striker.http.request.duration`. Both carry `method`, `route` (the matched route pattern, or `unmatched`) and `status`. A request with an `Upgrade: websocket` header SHALL be counted but MUST NOT record a duration.

#### Scenario: Live match websocket
- **WHEN** a viewer watches a live match for 95 minutes over `/v1/teams/match/:id/live`
- **THEN** `striker.http.requests` grows by 1 for that route and `striker.http.request.duration` records nothing for it

#### Scenario: Unknown path
- **WHEN** a client requests `/v1/nope`
- **THEN** the request is counted with `route="unmatched"` and `status=404`

### Requirement: Economy flow metric
Every committed ledger row SHALL add |amount| to the counter `striker.economy.flow`, labelled `reason` and `direction`. `reason` is the snake_case name of its `TRANSACTION_*` constant, or `other` if unmapped. `direction` is `in` when amount > 0 and `out` when amount < 0. A rolled-back write MUST NOT be counted, and an amount of 0 is skipped.

#### Scenario: Wages after a match
- **WHEN** a team's ledger records −120 000 with `TRANSACTION_WAGES` and the transaction commits
- **THEN** `striker.economy.flow{reason="wages",direction="out"}` grows by 120 000

#### Scenario: Mass injection
- **WHEN** an admin injects +50 000 into 512 teams with `TRANSACTION_ADMIN_MASS_INJECTION`
- **THEN** `striker.economy.flow{reason="admin_mass_injection",direction="in"}` grows by 25 600 000

#### Scenario: Transfer between teams
- **WHEN** a 2 000 000 transfer commits
- **THEN** `transfer_out`/`out` and `transfer_in`/`in` each grow by 2 000 000, so the net creation over both is 0

#### Scenario: Every reason has a label
- **WHEN** a new `TRANSACTION_*` constant is added without a label
- **THEN** a unit test that iterates every constant fails

### Requirement: Engine job metrics
Every scheduled engine job run SHALL add 1 to `striker.engine.job.runs`, record its duration in s in `striker.engine.job.duration`, and set the gauge `striker.engine.job.last_run` to the unix time it finished. All three carry `engine_job`, a stable snake_case name, never `job`. A once-per-period claim SHALL add 1 to `striker.engine.job.claims` with `engine_job` and `outcome`=`won` or `lost`.

#### Scenario: Two replicas, one daily job
- **WHEN** two replicas fire `birthdays` at 00:00 UTC
- **THEN** `striker.engine.job.claims{engine_job="birthdays"}` grows by 1 with `outcome="won"` on one instance and by 1 with `outcome="lost"` on the other, and both record a run

#### Scenario: Job label does not collide with the resource
- **WHEN** the metrics reach a Prometheus-compatible backend over OTLP
- **THEN** `job` still holds the resource's `service.name` (`striker-engine`), and each engine job is its own series under `engine_job`

#### Scenario: Simulation job stalls
- **WHEN** no replica has finished `simulate_matches` for 5 minutes
- **THEN** the dashboard shows `time() - max(striker_engine_job_last_run_seconds{engine_job="simulate_matches"})` ≥ 300

### Requirement: Simulation backlog gauges
The engine SHALL export `striker.matches.overdue` and `striker.matches.in_simulation`, labelled `type` = `league`, `cup` or `friendly`. Overdue counts unplayed matches with both teams set and kickoff more than 5 minutes ago. In simulation counts unplayed rows with `in_simulation = true`. Only the engine registers them; the dashboard reads them with `max`, never `sum`.

#### Scenario: Engine down for 20 minutes
- **WHEN** all engine replicas stop and 8 league matches pass their kickoff by more than 5 minutes
- **THEN** after the engine restarts, its first export reports `striker.matches.overdue{type="league"}` = 8 until the matches are simulated

#### Scenario: Cup bye
- **WHEN** a cup tie has no away team
- **THEN** it is never counted as overdue

### Requirement: Backlog gauge cost
Each backlog gauge query MUST use an index limited to unplayed rows, and MUST NOT scan played match history. With 1 000 000 played matches, a query SHALL read only unplayed rows.

#### Scenario: Query plan
- **WHEN** `EXPLAIN` runs on the overdue query for `matches` after the migration
- **THEN** the plan uses the partial unplayed index and no sequential scan of `matches`

### Requirement: Live connection gauge
The svc process SHALL keep `striker.live.connections` equal to the open live-match websockets on that instance: +1 after a successful upgrade, −1 when the connection closes for any reason.

#### Scenario: Viewer closes the tab
- **WHEN** 3 viewers are connected to one instance and one closes the tab
- **THEN** that instance reports 2

### Requirement: Database pool metrics
Every process SHALL export its own `database/sql` pool state: the gauge `striker.db.connections` with `state` = `in_use` or `idle`, the counter `striker.db.connection_waits`, and the counter `striker.db.connection_wait_time` (s).

#### Scenario: Pool saturated
- **WHEN** 50 connections are in use, the cap is 50, and 10 more queries wait
- **THEN** that instance reports `in_use`=50 and `striker.db.connection_waits` grows by 10

### Requirement: Web push outcome metric
Each web push delivery attempt to one subscription SHALL add 1 to `striker.push.sent` with `outcome`: `sent` on success, `expired` when the subscription is gone (404/410) and gets deleted, and `failed` on any other error. When VAPID keys are not configured, nothing is counted.

#### Scenario: Stale subscription
- **WHEN** the push service answers 410 for a subscription
- **THEN** `striker.push.sent{outcome="expired"}` grows by 1 and the subscription is deleted

### Requirement: Dashboard rows
`backend/resources/grafana/striker-dashboard.json` SHALL be importable with the single datasource input `DS_PROMETHEUS`. It SHALL have, in this order, the rows: Service health, Engine health, Simulated world, Economy, Economy flows, Transfer & loan market, Community, Live & delivery and Per node. Every metric this spec names MUST appear in at least one panel.

#### Scenario: Metric coverage check
- **WHEN** a test lists every instrument name in `common/telemetry.go`
- **THEN** each name, converted to its Prometheus form, appears in some panel expression of the dashboard JSON

### Requirement: Service health panels
The Service health row SHALL show:
- the request rate by route;
- the 5xx ratio;
- the 4xx ratio;
- a table of the 10 routes with the most 5xx responses in the selected range;
- p50 and p95 latency overall and p95 by route, all excluding routes ending in `/live`.

#### Scenario: Bad deploy returns 500 on one route
- **WHEN** `/v1/teams/division/:id/` starts failing
- **THEN** it tops the 5xx table and the 5xx ratio rises, while the p95 is not hidden behind websocket durations

### Requirement: Engine health panels
The Engine health row SHALL show:
- overdue and in-simulation matches by type;
- the seconds since each job's last run, with a threshold at 300 s for jobs that run every minute;
- the p95 duration by job;
- the claims won by instance for each claimed job.

#### Scenario: Replica never wins
- **WHEN** one of two replicas has won no claim in 24 h
- **THEN** the claims panel shows that instance at 0, while the other shows every claim

### Requirement: Economy flow panels
The Economy flows row SHALL show:
- money credited per day by reason, stacked;
- money debited per day by reason, stacked;
- net money created per day, credited minus debited, excluding `transfer_in`, `transfer_out` and `loan_fee`;
- `striker_economy_total_budget` as a time series.

#### Scenario: Sponsors outpace wages
- **WHEN** sponsorship credits 9 000 000 per day and wages and upkeep debit 7 000 000 per day
- **THEN** the net creation panel shows +2 000 000 per day, and the total-budget trend rises

### Requirement: Game activity panels
The dashboard SHALL also show:
- goals per match by competition type, as goals divided by matches over 1 h windows;
- CPU market actions per day by `action`;
- each season rollover as an annotation on every time series.

#### Scenario: Rollover marker
- **WHEN** a country's season rolls over at 14:05
- **THEN** every time-series panel shows an annotation at 14:05

### Requirement: Per-node filters
The Per node row SHALL be filterable by the template variables `job` and `instance`, multi-select with "All" as default, read from `striker_process_uptime_seconds`. Both share panels SHALL use a 1 h rate window.

#### Scenario: Compare two engine replicas
- **WHEN** an operator selects `job=striker-engine` and two instances
- **THEN** the goroutine, heap, CPU and uptime panels show only those two series

## Code map

- Instruments and registration: `backend/common/telemetry.go` (every `striker.*` instrument,
  `RegisterMarketGauges`, `registerMatchBacklogGauges`, `RegisterDBPoolMetrics`, `LedgerFlows`,
  `transactionReasonLabel`, `RecordEngineJobRun`, `RecordEngineJobClaim`, `EngineJobAttribute`,
  `RecordPushOutcome`, `CountMatchBacklog` / `CountMatchBacklogStmtSql`).
- CPU level gauges: `registerAiLevelGauges` and `AiLevelAttribute` (`common/telemetry.go`), fed by
  `GetAiLevelStats` / `GetAiLevelStatsStmtSql` / `foldAiLevelStats` (`common/ailevelstats.go`),
  which also fill `AdminOverview.AiLevels` (`handlers/adminHandlers.go`, frontend
  `features/admin/OverviewTab.tsx` `AiLevelsTable`). Tests: `common/ailevelstats_test.go`,
  `common/ailevelstats_integration_test.go`.
- Ledger flow: `insertLedgerRow` and `insertLedgerRowForAllTeams` (`common/database.go`), called by
  `AdjustTeamBudget`, `AdjustAllTeamsBudget` and the transactions in `common/transfers.go`,
  `offers.go`, `loans.go`, `contracts.go` and `stadium.go`.
- HTTP: `metricsMiddleware` (`service/fmservice.go`). Live viewers: `serveLiveMatch`
  (`handlers/liveMatchHandlers.go`). Push: `WebPushNotifier.sendOne`
  (`common/webpushnotifier.go`).
- Engine jobs: `scheduleJob` and `instrumentedJob` (`engine/scheduler.go`), `claimRun`
  (`engine/tasks.go`). Process wiring: `service.StartSvc` and `engine.StartEngine` call
  `RegisterDBPoolMetrics`; only `engine.StartEngine` calls `RegisterMarketGauges`.
- Migration: `resources/sql/history/init85_observability_indexes.sql` (partial indexes on unplayed
  matches, cup ties and friendlies).
- Dashboard: `backend/resources/grafana/striker-dashboard.json`, guarded by
  `common/telemetry_dashboard_test.go`.
- Tests: `common/ledgerflow_test.go`, `common/dbpoolmetrics_test.go`,
  `common/webpushnotifier_test.go`, `common/match_backlog_integration_test.go`,
  `engine/scheduler_test.go`, `engine/engine_replicas_integration_test.go`
  (`TestClaimRun_RecordsWonAndLostPerReplica`), `service/metrics_middleware_test.go`,
  `handlers/liveMatchHandlers_test.go`. They read the instruments through
  `common/metrictest` (one `ManualReader` per test binary).
- Docs: the "OpenTelemetry metrics" section of `backend/README.md`.
