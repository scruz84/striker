# Tasks

## 1. Projection

- [x] 1.1 Add `engine/expectedgoals.go` (`ProjectExpectedGoals`, `expectedConversion`, `shooterMix`); verify "Projection matches the engine" and the agreement with seeded simulations in `expectedgoals_test.go`.

## 2. Formations

- [x] 2.1 Make `formationRankForAiLevel` return 0 for a Manager; verify "Manager's shape" in `lineup_test.go`.
- [x] 2.2 Add `AiPlanOptions.Formation`, `engine/aiformation.go` (`chooseFormationAgainst`) and the Director second pass in `planAiKickoffs`; verify "Exploiting a weak flank" and "Friendly" with unit tests and the kickoff integration tests.
- [x] 2.3 Mirror the re-pick in the calibration and record it: Manager 1.500 vs 1.271, Director 1.734 vs 1.041 points per match (4,000 matches each, seed 11).

## 3. Specs and checks

- [x] 3.1 Update the Code map of `cpu-team-management`; verify `openspec validate improve-cpu-tactics --strict` passes.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
