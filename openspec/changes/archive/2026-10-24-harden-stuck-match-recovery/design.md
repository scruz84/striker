# Design

## Context

See proposal.md (Why) for the failure modes. These are the parts of the current code the design
builds on:

- **Claims.** League matches and friendlies are claimed by `GetSimulableMatches` /
  `GetSimulableFriendlies`, which set `in_simulation` and `in_simulation_since` in the
  transaction that selects them `FOR UPDATE`. Recovery (`GetStuckMatches` / `GetStuckFriendlies`)
  re-claims rows that were stamped at least `StuckSimulationAge` (10 min) ago, in one statement.
  Cup ties have no claim: `simulateCupTieOnce` takes a session advisory lock on
  `NamespacedLockKey(LockNamespaceCupTieSimulation, tieId)` and re-reads `played` inside it. A
  failing tie is therefore retried every minute.
- **Saving a result.** `simulateMatch` calls `UpdatePlayedMatch` (errors are logged and
  swallowed), then `UpdateClassification`, `UpdateTeamFinancesAfterMatch`, `PayMatchdayWages`
  and the report flavor, as separate statements. `SetFriendlyPlayed` and `SetCupTiePlayed` write
  `played = true` first and then insert events and instants, all outside a transaction.
- **Post-match effects.** `BroadcastMatchLive` runs `onComplete` (`UpdatePlayerAvailability`,
  `updateSkillsAfterPlayedMatch`, the result notification) at the end of an in-memory goroutine.
  These effects need `MatchSimulation` state that is never persisted:
  `PendingUnavailabilities`, `Home/AwayMinutesByPosition`, and the resolved lineups.
- **Admin.** `CountStuckSimulations` counts every row with `in_simulation = true` and
  `played = false`, with no age filter. `AdminFinishStuckMatchesHandler` returns how many rows it
  attempted.
- **Standings.** `classification` is only ever changed by incrementing it per match
  (`UpdateTeamClassificationStmtSql`) or by resetting it at rollover. A row can therefore be
  rebuilt exactly from the current season's played league matches.

## Goals / Non-Goals

**Goals:**
- No fixture can stay unplayed for more than about an hour after its first attempt, whatever
  its simulation does.
- A result is never booked twice, whether that is standings, money, events or effects.
- Post-match effects survive an engine restart.
- Every new step is safe under several engine replicas (engine-replicas spec).

**Non-Goals:**
- Making the replay itself resumable. A dead broadcast is still closed by
  `RecoverStaleLiveBroadcasts` with the final score.
- Recovering ticket revenue or sponsorship lost when the process dies between the result commit
  and the budget update.
- Making ledger writes part of the result transaction. `AdjustTeamBudget` opens its own
  transaction and is the only path for money, so it stays separate.

## Decisions

### 1. One result transaction, guarded by `played = false`

Add `common.SaveLeagueMatchResultTx`, `SaveFriendlyResultTx` and `SaveCupTieResultTx`. Each one
runs inside a single `tx`:

1. `UPDATE <table> SET played = true, in_simulation = false, home_goals, away_goals[, winner…]
   WHERE <id> = $1 AND played = false`. If no row is affected, roll back and return
   `ErrAlreadyPlayed`, which the caller treats as "another simulation won" and stops.
2. `DELETE` any events and instants left by earlier failed attempts, then insert the new ones,
   with every insert error returned.
3. For a league match, take `pg_advisory_xact_lock(NamespacedLockKey(LockNamespaceStandings,
   leagueId))` and apply the classification increments. For a World Cup group friendly, update
   its group standings in the same transaction.
4. Insert the pending post-match row (Decision 4).
5. Commit.

Only after a successful commit does the caller book money (`UpdateTeamFinancesAfterMatch`,
`PayMatchdayWages`, `awardCupRoundPrize`, `creditFriendlyTicketRevenue`), write the report
flavor and start `BroadcastMatchLive`.

*Alternative considered:* keep separate statements and add idempotency keys to the ledger. That
is more invasive, and it still double-counts standings.

### 2. Attempt bookkeeping and release on failure

Migration `init86_simulation_failures.sql` adds three columns to `matches`, `friendly_matches`
and `cup_ties`: `simulation_attempts int NOT NULL DEFAULT 0`, `last_simulation_error text` and
`last_simulation_attempt_at timestamptz`.

- **Claim.** Every claim statement sets `simulation_attempts = simulation_attempts + 1` and
  `last_simulation_attempt_at = now()`. That covers the live claim, the stuck re-claim, and a cup
  tie's first statement inside its lock (`UPDATE cup_ties … WHERE tie_id = $1 AND played =
  false`).
- **Selectors.** All three add `simulation_attempts < MaxSimulationAttempts` (3) and
  `(last_simulation_attempt_at IS NULL OR last_simulation_attempt_at <= now() - interval '10
  minutes')`.
- **Failure.** On an error, and on a panic inside the existing `recover()`,
  `common.RecordSimulationFailure(kind, id, err)` sets `in_simulation = false` and stores
  `last_simulation_error` (truncated to 500 chars). The retry then comes from the per-minute
  tick after the 10-minute backoff, not only from the 15-minute recovery.
- **Connection errors.** If `common.IsConnectionError(err)` is true (`driver.ErrBadConn`,
  `sql.ErrConnDone`, `context.DeadlineExceeded` while waiting for a pool connection, a
  `*pgconn.ConnectError`, or SQLSTATE class `08`, `57P01`, `53300`), the same write also does
  `simulation_attempts = GREATEST(simulation_attempts - 1, 0)`. If the database is fully down,
  claims fail too and nothing is counted. So only failures with a working database move a
  fixture towards the default.
- **Crashes.** A crash leaves the row claimed. Stuck recovery re-claims it (and counts the
  attempt) as it does today, so a deterministic crash is bounded too.

### 3. Default 0-0 result

`engine.SettleDefaultResults` runs at the start of `FinishStuckSimulationsJob` (every 15 min)
and also from the admin action. For each table it claims candidates with one statement:

```sql
UPDATE matches SET in_simulation = true, in_simulation_since = now()
WHERE played = false AND simulation_attempts >= 3 AND match_date <= now()
  AND (in_simulation = false OR in_simulation_since <= now() - interval '10 minutes')
RETURNING ...
```

Each claimed row is then written through the Decision 1 save function with goals 0-0, one
`MATCH_DEFAULT_RESULT` event (new `common.MATCH_DEFAULT_RESULT = 7`, minute 0, `TeamOwner` =
home, `OtherTeam` = away) and no instants. After the commit the effects depend on the kind:

- **League match:** `PayMatchdayWages` for both sides. No `UpdateTeamFinancesAfterMatch` and no
  flavor. The pending post-match row has an empty skill payload, so availability is still
  counted down and the result notification is sent.
- **Cup tie:** the winner is drawn by `rand.Float32() < 0.5`, `went_to_penalties = true`, and
  `awardCupRoundPrize` pays the winner. The notification uses the existing
  `notifications.messages.cupTieResultPenalties` key.
- **Friendly:** the World Cup group draw is written inside the transaction. No gate split. The
  friendly result notification is sent.

The live channel gets one `complete` 0-0 message, so the viewer and dashboard settle at once.
The pending row's `due_at` is set to `now()`, so the next 5-minute recovery applies it.

This adds no new notification key and no new `TRANSACTION_*` reason: wages keep using
`TRANSACTION_WAGES`, and the prize keeps `TRANSACTION_PRIZE_MONEY`.

### 4. Pending post-match effects (outbox)

New table in the same migration:

```sql
CREATE TABLE IF NOT EXISTS pending_post_match_effects (
  kind text NOT NULL,           -- 'league' | 'cup' | 'friendly'
  match_id bigint NOT NULL,
  due_at timestamptz NOT NULL,  -- match_date + LiveBroadcastWindow (2h)
  payload jsonb NOT NULL,
  PRIMARY KEY (kind, match_id));
CREATE INDEX IF NOT EXISTS pending_post_match_effects_due ON pending_post_match_effects (due_at);
```

`payload` is a new `common.PostMatchPayload`. It holds the home and away ids, the goals, the
went-to-penalties flag, `PendingUnavailabilities`, both `MinutesByPosition` maps, and the player
ids that featured. That is enough to rebuild what `UpdatePlayerAvailability`,
`updatePlayerSkillsAfterMatch` and the notifications read. Player rows are re-read when the
effects are applied.

- **Claim.** `DELETE FROM pending_post_match_effects WHERE kind = $1 AND match_id = $2 RETURNING
  payload`, so whoever deletes the row applies it, once. `onComplete` uses this claim.
- **Recovery.** `LiveBroadcastRecoveryJob` (5 min) also drains due rows with `DELETE … WHERE
  (kind, match_id) IN (SELECT … WHERE due_at <= now() ORDER BY due_at LIMIT 200 FOR UPDATE SKIP
  LOCKED) RETURNING …`.
- **Trade-off.** Delivery is at most once: a crash between the claim and the effects loses them.
  This is preferred to at-least-once, because applying them twice would double an injury
  countdown and send two notifications.

*Alternative considered:* re-deriving the effects from stored events and lineups. That loses
the minutes-by-position data that skill growth needs, and duplicates simulation logic.

### 5. One-off standings repair

`engine.RepairStandingsOnce` runs at engine start, after `EnsureSchema`. It is guarded by
`common.ClaimEngineJobSlot(dbCtx, "standings_repair_v1", time.Unix(0, 0), 24*time.Hour)`, a
fixed slot that only one replica ever wins. For each league with a mismatch, it takes the
Decision 1 standings lock and rebuilds the mismatched rows with one `UPDATE classification c
SET … FROM (aggregate of played matches for c.team_id, c.league_id, c.season) WHERE c.matches
<> agg.matches`. `CountStandingsMismatches` feeds the admin overview, so any leftover drift
stays visible after the one-off run.

### 6. Admin overview and action

- `CountStuckSimulations` is replaced by `GetSimulationHealth`, which returns:
  - `stuck`: rows claimed at least 10 min ago;
  - `failing`: unplayed rows with `simulation_attempts > 0` and a `last_simulation_error`, as
    a list of kind, id, teams, kickoff, attempts and error, capped at 50;
  - `standingsMismatches`.
  All three cover cup ties.
- `POST /v1/admin/finish-stuck-matches` runs `SettleDefaultResults` and then
  `FinishStuckSimulations`. It answers with `{resimulated, defaulted, stillFailing}` per kind.
  The frontend shows those counts, and a "failing matches" table in `OverviewTab`.
- New locale keys go in `admin.overview.*` (`failingMatchesTitle`, `attempts`, `lastError`,
  `standingsMismatchAlert`) and in `admin.actions.finishStuckMatchesSuccess`, which is reworded
  for the new counts. `match.commentary.defaultResult` and the events-report label for type 7
  go in both `en.json` and `es.json`.

### 7. Migration `init86_simulation_failures.sql`

The migration is re-runnable:

- the three-column `ADD COLUMN IF NOT EXISTS` on each table, and the outbox table and index;
- seeding of already-stuck rows, wrapped in `DO $$ … IF NOT EXISTS (SELECT 1 FROM
  information_schema.columns WHERE table_name = 'matches' AND column_name =
  'simulation_attempts') …` around the `ALTER` and the `UPDATE`.

The seeding sets `simulation_attempts = 2` on unplayed rows claimed more than 1 hour ago (or
never stamped), for `matches` and for `friendly_matches`. It therefore runs only on the
start-up that adds the column, and a later long outage is never mistaken for an old stuck row.
Cup ties cannot be detected this way, because a failing tie was never claimed. They start
counting from zero, which means the default after about 30 minutes.

## Risks / Trade-offs

- **[Risk] A bug in the save path defaults a whole round to 0-0.** → Connection errors are not
  counted. The admin list shows the errors before the third attempt. `MaxSimulationAttempts`
  and the backoff are constants that can be raised.
- **[Risk] Rolling deploy: an old-build replica still uses the non-atomic path and skips the
  standings lock.** → `played = false` guards the new save, so an old replica that loses the
  race only double-books if it wins. The standings repair can be re-run by bumping the slot name
  (`standings_repair_v2`).
- **[Risk] Rolling deploy: matches simulated by the old build have no outbox row.** → They keep
  today's behaviour, and only a restart during their replay loses their effects. This is a
  one-time window.
- **[Trade-off] Money is booked after the commit.** A crash in between loses gate revenue or
  wages once rather than ever booking them twice.
- **[Trade-off] At-most-once post-match effects.** See Decision 4.

## Migration Plan

1. Deploy. `EnsureSchema` applies `init86_simulation_failures.sql` and seeds the already-stuck
   rows with 2 attempts.
2. The first per-minute tick, or the recovery job, gives each seeded row one attempt on the new
   code. A row that fails again is settled 0-0 by the next `SettleDefaultResults`, within about
   25 minutes.
3. The standings repair runs once on the first replica to start, and the admin overview should
   then show 0 standings mismatches.
4. Rollback: the old build ignores the new columns and the outbox table. Rows already defaulted
   stay played, and pending outbox rows are simply not drained. Re-deploying the new build
   drains them.

## Open Questions

None that change the specs. The owner already chose the default 0-0, and the proposal's two
remaining questions are written into the specs with their proposed answers: the defaulted
matchday counts down availability, and connection errors are not counted. If the owner
overrides either one, only one scenario and one task change.
