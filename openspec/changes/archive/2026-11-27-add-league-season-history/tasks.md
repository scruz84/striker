# Tasks

## 1. Schema and backfill

- [x] 1.1 Add `backend/resources/sql/init101_league_season_history.sql`: `classification.seed`, the seven `season_history` counters, `season_history_idx2 (league_id, season)`, the counter backfill from played league matches and the `seed` backfill for current seasons (see design); verify "Backfill" and "Deploy mid-preseason" with an integration test that runs the migration twice.

## 2. Starting order

- [x] 2.1 Add the pure `engine.ComputeSeasonSeeds` in `backend/engine/seasonseeds.go`; verify "Promoted teams start at the bottom", "Relegated teams start at the top" and "Promoted from two groups" with unit tests, plus a league with no previous season.
- [x] 2.2 Write the seeds in `advanceCountrySeasonIfComplete` after `ResetClassificationForSeason` (`UpdateClassificationSeedStmtSql`); verify with a rollover integration test, including the existing concurrent-rollover test still advancing by exactly one season.
- [x] 2.3 Append `seed ASC NULLS LAST, team_id ASC` to `GetDivisionDetailsStmtSql`, `GetTeamClassificationPositionStmtSql` and `GetRankedClassificationStmtSql`, expose `seed` on `ClassificationPosition`, and use it in the frontend live revert sort; verify "Matchday 0" and "Exact tie later".

## 3. Final tables and endpoints

- [x] 3.1 Store the seven counters in `recordSeasonHistory` (`GetFinalClassificationStmtSql`, `RecordSeasonHistoryLeagueResult`); verify "Rollover" in the rollover integration test.
- [x] 3.2 Add `GET /v1/teams/division/:id/seasons` and `GET /v1/teams/division/:id/seasons/:season` (cached without expiry), and the `season` query parameter on `/weeks/:week`; verify "Third season", "Season 2 table", "Current season", "Past season week" and "Invalid week" with handler or integration tests.

## 4. Frontend

- [x] 4.1 Add the API hooks and types (`LeagueSeasons`, `LeagueSeasonTable`, `leagueId` on `SeasonHistoryEntry`) and en/es strings.
- [x] 4.2 Add the season picker and the past-season mode to `DivisionDetailsPage` (table, zones, trophy, matchdays, hidden panels and tabs, subtitle); verify "Pick season 2", "First season", "Browse season 2", "Promoted in a past season" and "Past season on a phone" with vitest tests.
- [x] 4.3 Link each season in the team history tab to its past table; verify "Open a past table from the team page" with a vitest test.

## 5. Checks

- [x] 5.1 Run `go test ./...` (with Postgres for the integration tests), `npx tsc --noEmit && npx oxlint && npx vitest run`, and `openspec validate add-league-season-history --strict`; verify all pass and check the 360 px layout of the picker.
