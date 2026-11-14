# Tasks

## 1. Backend

- [x] 1.1 Add `resources/sql/init112_international_cup_runners_up.sql` and guard the `init103` backfill (design Decision 4); integration test that runs both files twice on an upgraded table.
- [x] 1.2 `common/regionalcups.go`: `RunnerUpTeamId`, `RegionalCupEntrant`, `CreateRegionalCup` with positions, `InternationalCupEntry.FinalPosition`.
- [x] 1.3 `engine/regionalcupdraw.go` (`regionalCupFirstRound`) with unit tests for 2, 3 and 4 countries, a missing runner-up, no same-country round-1 tie and byes spread through the bracket.
- [x] 1.4 `engine/cups.go`: `RegionalCupEntrantsPerCountry`, `regionalCupTeams`, use the new draw in `startRegionalCupIfDue`.
- [x] 1.5 Update `cups_integration_test.go` (runner-ups in the fixture, entries per team) and `engine_replicas_integration_test.go` (4 entries, 2 ties).

## 2. Frontend

- [x] 2.1 `cup.noInternationalCup` and the international cup FAQ answer in `en.json` and `es.json`; `InternationalCupPage.tsx` doc comment.

## 3. Specs and checks

- [x] 3.1 Update the `international-cup` spec Purpose and Code map.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.
