# Proposal

## Why

The per-minute instants (`match_instants`, `cup_tie_instants`, `friendly_instants`) are about 94%
of the data a season stores: roughly 96 rows of ~1.4-1.9 KB per match. With 25,000 teams that is
around 40-55 GB per season, and nothing ever deletes them, so the database grows without bound. Their
only use is the minute-by-minute zone duel and the rating scrubber of a finished match, which
matter for recent matches and hardly at all for matches several seasons old. An admin should be
able to bound that growth, without anybody losing the result, report, lineups or ratings of an old
match.

## What Changes

- New global setting, **off by default**: archiving of old seasons' instants, plus how many
  finished seasons keep their instants (default 1, range 0-10).
- An admin turns it on and sets the count in the admin web (Rules tab), through a new admin
  endpoint. The admin overview returns both values.
- While it is on, a daily engine job deletes the instants of every fixture older than the seasons
  kept: league matches and domestic cups/playoffs by their country's season, regional and
  international cups by their edition within their region, friendlies by kickoff date against the
  start of their home team's country's oldest kept season. Rows are deleted in small batches.
- Nothing else about a match is deleted: the result, events, ratings, lineups, substitutions,
  orders and flavor stay. An archived match shows in the viewer like a match from before instants
  were stored. Its Tácticas tab shows only the lineups pitch, and its Notas tab shows the stored
  ratings. Scrubbing to an earlier minute rebuilds ratings without the zone component.
- Turning the setting off stops the job. Instants already deleted are not restored.
- The instants' only readers are the live replay and the finished match's zone duel and rating
  scrubber, which matter for a few days after kickoff. A second setting, **on by default**, keeps
  a played fixture's instants for a number of days (default 7, range 0-365, 0 keeps them for
  good). The same daily job deletes the instants of every older league match, cup tie and
  friendly whether the season archive is on or off, which bounds the instants tables to about a
  week of matches. An admin sets the days in the Rules tab through
  `PUT /v1/admin/instant-retention`. The season archive remains for an admin who sets the days to 0.
- The same daily job also prunes two append-only logs nothing reads past a short window, with
  fixed retentions and no setting: team-level ledger rows older than 180 days (the finance page
  only reads a team's latest 50, and every club, CPU ones included, writes several per match) and
  transfer offer log rows older than 30 days (only the last day is counted). Ledger rows about a
  player stay, since they are his transfer history.

## Capabilities

### New Capabilities
- `match-data-retention`: which stored match data is kept and for how long. Covers the instant
  archive setting, which fixtures it archives and what survives archiving.

### Modified Capabilities
- `engine-replicas`: the periodic jobs table gains the new daily `instant_archive` job slot.

## Impact

- Schema: new migration adding `season.archive_instants_enabled` (boolean, default false) and
  `season.archive_keep_seasons` (smallint, default 1); a later one (`init121`) adds
  `season.instants_keep_days` (smallint, default 7).
- Backend: `common` settings getters/setters and batched delete statements; new
  `engine.InstantArchiveJob` scheduled daily; admin handler `PUT /v1/admin/instant-archive`; new
  fields on `common.AdminOverview`.
- Frontend: a new form in `features/admin/RulesTab.tsx`, a new hook and types, and en/es strings.
- Disk: deleted rows free space inside Postgres for new rows, so the tables stop growing. They do
  not shrink on disk unless an operator runs `VACUUM FULL` or `pg_repack`, which is out of scope.

## Out of scope

- Moving instants to cold storage or a file export before deleting them.
- Archiving anything other than instants (events, ratings, lineups, notifications, ledgers).
- Shrinking files on disk (`VACUUM FULL`).
- Telling the viewer that a match's detail was archived (it shows like any match without instants).

## Open questions for the owner

- Defaults: assumed off, keeping 1 finished season once on (the current season is always kept). Change
  them if a different default is wanted.
