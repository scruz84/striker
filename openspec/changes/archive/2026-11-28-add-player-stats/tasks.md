# Tasks

## 1. Schema and backfill

- [x] 1.1 Add `backend/resources/sql/init123_player_match_stats.sql` (table, indexes, `schema_backfills`, one-off backfill from ratings and events); verify it backfills once and a second run inserts nothing.

## 2. Writing

- [x] 2.1 Add `BuildPlayerMatchStatLines` and `replacePlayerMatchStatsTx`, called from `replaceMatchEventsTx`; verify "A league match" and "A club friendly" with `TestBuildPlayerMatchStatLines` and `TestPlayerMatchStats_TotalsAcrossTheReaders`.

## 3. Reading

- [x] 3.1 Rebuild `GetPlayerCareerStats` and add `GetPlayerSeasonStats`; verify "Match still on air", "Friendly goals left out" and "League and friendly in one season".
- [x] 3.2 Add `GetLeagueLeaders` and `GetTeamSeasonStats` with their endpoints; verify "Rating threshold" and "A retired scorer".
- [x] 3.3 Rebuild `GetTopScorers*` and `GetPlayerOfTheSeason` on the stats; verify "Retired in the rollover" and update the tests that inserted bare events (`topscorers`, `playerbirthcountry`, `stoppage_time`).

## 4. Frontend

- [x] 4.1 Add the types, hooks and en/es strings; the profile's season table and career rows; `LeagueLeadersCard` in place of the top scorers (phone tab "Stats", past seasons included); the squad tab's stats view. Verify with vitest tests and a 360 px check.

## 5. Specs and checks

- [x] 5.1 Copy the design's Code map into `openspec/specs/player-stats/spec.md` after archiving and point the season-awards Code map at the stats readers.
- [x] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.
