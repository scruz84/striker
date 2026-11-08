# Tasks

## 1. Rules

- [x] 1.1 Add the tier tables to `common/playerintelligence.go` and use them in `wageFloorShare`, `PreContractMinimumWage` / `PreContractKeepWage` / `PreContractCounterBar` and `ReleaseClauseDiscount` / `DiscountedWage`; verify "Shrewd last-season veteran", "Mastermind", "Matching a rival" and "Mastermind and a low clause" with unit tests and keep the existing tests green.
- [x] 1.2 Average `PlayerSkillTrend` over the last 5 runs; verify "One good match" and "Fewer runs" with a unit test of the averaging and an integration test.

## 2. Insight

- [x] 2.1 Add `resources/sql/init106_negotiation_insights.sql`, `RecordNegotiationInsight` in every negotiation function and the club-reset cleanup; verify "After a refused offer" with an integration test.
- [x] 2.2 Add `negotiationStyle` to the player details for the viewer's club; verify "Another club" with a handler test; show the "Negotiator" row on the player page with en/es strings and a vitest test.

## 3. Specs and checks

- [x] 3.1 Add the Code map to the new `player-negotiation` spec and update `team-finances`, `pre-contracts`, `release-clauses` and `backend/README.md`; verify `openspec validate add-negotiation-intelligence --strict` passes.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
