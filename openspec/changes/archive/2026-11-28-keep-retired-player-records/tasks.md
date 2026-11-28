# Tasks

## 1. Schema

- [x] 1.1 Add `backend/resources/sql/init122_retired_players.sql` (`retired_players`, drop the three player foreign keys); verify it re-runs cleanly on a database that already has it.

## 2. Backend

- [x] 2.1 Write the snapshot in `RetirePlayer` and `RetireFreeAgentAsTrainer` (`recordRetiredPlayerTx`); verify "Rostered veteran", "Free agent into coaching" and "Golden Boot winner retires" with `TestRetirePlayer_KeepsHisRecord` and `TestRetireFreeAgentAsTrainer_KeepsHisRecord`.
- [x] 2.2 Answer a retired id from `PlayerDetailsHandler` (`retiredPlayerDetails`) and keep 404 for an unknown one.
- [x] 2.3 Keep the market panels on players in the game (`EXISTS` in the top fee, recent league transfers and comparables statements); verify the comparables tests and `TestTransferComparables_UsesIndex`.
- [x] 2.4 Clear the kept tables in `teamPlayerDependentTables`.

## 3. Frontend

- [x] 3.1 Add `RetiredPlayerDetails`, `isRetiredProfile`, `usePlayerProfile` and `RetiredPlayerView`, with en/es strings; verify "Open a retired player" with a vitest test.

## 4. Specs and checks

- [x] 4.1 Copy the design's Code map into `openspec/specs/retired-players/spec.md` after archiving.
- [x] 4.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.
