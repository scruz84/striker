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

## Risks / Trade-offs

- [Turning it on over many seasons deletes millions of rows] → batches of 200 fixtures (~19k
  instant rows) per short transaction. Autovacuum reclaims the space for reuse, and the job can be
  stopped at any time by turning the setting off.
- [Deleted instants cannot come back] → off by default, the UI warns, and only admins can change it.
- [Disk does not shrink] → documented. The tables stop growing once the archive keeps pace.
- [A friendly whose country has no league match in season S − K is never archived] → only possible
  for a country still in its first K seasons. Its friendlies are archived once it gets there.

## Migration Plan

Deploy normally: `EnsureSchema` applies `init118` on startup (metadata-only column adds plus small
partial indexes). Nothing changes until an admin turns the archive on. Rollback: turn it off. The
columns are harmless if the code is reverted.
