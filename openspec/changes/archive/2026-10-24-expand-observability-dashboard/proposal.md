# Proposal

## Why

The Grafana dashboard (`backend/resources/grafana/striker-dashboard.json`) shows HTTP traffic,
game activity, the transfer market and per-node process health. It cannot answer the questions
that matter most for this game in production:

- Is the engine keeping up? There is no metric for overdue or stuck matches, for engine job
  duration, or for which replica wins each job, even though the engine now runs as several
  replicas.
- Where does money come from and go? `striker_economy_total_budget` shows the total, but not the
  flows by reason (wages, tickets, sponsors, prizes, upkeep) that the economy tuning works with.
- Is the API saturated? The p95 latency panel is distorted by the live-match websockets, which
  are recorded as requests lasting the whole match. Nothing shows the DB connection pool (capped
  at 50 per process), live viewers or failed web push deliveries.

There are also gaps that need no new code. `striker_ai_market_actions_total` is exported but has
no panel, although the README says the dashboard covers every metric. Goals per match is not
shown, although the README gives it as the reason `striker.goals.scored` exists. 4xx errors are
not visible. The two per-node share panels use different windows.

## What Changes

- **Dashboard, existing metrics:**
  - a 4xx rate panel and a 5xx-by-route table;
  - p50/p95 latency without websocket routes;
  - goals per match by competition;
  - CPU market actions per day by action;
  - a total-money trend alongside the stat;
  - season-rollover annotations;
  - `job` / `instance` template variables for the per-node row;
  - the same 1h window on both per-node share panels.
- **HTTP metrics:** the middleware stops recording `striker.http.request.duration` for websocket
  upgrade requests. They are still counted in `striker.http.requests`.
- **New metrics:**
  - `striker.economy.flow` (counter, `reason` and `direction`): money credited and debited
    through the ledger.
  - `striker.engine.job.runs`, `striker.engine.job.duration` and `striker.engine.job.last_run`,
    labelled by `engine_job` (not `job`, which the OTLP backend reserves for `service.name`), plus
    `striker.engine.job.claims` (`engine_job`, `outcome`=`won`/`lost`) for once-per-period claims.
  - `striker.matches.overdue` and `striker.matches.in_simulation` (gauges, `type`), DB-wide and
    registered by the engine only.
  - `striker.live.connections` (up/down counter per svc instance).
  - `striker.db.connections` (`state`=`in_use`/`idle`), `striker.db.connection_waits` and
    `striker.db.connection_wait_time` per process.
  - `striker.push.sent` (`outcome`=`sent`/`failed`/`expired`).
- **New dashboard rows:** "Engine health" (backlog, job duration and freshness, claims per
  replica), "Economy flows" (credited vs. debited per day by reason, net creation) and "Live &
  delivery" (live viewers, push outcomes, DB pool).
- **Migration** `init85_observability_indexes.sql`: partial indexes on unplayed matches, so the
  overdue gauges do not scan match history on every export tick.

## Capabilities

### New Capabilities
- `observability`: the metrics the svc and engine processes export (names, labels, which process
  registers each one, and its cost limits) and the Grafana dashboard shipped with the repo (rows,
  panels and the questions each one answers).

### Modified Capabilities
<!-- None. Adding metrics changes no game behavior. engine-replicas requirements stay as they are;
     the claim metric only observes them. -->

## Impact

- Backend:
  - `common/telemetry.go` (new instruments, `RegisterMarketGauges` extended, DB pool gauges);
  - `service/fmservice.go` (`metricsMiddleware`);
  - `common/database.go`, `transfers.go`, `offers.go`, `loans.go`, `contracts.go`, `stadium.go`
    (record the ledger flow after commit);
  - `engine/scheduler.go` / `engine/tasks.go` (job wrapper, claim outcome);
  - `handlers/liveMatchHandlers.go` (connection gauge);
  - `common/webpushnotifier.go` (push outcome).
- Migration: `backend/resources/sql/init85_observability_indexes.sql`, re-runnable with
  `CREATE INDEX IF NOT EXISTS`.
- Dashboard JSON and the "OpenTelemetry metrics" section of `backend/README.md`.
- No frontend change, no notification and no `TRANSACTION_*` reason. Money movement is unchanged:
  the flow counter is recorded after the existing ledger write commits.

## Out of scope

- Alert rules and Grafana provisioning (datasource or dashboard as code in docker-compose).
- Tracing and logs (only metrics).
- Business KPIs that need new columns, such as daily active managers. `striker.logins` remains the
  activity signal.

## Open questions

- Who looks at the dashboard, and with what Grafana? The plan keeps the JSON importable
  (`__inputs` `DS_PROMETHEUS`, Grafana ≥ 10). If the target is Grafana Cloud with a different
  OTLP-to-Prometheus naming scheme, the PromQL suffixes need adjusting, as the README already
  warns.
