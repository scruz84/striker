# Design

## Context

- Instants are written once per simulated minute by `common` (`InsertMatchInstantStmtSql`,
  `InsertCupTieInstantStmtSql`, `InsertFriendlyInstantStmtSql`) and read back only by the three
  "instants by id" queries behind the finished match viewer (`database.go`, `cups.go`,
  `friendlies.go`). Nothing else reads them: ratings, standings, stats and history use other tables.
- The frontend already handles a played match with no instants (pre-`init89` matches): the
  Tácticas tab drops the zone duel (`Lineups.tsx`), the timeline comes from events
  (`matchTimeline`), and the Notas tab shows the stored ratings. The specs already describe this
  case (match-viewer, "A match without stored instants").
- Seasons are per country (`countries.current_season`). Domestic cups and playoffs carry `country`
  and `season`. Regional and international cups carry `region` and an edition number in `season`.
  Friendlies have no season, only `match_date`.
- Global rules live on the singleton `season` row (`max_friendlies_per_week`,
  `offer_negotiation_window_minutes`, ...), with a getter/setter in `common`, a `PUT /v1/admin/...`
  handler and a form in `features/admin/RulesTab.tsx`.

## Goals / Non-Goals

**Goals:**
- Delete old instants in batches that hold locks and WAL briefly, never touching a match that is
  unplayed, being simulated or within the kept seasons.
- Make each batch's cost stay flat over the years, not grow with the number of already-archived
  matches.
- Make replicas safe even without the slot claim (the slot only avoids wasted work).

**Non-Goals:**
- Reclaiming disk space at the file level (see proposal, Out of scope).
- Telling the viewer that a match was archived (the new flag would make it possible later).

## Decisions

### 1. Migration `init118_instant_archive.sql`
- `season.archive_instants_enabled boolean NOT NULL DEFAULT false` and
  `season.archive_keep_seasons smallint NOT NULL DEFAULT 1 CHECK (0..10)`.
- `instants_archived boolean NOT NULL DEFAULT false` on `matches`, `cup_ties` and
  `friendly_matches`. In Postgres 11+, `ADD COLUMN` with a constant default does not rewrite the table.
- Partial indexes on rows still to archive: `matches (season) WHERE played AND NOT
  instants_archived`, `cup_ties (cup_id) WHERE played AND NOT instants_archived`,
  `friendly_matches (match_date) WHERE played AND NOT instants_archived`. Once the archive runs, each
  index only holds the kept seasons' played fixtures, so it stays small.
- All `IF NOT EXISTS`, re-runnable. No `TRANSACTION_*` reason and no notification key is added.

*Alternative:* a per-country "archived through season" watermark. Rejected because friendlies and
regional cups do not follow country seasons, so they would need watermarks of their own. A per-row
flag is uniform and idempotent. *Alternative:* probe `EXISTS (instants)` on every batch. Rejected:
each batch would rescan every old match, a cost that grows forever.

### 2. One atomic statement per batch
For each kind, one statement marks up to 200 eligible rows archived and deletes their instants:

```sql
WITH batch AS (
  UPDATE matches SET instants_archived = true
  WHERE match_id IN (SELECT m.match_id FROM matches m
                     JOIN leagues l ON l.league_id = m.league_id
                     JOIN countries c ON c.name = l.country
                     WHERE m.played AND NOT m.instants_archived
                       AND m.season <= c.current_season - 1 - $1
                     ORDER BY m.match_id LIMIT 200
                     FOR UPDATE OF m SKIP LOCKED)
  RETURNING match_id)
DELETE FROM match_instants WHERE match_id IN (SELECT match_id FROM batch)
```

The pass repeats it until a batch archives no row, then moves on to the next kind. Cup ties join
`cups`: the domestic branch uses the country's season, the regional branch uses
`max(season) per region`, and both require `cups.completed`. Friendlies join the home team's
country and compare `match_date` with `min(match_date)` of that country's league matches of
season `current_season - K` (`$1` = K). Each is a named `...StmtSql` constant with a small wrapper
returning the number of rows archived.

`SKIP LOCKED` plus the flag make two concurrent passes split the work instead of colliding, so a
long first pass that overlaps the next day's slot is harmless.

### 3. Engine job
`engine.InstantArchiveJob` runs daily at 04:41 UTC, clear of the other daily jobs, and claims
`claimRun(dbCtx, "instant_archive", 24h)`. It reads the setting on every run and returns at once
while the archive is off. It logs the counts per kind and records them in the existing engine job
metrics.

### 4. Admin API and UI
- `common.GetInstantArchiveSettings` / `SetInstantArchiveSettings(enabled, keepSeasons)`, which
  validates 0-10.
- `PUT /v1/admin/instant-archive` with `{ "enabled": bool, "keepSeasons": int }` on the admin group,
  answering 400 for out-of-range values. `common.AdminOverview` gains `instantArchiveEnabled` and
  `instantArchiveKeepSeasons`.
- `RulesTab` gains an `InstantArchiveForm`: a switch, a number field (0-10), and a note that the
  deletion is permanent and keeps results, chronicle and ratings. Strings go under
  `admin.instantArchive.*` in `en.json` / `es.json`.

### 5. Friendly retention by days
Friendlies have no season, are played every week by a large share of the clubs, and their detail is
only replayed right after the match. Keeping them for whole seasons (the archive's friendly rule)
would still store tens of thousands of them per week, and only while the admin has the archive on.

- Migration `init121_friendly_instant_retention.sql`: `season.friendly_instants_keep_days smallint
  NOT NULL DEFAULT 7 CHECK (0..365)`, re-runnable.
- `common.ArchiveExpiredFriendlyInstantsStmtSql` is the same one-statement batch shape as §2, over
  played, not yet archived club friendlies with `match_date < now() - keepDays days` (home team not
  `is_national_team`, which also leaves out World Cup group matches), ordered by `match_date` so it walks the existing
  `friendly_matches_instants_to_archive` partial index. It sets the same `instants_archived` flag,
  so the season rule never looks at those rows again.
- `engine.RunInstantArchive` runs this expiry first on every daily slot, archive on or off, and
  then the season archive if it is on. `0` skips the expiry.
- `common.Get/SetFriendlyInstantKeepDays` (0-365), `PUT /v1/admin/friendly-instant-retention`
  `{ "keepDays": int }` (400 out of range or missing), `AdminOverview.friendlyInstantKeepDays`, and a
  `FriendlyInstantRetentionForm` card in `RulesTab` with strings under
  `admin.friendlyInstantRetention.*`.

*Why on by default:* unlike seasons of league history, a friendly's per-minute detail has no value
after a few days, and without the expiry the friendly instants table grows by the week. *Why national
teams are excluded:* there is one national team per country against thousands of clubs, so
national-team friendlies and World Cup group matches are few next to club friendlies, so the season archive is enough for them, as for league matches and
cups.

### 6. Pruning old log rows
- `common.PruneOldLedgerRows` walks `team_transactions` by primary key in pages of 5,000
  (`PruneLedgerPageStmtSql`), deleting the page's rows with `player_id IS NULL` created before the
  cutoff (`LedgerRetention`, 180 days), and stops after the first page that reaches the cutoff.
  Every ledger row takes `now()` at insert, so ids grow with `created_at`. Walking the primary key
  needs no new index on a table this change is trying to keep small; the only rows walked again
  each day are the old player-tied rows it keeps.
- `common.PruneOldOfferLog` deletes `transfer_offer_log` rows older than `OfferLogRetention`
  (30 days) in batches through the existing `transfer_offer_log_created_idx`.
- `engine.runLogRetention` runs both at the start of every daily `instant_archive` slot. They are
  fixed constants, not admin settings: nothing in the game reads these rows.

*Reviewed and left alone:* league and cup events, lineups and ratings (standings, top scorers,
awards, player history and caps read them); national-team friendly lineups (caps); notifications
(only human users get them, so they are few); `player_progression_log` (one row per player per
season, and a player's page shows its history); `player_offer_rejections` (already pruned);
`engine_job_runs` (one row per job).

## Risks / Trade-offs

- [Turning it on over many seasons deletes millions of rows] → batches of 200 fixtures (~19k
  instant rows) per short transaction. Autovacuum reclaims the space for reuse, and the job can be
  stopped at any time by turning the setting off.
- [Deleted instants cannot come back] → off by default, the UI warns, and only admins can change it.
- [Disk does not shrink] → documented. The tables stop growing once the archive keeps pace.
- [The friendly expiry is on by default, so upgrading deletes club friendlies' instants older than 7 days
  at the next 04:41 UTC slot] → this is the intended bound on growth; an admin who wants them kept
  sets the days to 0 before that slot (or higher, up to 365).
- [Ledger pruning is not a setting, and the first pass on a world older than 180 days deletes most
  of its ledger] → those rows are never read; budgets, the finance page (latest 50 rows) and
  player transfer history do not change.
- [A friendly whose country has no league match in season S − K is never archived] → only possible
  for a country still in its first K seasons. Its friendlies are archived once it gets there.

## Migration Plan

Deploy normally: `EnsureSchema` applies `init118` on startup (metadata-only column adds plus small
partial indexes). Nothing changes until an admin turns the archive on. Rollback: turn it off. The
columns are harmless if the code is reverted.

## Code map for the archived spec

To copy into `openspec/specs/match-data-retention/spec.md` when this change is archived.


- Setting and archive: `backend/common/instantarchive.go` (`InstantArchiveSettings`,
  `Get/SetInstantArchiveSettings`, `ArchiveOldInstants`, the `Archive*InstantsStmtSql` batches,
  `FriendlyArchiveCutoffsStmtSql`, `InstantArchiveBatch`).
- Schema: `backend/resources/sql/init118_instant_archive.sql` (`season.archive_instants_enabled`,
  `season.archive_keep_seasons`, `instants_archived` on `matches` / `cup_ties` /
  `friendly_matches`, the `*_instants_to_archive` partial indexes);
  `backend/resources/sql/init121_friendly_instant_retention.sql` (`season.friendly_instants_keep_days`).
- Log retention: `common/dataretention.go` (`LedgerRetention`, `OfferLogRetention`,
  `PruneOldLedgerRows`, `PruneOldOfferLog`), `engine/instantarchive.go` (`runLogRetention`); tests
  in `common/dataretention_integration_test.go`.
- Friendly expiry: `common/instantarchive.go` (`Get/SetFriendlyInstantKeepDays`,
  `ArchiveExpiredFriendlyInstants`, `ArchiveExpiredFriendlyInstantsStmtSql`),
  `engine/instantarchive.go` (`runFriendlyInstantExpiry`), `AdminSetFriendlyInstantRetentionHandler`
  (`PUT /v1/admin/friendly-instant-retention`), `AdminOverview.friendlyInstantKeepDays`,
  `FriendlyInstantRetentionForm` / `useAdminSetFriendlyInstantRetention`, strings
  `admin.friendlyInstantRetention.*`.
- Engine: `backend/engine/instantarchive.go` (`InstantArchiveJob`, slot `instant_archive`,
  `RunInstantArchive`, daily at 04:41 UTC via `StartInstantArchiveScheduler` in `StartEngine`).
- API: `AdminSetInstantArchiveHandler` (`handlers/adminHandlers.go`, `PUT /v1/admin/instant-archive`),
  `AdminOverview.instantArchiveEnabled` / `instantArchiveKeepSeasons`.
- Frontend: `features/admin/RulesTab.tsx` (`InstantArchiveForm`), `useAdminSetInstantArchive`
  (`api/hooks.ts`), strings `admin.instantArchive.*`. An archived match reads like one without
  instants in `features/match/Lineups.tsx` and `utils/matchMinute.matchTimeline`.
- Tests: `common/instantarchive_integration_test.go`, `engine/instantarchive_integration_test.go`,
  `handlers/instantArchiveHandler_test.go`, `service/security_test.go`
  (`TestInstantArchive_RequiresAdminSession`), `features/admin/RulesTab.test.tsx`.
