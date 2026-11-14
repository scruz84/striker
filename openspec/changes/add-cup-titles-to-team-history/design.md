# Design

## Context

`season_history` is one row per (team, season). Two writers upsert it independently: the
rollover (league columns) and `engine.advanceCup` when a cup completes (`cup_result`).
`advanceCup(dbCtx, cupId, season)` gets `cups.season`, which for a domestic cup is the country's
league season and for a regional cup or the World Cup knockout is an edition / cycle number.

## Decisions

### 1. A separate column for the international title
`season_history.international_cup_id bigint REFERENCES cups(cup_id) ON DELETE SET NULL`.
A cup id is enough: region and edition come from `cups`, and the name is translated in the
client from the region (`cup.names.<region>`), like the International Cup page does. `cup_result`
stays as it is (`'Champion'`) for the domestic cup, so no existing row changes meaning.

### 2. Which writer runs for which cup
`advanceCup` takes the whole `common.Cup` (country, region, season) instead of `(cupId, season)`.
When the cup completes:
- `cup.Country != ""` (domestic): `RecordSeasonHistoryCupResult(team, cup.Season, "Champion")`,
  unchanged.
- `cup.Region == "world"` (World Cup knockout): no history row.
- any other region: `season := GetCurrentSeasonForTeam(team)`, then
  `RecordSeasonHistoryInternationalCup(team, season, cup.CupId)`, an upsert that only sets
  `international_cup_id`.

A cup only completes once its final has left its live window (`IsCupRoundResolved`), so the title
is never visible before the final's replay ends. `UpsertSeasonHistoryLeagueStmtSql` lists its own
columns in `DO UPDATE`, so the rollover never clears `international_cup_id`.

### 3. Migration `resources/sql/init111_team_cup_titles.sql`
Re-runnable, in this order:
1. `ALTER TABLE season_history ADD COLUMN IF NOT EXISTS international_cup_id ...`.
2. Clear `cup_result` where no domestic cup (`country IS NOT NULL AND kind = 'cup'`) of the
   team's season has that team as champion. The match on `c.season = sh.season` and the champion
   is exact, because the domestic writer always used `cups.season`.
3. Delete rows with no league, no position, no counters, no cup result, no international cup and
   no promotion/relegation: rows created only by the bug.
4. Backfill: for each completed regional cup (region not null and not `'world'`), take the final's
   `max(cup_ties.match_date)`. The season is the `matches.season` of the champion's latest league
   match on or before that date, or its first one after it when it has none before. Upsert
   `international_cup_id`, only where it is still NULL.

### 4. API
`GetSeasonHistoryStmtSql` LEFT JOINs `leagues` (for `tier`) and `cups` (for region and edition):
`SeasonHistoryEntry` gains `LeagueTier *int` and
`InternationalCup *SeasonHistoryInternationalCup {CupId, Region, Edition}`.

### 5. Frontend
- `TrophyCabinet`: maps `cupResult === 'Champion'` to `team.cupWinner`. Any other stored value is
  shown as it is. A gold chip `team.internationalCupWinner` (`{{cup}}`, `{{edition}}`) links to
  `/international-cup?region=<region>&cup=<cupId>`.
- `HonoursSummary` (same file): counts entries with `finalPosition === 1 && leagueTier === 1`,
  `cupResult === 'Champion'`, and international cups by region in the page's region order.
- `InternationalCupPage`: `useSearchParams` for `region` and `cup`. A tab change writes
  `region` only. `RegionalCup` takes an optional `cupId` and uses it instead of the latest id.

New locale keys (en/es): `team.cupWinner`, `team.internationalCupWinner`, `team.honours`,
`team.honoursLeague`, `team.honoursCup`.

No `TRANSACTION_*` reason or notification key is added.

## Risks

- The backfill season is approximate for a final played between a country's last matchday and
  its rollover (it lands in the season that just ended). That is acceptable and only affects
  editions that already exist.

## Code map (for the new team-history spec at archive)

- Engine: `backend/engine/cups.go` (`advanceCup`, `recordCupTitle`).
- Data: `backend/common/seasonhistory.go` (`SeasonHistoryEntry`, `SeasonHistoryInternationalCup`,
  `RecordSeasonHistoryCupResult`, `RecordSeasonHistoryInternationalCup`, `GetSeasonHistory`).
- Schema: `resources/sql/init111_team_cup_titles.sql` (`season_history.international_cup_id`,
  clean-up and backfill), `history/init15_season_history.sql`, `init101_league_season_history.sql`.
- API: `TeamHistoryHandler` (`handlers/seasonHistoryHandlers.go`).
- Frontend: `features/team/TeamDetailsPage.tsx` (`TrophyCabinet`, `HonoursSummary`); test
  `features/team/TrophyCabinet.test.tsx`.
- Tests: `common/cuptitles_integration_test.go`, `engine/cup_titles_integration_test.go`.
