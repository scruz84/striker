# Design

## Context

- Every instrument lives in `common/telemetry.go` as a package var created from the global
  `Meter`. They are no-ops until `InitMetrics` installs a provider, so call sites never check
  whether export is enabled.
- The DB-wide gauges are registered only by `engine.StartEngine` (`RegisterMarketGauges`, guarded
  by `sync.Once` for `--svc --engine`) and read with `max()`, because every engine replica exports
  the same value.
- Per-process gauges are registered in `InitMetrics` → `registerProcessHealthGauges`, which has no
  `*sql.DB`.
- The OTLP periodic reader exports every 60 s by default, so every gauge callback runs once per
  minute per process.
- `service.name` is `striker-svc`, `striker-engine` or `striker-combined`, and maps to the
  Prometheus label `job`. `service.instance.id` (the hostname) maps to `instance`.
- Ledger rows are written by `AdjustTeamBudget` / `AdjustAllTeamsBudget` and, inside their own
  transactions, by `transfers.go`, `offers.go`, `loans.go`, `contracts.go` and `stadium.go`
  through `InsertTeamTransactionStmtSql` (32 call sites in total).
- The engine jobs are quartz jobs (`engine/tasks.go`), each with its own near-identical
  `Start*Scheduler` in `engine/scheduler.go`. Once-per-period jobs go through
  `claimRun` → `common.ClaimEngineJobSlot`.
- Prometheus naming after OTLP ingestion: dots become underscores, the unit is appended (`s` →
  `_seconds`, `ms` → `_milliseconds`, `By` → `_bytes`), monotonic counters get `_total`, and an
  up/down counter becomes a plain gauge.

## Goals / Non-Goals

**Goals:**
- Each new metric costs at most one indexed query per process per minute, or an in-memory
  increment.
- A test fails when a metric is added without a panel, or a `TRANSACTION_*` reason without a
  label.

**Non-Goals:**
- Changing how money moves, or the engine's locking or claims. The metrics only observe them.
- A dashboard generator. The JSON stays hand-edited and is guarded by a test.

## Decisions

1. **Ledger flow through one insert helper.** Add `insertLedgerRow(tx, flows *LedgerFlows, teamId,
   amount, reason, description, playerId)` and `(*LedgerFlows).Commit(ctx)` in `common`.
   - Every site that today executes `InsertTeamTransactionStmtSql` (and the all-teams variant,
     which records `amount × affected`) calls the helper inside its transaction, then calls
     `flows.Commit` only after `tx.Commit()` succeeds. A rolled-back write is therefore never
     counted.
   - The all-teams statement goes through a sibling, `insertLedgerRowForAllTeams(tx, flows, teams,
     ...)`, which buffers `amount × teams`.
   - A source test (`ledgerflow_test.go`) asserts that `InsertTeamTransactionStmtSql` and
     `InsertTeamTransactionForAllTeamsStmtSql` appear only inside those two helpers, and that no
     other string in the backend inserts into `team_transactions`.
   - `transactionReasonLabel(int) string` maps every constant to snake_case. A test parses
     `fmconstants.go` with `go/parser`, finds every `TRANSACTION_*` and requires a label other
     than `other`. That catches the reason `amortize-transfer-fees` will add.
   - *Alternative:* an observable counter, `SELECT reason, SUM(amount) FROM team_transactions
     GROUP BY reason`. Rejected: it does a full scan of an ever-growing table every minute on
     every engine replica.
   - *Alternative:* a database trigger plus a summary table. Rejected: it changes the money path
     and needs a migration, for no gain over in-process counting.

2. **Job instrumentation in one wrapper.** Replace the 14 copies of `Start*Scheduler` with
   `scheduleJob(job quartz.Job, cron string, name string)`, which wraps the job in
   `instrumentedJob{inner, name}`.
   - `Execute` times `inner.Execute`, adds to `striker.engine.job.runs` and records
     `striker.engine.job.duration` (unit `s`, explicit buckets 0.1, 0.5, 1, 2, 5, 10, 30, 60, 120,
     300, 600).
   - It then stores the finish time in a `sync.Map`, which the observable gauge
     `striker.engine.job.last_run` (unit `s`) reports per job.
   - Job names: `simulate_matches`, `advance_season`, `free_agent_progression`,
     `international_cup`, `cups`, `free_agent_injection`, `friendlies`, `national_team_friendlies`,
     `national_team_call_ups`, `finish_stuck_simulations`, `offer_response`, `birthdays`,
     `ai_activity`, `live_broadcast_recovery` (the one more job `tasks.go` had). The cron
     expressions stay the same.
   - The label is `engine_job`, not `job`. Verified against a Prometheus 3.5 OTLP receiver: the
     resource's `service.name` becomes `job`, and a data-point attribute with that name is
     overwritten, so every job collapsed into one `job="striker-engine"` series. `engine_job` keeps
     them apart.
   - `claimRun` records `striker.engine.job.claims{job, outcome}`. A database error inside
     `ClaimEngineJobSlot` already returns false and logs, so it counts as `lost`.
   - A quartz job that skips because it lost its claim still counts as a run. That is intended:
     runs show the trigger fired on that replica, and claims show who did the work.

3. **Backlog gauges.** Add `CountMatchBacklogStmtSql`, a single statement of three `UNION ALL`
   branches returning `(type, overdue, in_simulation)` for `matches`, `cup_ties` (with
   `home_team_id IS NOT NULL AND away_team_id IS NOT NULL`) and `friendly_matches`, filtered on
   `played = false`. A small wrapper feeds two observable gauges registered in
   `RegisterMarketGauges`, so they are engine-only and read with `max()`.
   - The migration `init85_observability_indexes.sql` adds, with `CREATE INDEX IF NOT EXISTS`,
     `matches_unplayed_date_idx ON matches(match_date) WHERE played = false`, and the same on
     `friendly_matches` and `cup_ties`.
   - Both gauges come from one query callback through `Meter.RegisterCallback`, so it is one
     round trip per minute.

4. **DB pool metrics.** Add `RegisterDBPoolMetrics(db *sql.DB)` (`sync.Once`), called from
   `service.StartSvc` and `engine.StartEngine`. It reads `db.Stats()` in one callback:
   - `in_use` and `idle` for `striker.db.connections`;
   - `WaitCount` for the observable counter `striker.db.connection_waits`;
   - `WaitDuration.Seconds()` for `striker.db.connection_wait_time` (unit `s`).

   These are per process, so the dashboard sums them by instance.

5. **Live connections.** `striker.live.connections` is an `Int64UpDownCounter` in `common`.
   `liveMatchHandlers.go` adds +1 right after `Upgrade` succeeds and `defer`s −1, so every exit
   path decrements it.

6. **HTTP duration.** `metricsMiddleware` checks `websocket.IsWebSocketUpgrade(c.Request)` (gorilla
   is already a dependency) and skips `HTTPRequestDuration.Record` for those requests. The
   dashboard also filters `route!~".*/live"`, for series exported before the change.

7. **Push outcome.** In `WebPushNotifier.sendOne`, a transport error maps to `failed`, a 404/410
   (the existing deletion path) to `expired`, any other 2xx to `sent`, and any other status
   (429, 5xx, ...) to `failed`. Today such statuses are silently treated as success; the change
   only counts them, without retrying. Each branch adds 1 to `striker.push.sent`.

8. **Dashboard.** Edit the JSON by hand: new rows, panels and the template variables `job` and
   `instance` (`label_values(striker_process_uptime_seconds, job)` / `..., instance)`, with
   `includeAll` and `multi`). Add an annotation query `changes(striker_season_rollovers_total[5m])
   > 0`. Panel ids stay unique, and the rows follow the order in the spec. Example queries:
   - net creation: `sum(rate(striker_economy_flow_total{direction="in",reason!~"transfer_in|loan_fee"}[1h])) - sum(rate(striker_economy_flow_total{direction="out",reason!~"transfer_out|loan_fee"}[1h]))`, times 86400;
   - goals per match: `sum by (type)(rate(striker_goals_scored_total[1h])) / sum by (type)(rate(striker_matches_simulated_total[1h]))`;
   - job freshness: `time() - max by (engine_job)(striker_engine_job_last_run_seconds)`.

   A Go test, `common/telemetry_dashboard_test.go`, checks the JSON so it runs in CI without
   Grafana:
   - it parses the JSON, checks that panel ids are unique and that panels do not overlap within
     a row;
   - it extracts every instrument name from `telemetry.go` with a regex, converts it to its
     Prometheus name (using a table for unit suffixes), and requires it in some `expr`.

Migration: `init85_observability_indexes.sql`. No `TRANSACTION_*` reason and no notification key
are added.

Engine replicas: the change adds no step that generates data. The backlog gauges are read-only,
and running them on every replica is expected (they are read with `max`). The engine-replicas rule
therefore needs no new lock or concurrent test.

## Risks / Trade-offs

- [A future ledger write bypasses the helper and is not counted] → The source test fails if
  either insert statement is used outside `insertLedgerRow`.
- [Counting before commit by mistake] → `LedgerFlows` is a plain in-memory buffer, and
  `LedgerFlows.Commit` is the only code that touches the counter. A unit test with an
  `sdkmetric.ManualReader` checks two things: buffered flows that are never committed export
  nothing, and committed ones export the right `reason`/`direction` sums. The call-site order
  (`tx.Commit()` before `flows.Commit`) is checked in review.
- [Label cardinality] → The labels are bounded: about 17 reasons × 2 directions, about 14 jobs ×
  2 outcomes, 3 types and 3 outcomes. No team, user or match id ever becomes a label.
- [Grafana or ingestion naming differs (unit suffixes)] → The README already warns about it. The
  dashboard test pins the expected names, so a change of convention is a single table edit.
- [The new partial indexes cost writes on insert] → They cover only `played = false` rows, which
  leave the index once played. Each table has a few hundred live rows.

## Migration Plan

Deploy the migration first; it is additive and re-runnable. Then deploy svc and engine, then
re-import the dashboard (same `uid` `striker-economy`, so it overwrites in place). Rollback is to
revert the code. The indexes are harmless and can stay.

## Code map

(To copy into `openspec/specs/observability/spec.md` after archiving: the delta spec cannot carry it, archive drops sections outside the requirements.)

- Instruments and registration: `backend/common/telemetry.go` (every `striker.*` instrument,
  `RegisterMarketGauges`, `registerMatchBacklogGauges`, `RegisterDBPoolMetrics`, `LedgerFlows`,
  `transactionReasonLabel`, `RecordEngineJobRun`, `RecordEngineJobClaim`, `EngineJobAttribute`,
  `RecordPushOutcome`, `CountMatchBacklog` / `CountMatchBacklogStmtSql`).
- Ledger flow: `insertLedgerRow` and `insertLedgerRowForAllTeams` (`common/database.go`), called by
  `AdjustTeamBudget`, `AdjustAllTeamsBudget` and the transactions in `common/transfers.go`,
  `offers.go`, `loans.go`, `contracts.go` and `stadium.go`.
- HTTP: `metricsMiddleware` (`service/fmservice.go`). Live viewers: `serveLiveMatch`
  (`handlers/liveMatchHandlers.go`). Push: `WebPushNotifier.sendOne`
  (`common/webpushnotifier.go`).
- Engine jobs: `scheduleJob` and `instrumentedJob` (`engine/scheduler.go`), `claimRun`
  (`engine/tasks.go`). Process wiring: `service.StartSvc` and `engine.StartEngine` call
  `RegisterDBPoolMetrics`; only `engine.StartEngine` calls `RegisterMarketGauges`.
- Migration: `resources/sql/init85_observability_indexes.sql` (partial indexes on unplayed
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
