# Tasks

## 1. Backend

- [x] 1.1 Add `resources/sql/init111_team_cup_titles.sql` (design Decision 3) and an integration test that seeds a wrongly recorded edition win, runs the file twice and checks the result.
- [x] 1.2 `common/seasonhistory.go`: `RecordSeasonHistoryInternationalCup`, and `LeagueTier` / `InternationalCup` in `SeasonHistoryEntry` and `GetSeasonHistory`.
- [x] 1.3 `engine/cups.go`: `advanceCup` takes the `common.Cup` and records each kind of cup as in design Decision 2; update `cup_advance_lock_integration_test.go`.
- [x] 1.4 Integration test: a regional champion is recorded in its current league season, not in the edition's number, and the World Cup knockout writes no history.

## 2. Frontend

- [x] 2.1 `api/types.ts`: `leagueTier`, `internationalCup` on `SeasonHistoryEntry`.
- [x] 2.2 `TrophyCabinet`: translated domestic chip, international chip with link, `HonoursSummary`; tests in `TrophyCabinet.test.tsx`.
- [x] 2.3 `InternationalCupPage`: `region` and `cup` URL parameters; test in `InternationalCupPage.test.tsx`.
- [x] 2.4 Locale keys in `en.json` and `es.json`.

## 3. Specs and checks

- [x] 3.1 New `team-history` spec with its Code map, and the `international-cup` Code map.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.
