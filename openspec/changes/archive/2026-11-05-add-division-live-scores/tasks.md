# Tasks

## 1. Backend: live state and endpoint

- [x] 1.1 Add `SetValueFor(key, value string, ttl time.Duration)` to the `Cache` interface and `RedisCache` (`backend/common/cache.go`), and to the in-memory fakes in `handlers/teamLoansHandler_test.go`, `engine/scorer_simulation_test.go`, `engine/match_viewer_parity_integration_test.go` and `common/market_listings_integration_test.go`; verify with `go build ./... && go vet ./...`.
- [x] 1.2 Extract `liveStateAsOfNow` with `LiveState` and the `LiveStatusPending/Live/Halftime/Finished` constants in `backend/common/livematches.go`, keeping `liveScoreAsOfNow` as a wrapper; verify with unit tests in `common/livematches_test.go` (using `liveHubForTest`) for "Simulated ahead of kickoff", "Engine running late", "Half-time", "Broadcast finished" and "Broadcast state unreadable", and check that the existing `TestEnrichSuggestedLiveMatch_*` tests still pass.
- [x] 1.3 Add `getLeagueLiveMatchesStmtSql` (no goal columns selected), `liveEarlySimulationWindow`, `LeagueLiveScore` and `GetLeagueLiveScores(dbCtx, leagueId)` with the 5 s `league_live_scores:<id>` cache; verify with an integration test (`common/league_live_scores_integration_test.go`) covering "Matchday being broadcast", "No matchday in play", "Final result already stored" (stored 3-2, broadcast 1-2 → 1-2), the kickoff ordering, and a second call within 5 s served from the cache.
- [x] 1.4 Add `LeagueLiveScoresHandler` in `handlers/statsHandlers.go` and the route `GET /division/:id/live` in `service/fmservice.go`; verify "Invalid id" → 400 and an empty league → `[]` (not `null`) with a handler test.

## 2. Frontend: shared data and component

- [x] 2.1 Add the `LeagueLiveScore` type (`api/types.ts`), `queryKeys.divisionLiveScores` and `useDivisionLiveScores` with the 15 s / stop-when-finished `refetchInterval` (`api/hooks.ts`); verify with a hook test that polling stops once every entry is `finished` and with an empty list ("Matchday ends").
- [x] 2.2 Add `utils/liveScoreLabel.ts` ("63'", half-time, "FT"/"Final", kickoff time while pending) and `components/DivisionLiveScores.tsx` (rows with badges, goals, state chip and a link to `/match/:id`); verify with unit tests for each status, and a render test at 360 px with no horizontal overflow.
- [x] 2.3 Add the `liveScores.*` and `division.liveTable` / `division.positionUp` / `division.positionDown` strings to `i18n/locales/en.json` and `es.json`, with `_one`/`_other` plurals; verify "Spanish summary" ("1 partido · 0 goles") in a component test.

## 3. Dashboard: rest of the matchday

- [x] 3.1 Pass `divisionId` into `LiveMatchHero` and render the "Rest of the matchday" section (same `weekNumber`, own match excluded, collapsed on phones with the "N matches · G goals" toggle, expanded on desktop); verify in `features/dashboard/LiveMatchHero.test.tsx`: "League match live", "Score as broadcast", "Cup tie live" (no request made), "Endpoint down", "Phone summary" and "Desktop".
- [x] 3.2 Update `openspec/specs/team-dashboard/spec.md`'s Code map (`LiveMatchHero` section, `components/DivisionLiveScores.tsx`, `useDivisionLiveScores`, the `liveScores.*` strings); verify the listed paths exist.

## 4. League page: live scores in matchday cards

- [x] 4.1 Enable `useDivisionLiveScores` in `DivisionDetailsPage` only on the current season while `previousWeekMatches` has a played match inside its window, and pass a `Map<matchId, LeagueLiveScore>` into `FixturesCard`, which shows broadcast goals and the label chip for non-pending entries; verify "Match in play", "Broadcast over, window still open", "Simulated ahead of kickoff", "Quiet day", "Browsing an old matchday mid-broadcast" and the modified "Match still broadcasting" and "Page load while matchday 12 is broadcasting" in `DivisionDetailsPage.navigation.test.tsx`/`DivisionDetailsPage.test.ts`, and check that the existing "Page load mid-season" request assertion still holds.

## 5. League page: live table

- [x] 5.1 Extract `compareClassification` from `hideStillLiveResults` and add the pure `features/division/liveTable.ts` (`applyLiveScores` → positions, deltas, playing set); verify with unit tests for "Live win moves a team up", "Pending match adds nothing", a draw, a stray entry from another week (ignored) and seed tie-breaks matching `hideStillLiveResults`.
- [x] 5.2 Add the "Live" switch (on by default, current season only, shown only while an entry is `live`/`halftime`/`finished`) and the `live` prop on `ClassificationTable` (▲N/▼N markers, pulsing dot, zones by live position); verify "Matchday under way", "Switch off", "Nothing live", "Past season", "Position markers", "Into the relegation zone" and "Live table on a phone" (360 px, no horizontal page scroll) in `DivisionDetailsPage.test.ts`.
- [x] 5.3 Update `openspec/specs/league-page/spec.md`'s Code map (`liveTable.ts`, `compareClassification`, `FixturesCard` live scores, `useDivisionLiveScores`, the backend handler and `GetLeagueLiveScores`, the new strings); verify the listed paths exist.

## 6. Integration checks

- [x] 6.1 Run the app locally (`--svc --engine`), let a matchday kick off, and check on a phone viewport and on desktop that the dashboard hero, the matchday cards and the live table agree with each match's own live page minute by minute, and that no stored final score appears before full time.
  - Checked locally on a seeded world (Postgres + Redis, `--svc --engine`): the engine simulated matchday 1 for real and the broadcast was caught up to minute ~33. The endpoint returned the broadcast scores (1-0, 1-1, 0-0, 2-0, 0-0), not the stored finals (2-0, 2-4, 2-0, 3-0, 1-0). The dashboard hero, the matchday cards and the live table agreed (33' / 32' with the 15 s polling lag), the live table's points and goal difference used the broadcast score, and there was no horizontal scroll at 360 px or 1280 px. Half-time and full-time were covered by the unit and integration tests rather than waited out live.
- [x] 6.2 Run `openspec validate add-division-live-scores --strict` and verify it passes.
- [x] 6.3 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.

## Workflow follow-up

- When archiving, add a `## Code map` section to the new `openspec/specs/division-live-scores/spec.md` (`backend/common/livematches.go`: `GetLeagueLiveScores`, `liveStateAsOfNow`, `getLeagueLiveMatchesStmtSql`; `backend/common/cache.go`: `SetValueFor`; `backend/handlers/statsHandlers.go`: `LeagueLiveScoresHandler`; `frontend/src/api/hooks.ts`: `useDivisionLiveScores`; `frontend/src/components/DivisionLiveScores.tsx`; `frontend/src/utils/liveScoreLabel.ts`; tests).
- Archive the change after review.
