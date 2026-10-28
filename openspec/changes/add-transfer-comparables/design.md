# Design

## Context

- `player_transfers` (`init78_player_transfers.sql`) already records every completed move
  (`player_id`, `from_team_id`, `to_team_id`, `fee`, `kind`, `created_at`), written by
  `common.RecordPlayerTransfer` inside the same transaction as each flow (`ExecuteTransfer`,
  `AcceptOffer`, `ExerciseLoanBuyOption`, `SignFreeAgent`, `AcceptLoan`, `ReleasePlayer`,
  `AdvancePlayerContract`). It holds nothing about what kind of player moved.
- Public vs private player data: `Player.RedactSkills` zeroes the seven ratings, traits,
  stamina and form of someone else's unlisted player; `PlayerDetailsHandler` still exposes
  `BestPosition` (`common.BestCorePosition`), `AverageSkill` (`common.AverageSkill`) and `age`
  to every visitor. Those three are the only quality signals this change may use.
- `TransferOfferDialog` is opened from `PlayerProfileRail`, `SquadList` and
  `TransferMarketPage` with just a `Player` (and an optional `askingPrice`); the player-details
  payload is not loaded in all three places.

## Goals / Non-Goals

**Goals:**
- Match comparables by what the player was at sale time, never by his current private ratings.
- One cheap indexed query per dialog open (two at most when the band is broadened).
- No change to any money-moving flow beyond the extra columns `RecordPlayerTransfer` writes.

**Non-Goals:**
- A general "market value" estimate or any use of comparables by CPU teams.
- Back-filling snapshots for historical rows.

## Decisions

### 1. Snapshot columns on `player_transfers`, not a join to current `players`
Migration `backend/resources/sql/init88_player_transfer_snapshot.sql` adds nullable
`position smallint`, `age smallint`, `average_skill smallint` (`ADD COLUMN IF NOT EXISTS`) and a
partial index `player_transfers_comparables_idx ON player_transfers(position, created_at DESC,
transfer_id DESC) WHERE kind IN (0, 1, 4) AND fee > 0 AND average_skill IS NOT NULL`
(`CREATE INDEX IF NOT EXISTS`). No back-fill: old rows stay NULL and are excluded by the
predicate. See §7 for why the index has this exact shape.

*Alternative rejected:* matching on the comparable's current skills by joining `players`. It
would compare against today's player rather than the one sold, and the result itself would
leak private data ("this unlisted player now sits in the 60-64 band"). Snapshotting means the
only ratings ever read are the ones taken at the moment of sale, and only as the two public
aggregates.

### 2. `RecordPlayerTransfer` takes the snapshot itself
Before inserting, it reads `goal_keeper, defense, "lateral", pass, midfield, dribbling, shot,
age` for the player through the same `sqlExecutor` (so inside the caller's transaction, after
the row was locked) and computes `BestCorePosition` and `AverageSkill` in Go, then inserts
them with the existing columns. The signature stays the same, so the seven callers do not
change. Every kind gets a snapshot (harmless for loans/releases and keeps the rule simple).

*Alternatives rejected:* passing a `Player` from each caller (seven call sites, easy to pass a
stale struct); computing in SQL (would duplicate `BestCorePosition`'s lateral/midfield/striker
formulas, which live in `common/lineup.go`).

### 3. Query shape
`const getTransferComparablesStmtSql` selects from `player_transfers t` joined to `players p`
(first/last name only) and `LEFT JOIN teams` twice for club names, filtered by
`t.kind IN (PLAYER_TRANSFER_LISTING, PLAYER_TRANSFER_OFFER, PLAYER_TRANSFER_BUY_OPTION)`,
`t.fee > 0`, `t.position = $pos`, `t.age BETWEEN`, `t.average_skill BETWEEN`,
`t.created_at > now() - interval '60 days'`, `t.player_id <> $target`, ordered
`created_at DESC, transfer_id DESC LIMIT 8`. `GetTransferComparables(dbCtx, player)` runs it
with the narrow band (age ±2, average ±4); if it returns fewer than 3 rows it runs it again with
the wide band (age ±4, average ±8) and sets `Broadened`. Constants live next to the query:
`ComparablesWindowDays = 60`, `ComparablesLimit = 8`, `ComparablesMinNarrow = 3`, and the four
band widths. The partial index plus the 60-day window bound the scan regardless of history size.
`kind IN (0, 1, 4)` and `fee > 0` are written as SQL literals, not bind parameters (see §7).

The selected columns are an explicit allow-list; the response struct (`TransferComparable`)
has no field for any rating, trait, stamina, form, wage or contract, so nothing private can be
serialized even by mistake.

### 4. Summary computed server-side
`TransferComparablesResult{Position, Age, AverageSkill, Broadened, Count, MedianFee,
MinFee, MaxFee *int64, Comparables []TransferComparable}`: summary fields are pointers, nil when
empty. Computing it in Go keeps the median rule (even count → floor of mean of the middle two)
in one tested place.

### 5. Separate endpoint, fetched only while the dialog is open
`GET /v1/teams/player/:id/transfer-comparables` in `v1TeamGroup` (already behind
`authMiddleware`), handled by `handlers.TransferComparablesHandler`, which loads the target with
`common.GetPlayerById` (404 on `sql.ErrNoRows`, 400 on a bad id) and calls
`GetTransferComparables`. The reference values come from the target's current full row on the
server; only position, age and average are echoed back, so the endpoint reveals nothing the
player page doesn't. Not embedded in `PlayerDetails` because the dialog opens from places that
never load it, and the player page doesn't need it.

Frontend: `useTransferComparables(playerId, enabled)` in `api/hooks.ts` with query key
`['transferComparables', id]`, `enabled: open`, `staleTime` 60 s. A new presentational
`components/TransferComparables.tsx` renders loading (skeleton rows), error, empty, broadened
note, summary line and the list (MUI `List`/`Stack`, names linking to `/player/:id`, position via
`POSITION_LABEL_KEYS`, fees via `formatCurrency`, dates via the existing date formatter). It is
placed under the fee `NumberField` in `TransferOfferDialog` and hidden once `accepted`.
Strings go under `transfers.offerDialog.comparables.*` in `en.json` and `es.json`.

### 6. Every new query is backed by an index on its WHERE and ORDER BY columns
Rule for this change: each SQL statement it adds or changes MUST have its `WHERE`, `JOIN` and
`ORDER BY` columns covered by an index, existing or added in `init88_player_transfer_snapshot.sql`
(never in a later ad-hoc file), and the plan MUST be checked with `EXPLAIN`. Audit of the
statements this design introduces:

| Statement | Filter / order columns | Index |
|---|---|---|
| Snapshot read in `RecordPlayerTransfer` | `players.player_id =` | `players` primary key (exists) |
| Snapshot insert | none (INSERT) | adds one entry to the new partial index only for eligible rows |
| `getTransferComparablesStmtSql` | `kind`, `fee`, `average_skill IS NOT NULL` (predicate); `position =`; `created_at >`; `ORDER BY created_at DESC, transfer_id DESC` | new `player_transfers_comparables_idx` |
| …its joins | `players.player_id`, `teams.team_id` | primary keys (exist) |
| `GetPlayerById` in the handler | `player_id =` | primary key (exists) |

### 7. Index shape and how it is verified
- Leading `position` (equality), then `created_at DESC, transfer_id DESC` matching the
  `ORDER BY` exactly, so Postgres walks the index in order and stops after 8 matching rows: no
  sort node, and the 60-day bound is a range condition on the same index.
- `age` and `average_skill` bands are not index columns: they are ranges after an ordered
  column, so they would not narrow the scan; they are filtered on the few rows the ordered walk
  reads. Adding them would only grow the index.
- The partial predicate (`kind IN (0, 1, 4) AND fee > 0 AND average_skill IS NOT NULL`) keeps
  loans, releases, free agents and pre-snapshot rows out of the index entirely. Postgres only
  uses a partial index when it can prove the query implies the predicate at plan time, so the
  query MUST spell `kind IN (0, 1, 4)` and `fee > 0` as literals (built from the
  `PLAYER_TRANSFER_*` constants with `strconv`/`fmt` when the const string is declared), not as
  `$n` parameters. `average_skill BETWEEN $a AND $b` already implies `IS NOT NULL`.
- Verified by `TestTransferComparables_UsesIndex`, following the existing
  `TestCountMatchBacklog_UsesPartialIndexes` pattern (`common/match_backlog_integration_test.go`):
  inside a transaction, `SET LOCAL enable_seqscan = off`, `EXPLAIN` the statement with both
  bands, and assert the plan names `player_transfers_comparables_idx`, contains no
  `Seq Scan on player_transfers` and no `Sort` node for the transfer rows.
- If implementation ends up adding or reshaping any other query (for example a count, or a
  different filter), it goes through the same audit: add its row to the table in §6, add the
  index to `init88` if no existing one covers it, and extend the EXPLAIN test.

*Alternative rejected:* a plain (non-partial) index on `(position, created_at)`. It would also
hold every loan, release and free-agent row and every pre-snapshot row, and the planner would
still have to filter them out row by row.

### 8. Nothing else touched
No `TRANSACTION_*` reason, no notification key (so no `notificationtranslations.go` entry), no
engine job (so no replica-safety work), no CPU path change.

## Risks / Trade-offs

- [Empty lists right after deploy, since history has no snapshots] → the empty state is
  explicit and the offer still works; the list fills within days on an active server.
- [Snapshot average taken while a match's live replay is still running could reflect post-match
  skill drift early] → it exposes exactly what the player page's public `averageSkill` already
  shows at that moment, so no new leak; noted, not mitigated further.
- [Extra SELECT inside every transfer transaction] → one primary-key read on an already-locked
  row; negligible.
- [Sparse markets (few transfers per position) give broadened or empty results] → the broadened
  flag and note make that visible instead of silently showing distant players as "similar".

## Code map (for the main spec)

Archive does not carry a delta's Code map; copy this section into
`openspec/specs/transfer-comparables/spec.md` after archiving.


- `backend/resources/sql/init88_player_transfer_snapshot.sql` (snapshot columns, index).
- `backend/common/playertransfers.go` (`RecordPlayerTransfer` snapshot,
  `GetTransferComparables`, band and window constants).
- `backend/handlers/playerHandlers.go` (`TransferComparablesHandler`),
  `backend/service/fmservice.go` (route).
- `frontend/src/components/TransferComparables.tsx`, `TransferOfferDialog.tsx`;
  `frontend/src/api/hooks.ts` (`useTransferComparables`), `api/types.ts`.

## Migration Plan

1. Deploy `init88_player_transfer_snapshot.sql` (re-runnable: `ADD COLUMN IF NOT EXISTS`,
   `CREATE INDEX IF NOT EXISTS`) together with the backend that writes the columns.
2. Ship the frontend; an old frontend simply never calls the new endpoint.
3. Rollback: revert the code; the nullable columns and the index can stay.
