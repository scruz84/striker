# Tasks

## 1. Backend: matchday matches endpoint

- [x] 1.1 Add `LeagueWeekKey = "leagueWeek_key"` and `GetLeagueWeekMatches(dbCtx, leagueId, season, week)` to `backend/engine/catalog.go`. It wraps `common.WeekMatches`, caches the result with `ObjectCache.SetValue` (1 minute) under `leagueWeek_key<league>_<season>_<week>`, and always returns a non-nil slice. Verify with `go build ./...`.
- [x] 1.2 Add `LeagueWeekMatchesHandler` to `backend/handlers/statsHandlers.go`. It parses `:id` and `:week`, returns 400 for a non-integer id or a week below 1, resolves `common.GetCurrentSeasonForLeague`, and returns 200 with the matches, `[]` included. Split out the param parsing (`parseLeagueWeek`) so it can be unit tested without a database. Register `v1TeamGroup.GET("/division/:"+handlers.ID+"/weeks/:week", ...)` in `backend/service/fmservice.go` next to `/top-scorers`. Verify with a `statsHandlers_test.go` table test over `0`, `-1`, `abc`, `1`, `30` (gin test router, `httptest`): 400 for the first three, and a valid parse for the last two.
- [x] 1.3 Add an integration test (`backend/common/league_week_matches_integration_test.go`, following the guarded `*_integration_test.go` pattern). Seed one league, then check that week 1 returns its fixtures for the current season only, that a week past the last one returns `[]`, and that no row from a previous season leaks in. Verify that `go test ./common -run LeagueWeek` passes against the dev database.

## 2. Frontend: data hook

- [x] 2.1 In `frontend/src/api/hooks.ts`, add `queryKeys.leagueWeek(leagueId, week)` and `useLeagueWeekMatches(leagueId, week, seed?: { matches: WeekMatch[]; updatedAt: number })`. It calls `/v1/teams/division/${leagueId}/weeks/${week}`, uses `initialData` / `initialDataUpdatedAt` from `seed`, `placeholderData: keepPreviousData`, and `enabled: !!leagueId && week >= 1`. Verify with `npx tsc --noEmit`.

## 3. Frontend: matchday navigation

- [x] 3.1 In `features/division/DivisionDetailsPage.tsx`, add a pure exported helper `matchdayPair({ currentWeek, totalWeeks, requested })`. It returns `{ left, isDefault }`, with `left = clamp(requested ?? currentWeek, 1, totalWeeks - 1)` and a minimum of 1 when `totalWeeks < 2`. Add `matchdayChip(week, currentWeek)`, which returns `'lastWeek' | 'nextWeek' | undefined`. Verify with new cases in `DivisionDetailsPage.test.ts` covering W = 0, 12 and 30 with T = 30, `?week=99`, `?week=abc` and `?week=0`, matching the spec's scenarios.
- [x] 3.2 Read and write `week` with `useSearchParams` (`replace: true`, and remove the parameter on the default pair). Seed matchday W_backend = `divisionDetails.classification.positions[0].matches` from `previousWeekMatches` and W_backend + 1 from `nextWeekMatches`, using the division query's `dataUpdatedAt`. Verify with a Testing Library test that mocks `api.get`: rendering mid-season with no `week` makes no `/weeks/` call, and rendering with matchday W_backend still live calls only `/weeks/<W_backend-1>`.
- [x] 3.3 Add `MatchdayNavigator` (‹ `IconButton`, `TextField select` "Jornada 1…T", › `IconButton`, and a "back to current" `Button` shown only off the default; every control at least 44 px). Disable ‹ at L = 1 and › at L = T−1. Change `FixturesCard` to take `week`, `chip?` and `matches | undefined` (a skeleton while loading), with the title `division.matchday` and the chip labels from `division.lastWeek` / `division.nextWeek`. Keep the existing per-match `mightStillBeLive` / `isPast` score rule untouched. Verify with a Testing Library test: tapping › moves "Jornada 12/13" to "13/14", › is disabled at 29/30, "back to current" restores 12/13, and a browsed week with a match kicked off 40 minutes ago shows the LIVE chip and no score.
- [x] 3.4 Place the navigator above the two-column fixtures grid on desktop, and at the top of the "Jornadas" tab on phones. Leave the standings / scorers / transfers grid as it is. Verify by running the dev app (HOWTO_RUN.md) at 1280 px: › changes only the two cards.

## 4. Frontend: four mobile tabs

- [x] 4.1 Extend `DivisionTab` with `'transfers'`. Render four tabs (`table`, `fixtures`, `scorers`, `transfers`) with `sx={{ minWidth: 0, px: 1 }}`. The scorers tab shows only `TopScorers`, and the transfers tab shows only `RecentLeagueTransfers`. Verify with a Testing Library test at a mobile width (`useMediaQuery` false): the "Goleadores" panel has no "Traspasos recientes" heading, and the "Traspasos" tab shows it.
- [x] 4.2 i18n in `frontend/src/i18n/locales/en.json` and `es.json`:
  - Add `division.matchday` ("Matchday {{week}}" / "Jornada {{week}}"), `division.matchdayPicker` ("Matchday" / "Jornada"), `division.previousMatchday` ("Previous matchday" / "Jornada anterior"), `division.nextMatchday` ("Next matchday" / "Jornada siguiente"), `division.backToCurrent` ("Current matchday" / "Jornada actual") and `division.tabs.transfers` ("Transfers" / "Traspasos").
  - Change `division.tabs.table` to "Table" / "Tabla" and `division.tabs.scorers` to "Scorers" / "Goleadores".
  - Verify that both files hold the same new keys (`npx vitest run` covers locale parity if a test exists; otherwise diff the key sets with `jq`).
- [ ] 4.3 Run the mobile smoke test (`frontend/e2e/mobile-smoke.spec.ts`, signed in, 360 px). It already visits `/division/:id`. Extend that visit to tap each of the four tabs and the navigator's ‹ / ›. Verify that it passes: no horizontal scroll, no text below 11 px, no tap target below 44 px, and that all four tab labels are fully visible (screenshot in the PR).

## 5. Specs and docs

- [x] 5.1 Write the `## Code map` in `openspec/changes/league-matchday-navigation/design.md` (the delta spec cannot carry it: archive drops sections outside the requirements), to copy into the main spec after archiving. Name `features/division/DivisionDetailsPage.tsx` (`matchdayPair`, `MatchdayNavigator`, `FixturesCard`, `hideStillLiveResults`), `features/leagues/RecentLeagueTransfers.tsx`, `api/hooks.ts` (`useDivisionDetails`, `useLeagueWeekMatches`), `handlers.DivisionDetailsHandler` / `handlers.LeagueWeekMatchesHandler`, `engine.GetDivisionDetails` / `engine.GetLeagueWeekMatches`, `common.GetDivisionDetails` / `common.WeekMatches`, and the `division.*` locale keys. Verify with `openspec validate league-matchday-navigation --strict`.

## 6. Verification

- [x] 6.1 Backend: `go build ./... && go vet ./... && go test ./...` in `backend/`.
- [x] 6.2 Frontend: `npx tsc --noEmit && npx oxlint && npx vitest run` in `frontend/`.

## Workflow follow-up

- Archive with `/opsx:archive league-matchday-navigation` after merge, which creates `openspec/specs/league-page/spec.md` with its Code map.
