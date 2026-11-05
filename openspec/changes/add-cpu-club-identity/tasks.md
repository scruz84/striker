# Tasks

## 1. Levels

- [x] 1.1 Add `RandomAiLevelFrom` and `StepAiLevelToward`; verify the three scenarios of "AI level after a division change" with unit tests.
- [x] 1.2 Call `redrawAiLevelAfterMove` for each rollover move; verify with an integration test that a promoted CPU club changes by at most one step and an owned club is untouched.

## 2. Style

- [x] 2.1 Add `resources/sql/init102_team_ai_style.sql`, `common/aistyle.go` (`AiClubStyleFor`, `AssignAiStyle`, `BackfillAiStyles`, `ApplyClubStyle`, `AiTransferProfileForTeam`), `Team.AiStyle` and `aiProfile(team)` in the engine; verify "Youth club", "Frugal club", "Ambitious club" and "Stable" with unit tests.

## 3. Prospects

- [x] 3.1 Add `pickProspectByPotential` / `promoteProspectByPotential`; verify "Gifted prospect", "Ordinary prospect" and "Balanced Manager" with unit tests.

## 4. Specs and checks

- [x] 4.1 Update the Code map of `cpu-team-management`; verify `openspec validate add-cpu-club-identity --strict` passes.
- [x] 4.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
