# Design

## Decisions

### Starting order (`classification.seed`)

- **New column, not a reorder of rows.** `classification.seed int NULL` holds a league's starting
  order, 1..N. Postgres has no row order, so the order has to be a value the queries sort by.
- **Computed at rollover, inside the existing per-country advisory lock.**
  `advanceCountrySeasonIfComplete` already holds `pg_try_advisory_lock(hashtext(country))` (see
  engine-replicas "Season rollover once per country"), so no new lock is needed.
  `applyPromotionRelegation` already has every league's final ranked standings (`results`) and the
  moves. It returns them, and a new pure function
  `engine.ComputeSeasonSeeds(results, moves) map[int64]map[int64]int` (league to team to seed)
  builds the new order:
  1. Teams relegated into the league, ordered by their final position in the league they left,
     then points, goal difference, goals scored (the promotion-candidate ordering in
     `swapAdjacentTiers`). Several upper groups can send teams to one lower league.
  2. Teams that stayed, in their final order.
  3. Teams promoted into the league, same ordering as group 1, so the best promoted team is
     second to last and the other one last.
  After `ResetClassificationForSeason`, the rollover writes the seeds with
  `UpdateClassificationSeedStmtSql` (`UPDATE classification SET seed = $1 WHERE league_id = $2 AND
  team_id = $3`). The reset leaves `seed` untouched. A crash between the two steps leaves the old
  seeds in place, which is no worse than today. A rerun writes the same values.
- **Last tie-break everywhere.** `seed ASC NULLS LAST, team_id ASC` is appended to
  `GetDivisionDetailsStmtSql`, `GetTeamClassificationPositionStmtSql` and
  `GetRankedClassificationStmtSql`, and `ClassificationPosition` (`common/fmstructs.go`) gains `seed` so the frontend's live
  revert sort in `DivisionDetailsPage.tsx` (`hideStillLiveResults`) can apply the same tie-break.
  `team_id` makes the order total even for a league with no seeds yet (a brand-new league, or a
  team added mid-season).
- **No new notification, no economy effect.** The order only affects display and exact ties.

### Final tables in `season_history`

- **Columns on the existing table, not a new one.** `season_history` is already keyed by
  (team, season) and records `league_id`, `league_name`, `final_position`, `promoted`,
  `relegated`. Adding `matches, points, wins, draws, losses, goals_scored, goals_received`
  (all `int NULL`) makes one row a full table line. A past table is
  `WHERE league_id = $1 AND season = $2 ORDER BY final_position`. A new index
  `season_history_idx2 (league_id, season)` serves it.
- **Filled at rollover.** `GetRankedClassification` keeps returning only what promotion needs.
  `recordSeasonHistory` reads the full rows with a new `GetFinalClassificationStmtSql` (same
  ordering as `GetDivisionDetailsStmtSql`) and `RecordSeasonHistoryLeagueResult` takes the
  counters.
- **Backfill from matches.** Played league matches keep their `league_id` and `season`, so the
  migration rebuilds each past row with the same aggregation `standingsrepair.go` uses, only
  where `matches IS NULL`, so it can be rerun.

### Migration `init101_league_season_history.sql`

Re-runnable, in this order:
1. `ALTER TABLE classification ADD COLUMN IF NOT EXISTS seed int`.
2. `ALTER TABLE season_history ADD COLUMN IF NOT EXISTS ...` for the seven counters, and
   `CREATE INDEX IF NOT EXISTS season_history_idx2`.
3. Backfill the counters (guarded by `matches IS NULL AND league_id IS NOT NULL`).
4. Backfill `seed` for each league's current season where every row of that league is `NULL`.
   Each row's group comes from its team's `season_history` row for `season - 1`: relegated = 0,
   neither = 1, promoted = 2, no history = 3. Then `ROW_NUMBER() OVER (PARTITION BY league_id
   ORDER BY group, final_position, points DESC, goals_scored - goals_received DESC,
   goals_scored DESC, team_id)`.

### Endpoints

All sit in the authenticated `v1TeamGroup` next to the division details, with no ownership check.
- `GET /v1/teams/division/:id/seasons` returns `{ "current": 3, "past": [2, 1] }`. `past` is
  the distinct `season_history` seasons for the league, newest first.
- `GET /v1/teams/division/:id/seasons/:season` returns `{ id, name, season, totalWeeks,
  positions: [{ position, teamId, name, matches, points, wins, draws, losses, goalsScored,
  goalsReceived, promoted, relegated }] }`. `name` is the recorded `league_name`; team names are
  current. `totalWeeks` = 2 × (rows − 1). A season with no rows is 404, and so is the current
  season, which is not in history yet. A past table never changes, so it is cached in
  `ObjectCache` with no expiry, keyed by league and season.
- `GET /v1/teams/division/:id/weeks/:week?season=N` passes N to the existing `WeekMatches`. The
  cache key already includes the season. With no `season`, it uses the current season as today.

### Frontend

- **Season picker** (a small `Select` above the standings, shown only when `past` is not empty).
  The chosen past season is kept in `?season=N`. The current season removes the parameter.
  `week` and `season` combine.
- **Past-season mode:** the standings come from `/seasons/:season`, with no live revert. Zones
  come from `promoted`/`relegated`, not from the slot counts, and a trophy marks position 1. The
  matchday navigator uses that season's `totalWeeks`, defaults to its last pair (T−1, T) with no
  chips, and requests `/weeks/:week?season=N` for both cards. The subtitle reads "Season N · Final
  standings". Top scorers and recent transfers are hidden, and on phones only the "Tabla" and
  "Jornadas" tabs are shown.
- **Team history links:** each season row in `TeamDetailsPage`'s history tab (`TrophyCabinet`)
  that has a league id links to `/division/:leagueId?season=N`. `SeasonHistoryEntry` gains
  `leagueId`.

## Code map

- Migration: `backend/resources/sql/init101_league_season_history.sql`.
- Rollover: `backend/engine/season.go` (`advanceCountrySeasonIfComplete`,
  `applyPromotionRelegation`, `recordSeasonHistory`), `backend/engine/seasonseeds.go`
  (`ComputeSeasonSeeds`).
- Queries: `backend/common/database.go` (`GetDivisionDetailsStmtSql`,
  `GetTeamClassificationPositionStmtSql`, `GetRankedClassificationStmtSql`,
  `UpdateClassificationSeedStmtSql`, `GetFinalClassificationStmtSql`),
  `backend/common/seasonhistory.go` (`RecordSeasonHistoryLeagueResult`, `GetLeagueSeasons`,
  `GetLeagueSeasonTable`).
- Handlers and routes: `backend/handlers/seasonHistoryHandlers.go` (`LeagueSeasonsHandler`,
  `LeagueSeasonTableHandler`), `backend/handlers/statsHandlers.go` (`LeagueWeekMatchesHandler`
  `season` query), `backend/service/fmservice.go`; cache in `backend/engine/catalog.go`.
- Frontend: `frontend/src/features/division/DivisionDetailsPage.tsx`,
  `frontend/src/features/team/TeamDetailsPage.tsx`, `frontend/src/api/` (hooks, types),
  `frontend/src/i18n/locales/{en,es}.json` (`division.season*`).
- Tests: `backend/engine/seasonseeds_test.go`, a rollover integration test (seeds and history
  counters), the migration backfill test, handler tests, `DivisionDetailsPage` vitest tests.

After archiving, copy this Code map into `openspec/specs/league-page/spec.md` and widen its
Purpose to cover past seasons.
