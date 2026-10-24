# Tasks

## 1. HTTP metrics

- [ ] 1.1 In `backend/service/fmservice.go` `metricsMiddleware`, skip `HTTPRequestDuration.Record` when `websocket.IsWebSocketUpgrade(c.Request)`, and still count the request. Verify with a test (gin test router plus `sdkmetric.ManualReader` installed through `otel.SetMeterProvider`): a request carrying `Connection: Upgrade` and `Upgrade: websocket` adds 1 to the counter and no histogram point, while a plain GET adds both.

## 2. Economy flow metric

- [ ] 2.1 In `common/telemetry.go`, add the counter `striker.economy.flow`, `transactionReasonLabel(int) string` (snake_case of every `TRANSACTION_*`, else `other`), and the `LedgerFlows` buffer with `Add(reason, amount)` / `AddTimes(reason, amount, n)` / `Commit(ctx)`. A 0 amount is skipped, `direction` is `in`/`out`, and the value is |amount|. Verify with `ledgerflow_test.go`: an `sdkmetric.ManualReader` shows nothing before `Commit` and the right sums after it, including the spec's mass-injection scenario (50 000 × 512). A `go/parser` test also finds every `TRANSACTION_*` in `fmconstants.go` and requires a label other than `other`.
- [ ] 2.2 Add `insertLedgerRow(tx, flows, ...)`. Route every `InsertTeamTransactionStmtSql` / `InsertTeamTransactionForAllTeamsStmtSql` use in `common/database.go`, `transfers.go`, `offers.go`, `loans.go`, `contracts.go` and `stadium.go` through it, and call `flows.Commit(ctx)` only after `tx.Commit()` succeeds. Verify with a source test that both statements appear only inside `insertLedgerRow`, and with `go test ./common` green.

## 3. Engine job metrics

- [ ] 3.1 Add the instruments `striker.engine.job.runs`, `striker.engine.job.duration` (unit `s`, buckets 0.1…600 as in the design), the observable gauge `striker.engine.job.last_run` (unit `s`, fed from a `sync.Map`) and `striker.engine.job.claims`.
- [ ] 3.2 In `engine/scheduler.go`, replace the `Start*Scheduler` copies with `scheduleJob(job, cron, name)` wrapping `instrumentedJob`. Keep every cron expression unchanged and use the job names from the design. Verify with a unit test: `instrumentedJob` around a fake job adds one run with its `job` label and sets `last_run`. Diff the cron strings before and after to confirm they are identical.
- [ ] 3.3 In `engine/tasks.go` `claimRun`, record `claims{job, outcome=won|lost}`. Verify by extending `engine_replicas_integration_test.go` (or the claim test in `common/enginelocks_integration_test.go`): with N goroutines claiming the same slot, the ManualReader shows `won`=1 and `lost`=N−1.

## 4. Simulation backlog gauges

- [ ] 4.1 Add `backend/resources/sql/init85_observability_indexes.sql`. It runs `CREATE INDEX IF NOT EXISTS` for `matches_unplayed_date_idx`, `friendly_matches_unplayed_date_idx` and `cup_ties_unplayed_date_idx` on `(match_date) WHERE played = false`. Verify by running it twice against the dev database (HOWTO_RUN.md), with no error.
- [ ] 4.2 Add `CountMatchBacklogStmtSql` plus a wrapper, and register `striker.matches.overdue` / `striker.matches.in_simulation` (`type`) in `RegisterMarketGauges` with a single `Meter.RegisterCallback`. Verify with `common/match_backlog_integration_test.go`. Seed one league match 10 minutes past kickoff, one 2 minutes past, one in simulation, and one bye cup tie. Then check overdue{league}=1, in_simulation{league}=1 and overdue{cup}=0. Also check that `EXPLAIN` of each branch shows the partial index and no `Seq Scan on matches`.

## 5. Live, DB pool and push metrics

- [ ] 5.1 Add the `Int64UpDownCounter` `striker.live.connections`. In `handlers/liveMatchHandlers.go`, add +1 after a successful `Upgrade` and `defer` −1. Verify by extending `liveMatchHandlers_test.go` (Redis-gated like its siblings): the counter is 1 while a test client is connected and 0 after it closes.
- [ ] 5.2 Add `RegisterDBPoolMetrics(db *sql.DB)` (`sync.Once`; gauge `striker.db.connections{state}`, observable counters `striker.db.connection_waits` and `striker.db.connection_wait_time` in `s`), and call it from `service.StartSvc` and `engine.StartEngine`. Verify with a unit test on an `sql.DB` opened with a stub driver: the ManualReader reports both `in_use` and `idle` series.
- [ ] 5.3 In `common/webpushnotifier.go` `sendOne`, count `striker.push.sent{outcome}` as the design maps it. Verify with an `httptest.Server` playing the push service, returning 201, 410 and 503 in turn: `sent`=1, `expired`=1 (and the subscription is deleted, behind the existing DB test guard) and `failed`=1.

## 6. Dashboard

- [ ] 6.1 Edit `backend/resources/grafana/striker-dashboard.json`:
  - add the rows in the spec's order: Service health, Engine health, Simulated world, Economy, Economy flows, Transfer & loan market, Community, Live & delivery, Per node;
  - add the new panels (4xx ratio, 5xx-by-route table, p50/p95 without `/live`, backlog, job freshness with a 300 s threshold, job p95, claims won by instance, credited/debited by reason, net creation, total-budget trend, goals per match, CPU market actions, live connections, push outcomes, DB pool);
  - add the `job` / `instance` variables applied to the Per node row, and the season-rollover annotation;
  - change "Matches simulated share per node" to `[1h]`.

  Verify by importing it into a local Grafana 10 pointed at Prometheus (the docker run commands go in the PR) and attaching a screenshot of each row.
- [ ] 6.2 Add `common/telemetry_dashboard_test.go`. It parses the dashboard, requires unique panel ids, no overlapping `gridPos` within a row, the row titles in the spec's order and the `DS_PROMETHEUS` input. It also requires every instrument name found in `telemetry.go` (converted with the unit and suffix table from the design) to appear in some `expr`. Verify that `go test ./common -run Dashboard` passes, and that it fails when one panel is removed.

## 7. Specs and docs

- [ ] 7.1 Update the "OpenTelemetry metrics" section of `backend/README.md`: one bullet per new metric (labels, which process registers it, read with `max` or `sum`), the websocket duration exclusion, and the new rows. Verify that every instrument name in `telemetry.go` is mentioned (`grep`).
- [ ] 7.2 Add a `## Code map` to `openspec/changes/expand-observability-dashboard/specs/observability/spec.md`. It names `common/telemetry.go` (instruments, `RegisterMarketGauges`, `RegisterDBPoolMetrics`, `LedgerFlows`, `transactionReasonLabel`), `insertLedgerRow`, `service/fmservice.go` `metricsMiddleware`, `engine/scheduler.go` `scheduleJob` / `instrumentedJob`, `engine/tasks.go` `claimRun`, `handlers/liveMatchHandlers.go`, `common/webpushnotifier.go`, `init85_observability_indexes.sql` and the dashboard JSON with its test. In `openspec/specs/engine-replicas/spec.md`, add `striker.engine.job.claims` to the Code map as the way to see claim distribution. Verify with `openspec validate expand-observability-dashboard --strict`.

## 8. Verification

- [ ] 8.1 Run the stack locally with `OTEL_METRICS_ENABLED=true` and two engine replicas, exporting to a local OTel Collector and Prometheus. Confirm the following:
  - every new series appears;
  - a live match shows up in `striker_live_connections`;
  - stopping both engines for 10 minutes raises `striker_matches_overdue`;
  - one season rollover draws its annotation.
- [ ] 8.2 Economy check: `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` is unchanged, because the flow counter does not alter money movement.
- [ ] 8.3 Backend: `go build ./... && go vet ./... && go test ./...` in `backend/`.
- [ ] 8.4 Frontend (untouched, as a safety net): `npx tsc --noEmit && npx oxlint && npx vitest run` in `frontend/`.

## Workflow follow-up

- Archive with `/opsx:archive expand-observability-dashboard` after merge, which creates `openspec/specs/observability/spec.md` with its Code map.
