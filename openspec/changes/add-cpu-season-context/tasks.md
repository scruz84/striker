# Tasks

## 1. Objective

- [x] 1.1 Add `common/seasonobjective.go` (`SeasonSituation`, `SeasonObjectiveFor`, constants); verify "Early season", "Just above the drop", "Chasing promotion", "Dead rubber" and "Top division" with unit tests.
- [x] 1.2 Add `GetTeamSeasonSituation` (`CountRemainingLeagueMatchesStmtSql`); verify it with an integration test over a small league.

## 2. Effects

- [x] 2.1 Add `AdjustProfileForObjective` and pass the objective into `runAiTransferPass`; verify "Manager fighting relegation", "Director in a promotion push" and "Manager in a promotion push" with unit tests.
- [x] 2.2 Add `AiPlanOptions` / `giveYouthStarts` and set it in `planAiKickoff`; verify "Two prospects get a start", "Prospect too raw" and "Still something to play for" with unit tests, and that the plan still validates.

## 3. Specs and checks

- [x] 3.1 Update the Code maps of `cpu-team-management` and `cpu-transfer-market`; verify `openspec validate add-cpu-season-context --strict` passes.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
