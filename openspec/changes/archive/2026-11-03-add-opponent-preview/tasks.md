# Tasks

## 1. Backend

- [x] 1.1 Add `common/opponentpreview.go` (form, likely XI from the latest finished league/cup match, in-form players); verify "Established opponent", "No history", "Opponent changed its lineup" and "Latest match still being replayed" with integration tests.
- [x] 1.2 Add the pure `engine.ProjectZoneDuel` reusing `InitialSimulation.Apply`; verify with a unit test that it returns the same minute-1 ratings as a real simulation of the same lineups ("Stronger midfield" share included).
- [x] 1.3 Add `GET /v1/teams/preview/:kind/:id` (403 for non-participants, 409 after the lock); verify "Non-participant" and the lock with handler tests.

## 2. Frontend

- [x] 2.1 Add `OpponentPreviewCard` with form, likely XI, in-form players and `ZoneComparisonRows`, plus the hook and types; add en/es strings; verify with vitest tests for the full and the "No history" states.
- [x] 2.2 Place it in `LineupEditorPage` (next unlocked fixture, collapsed on phones) and in the scheduled view of `MatchViewer` for participants; refetch after a lineup save; verify "Lineup editor", "Lineup locked", "Next fixture already locked", "Participant before kickoff" and "Lineup changed" with vitest tests.

## 3. Specs and checks

- [x] 3.1 Add the Code map to the new `opponent-preview` spec and update the Code map of `match-viewer`; verify `openspec validate add-opponent-preview --strict` passes.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
