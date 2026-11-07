# Tasks

## 1. Schema

- [x] 1.1 Add `backend/resources/sql/init103_international_cup_entries.sql` (table, index, backfill from existing regional cups' round-1 entrants); verify "Existing editions on upgrade" with an integration test that runs the backfill twice.

## 2. Backend

- [x] 2.1 Add `common.GetRegionalCupCountryStates`, `common.HasActiveRegionalCup` and the transactional `common.CreateRegionalCup` in `backend/common/cups.go`; remove `GetMostRecentTier1Champions`.
- [x] 2.2 Add the pure `engine.regionalCupEntrants` with `MinRegionalCupCountries` and `RegionalCupMaxUnusedChampions`; verify "Every country has a new champion", "Waiting for the last country", "A slow country does not hold the region back", "One country only" and "Champion already played" with unit tests.
- [x] 2.3 Add `engine.StartRegionalCupsIfDue` under `LockNamespaceRegionalCupStart`, `RegionalCupJob` and its 5-minute scheduler; verify "Edition still running", "Two titles since the last edition", "Third edition" and the season_history-not-live-standings pick with an integration test, and "Three replicas tick together" with a concurrent integration test.
- [x] 2.4 Turn `StartInternationalCupsIfDue` into `StartWorldCupIfDue` (World Cup only), remove `StartInternationalCupsForNewSeason`; keep the existing claim test passing.

## 3. Frontend

- [x] 3.1 Title the international cup page with `cup.titleWithEdition`; update `cup.entryCriteria`, `cup.noInternationalCup` and the cup help answer in en/es.

## 4. Specs and checks

- [x] 4.1 Update the Code map of `world-cup`, `engine-replicas` and `match-scheduling`, and add `international-cup` on archive.
- [x] 4.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.
