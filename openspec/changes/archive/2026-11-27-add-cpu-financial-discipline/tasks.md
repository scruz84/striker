# Tasks

## 1. Crisis

- [x] 1.1 Add `engine/aifinances.go` (`inFinancialCrisis`, `pickCrisisSale`, `listForFundsIfInCrisis`) and skip `runAiTransferPass` in crisis; verify "In debt", "Two listings already", "Back in the black" and "Caretaker in debt" with unit and integration tests.
- [x] 1.2 Add `OfferContext.Distressed` and `CrisisOfferShare`; verify "Cheaper sale in debt" and "Not in debt" in `transferscoring_test.go` and the counter computed from the discounted threshold.

## 2. Renewals and releases

- [x] 2.1 Make `contractsToRenew` selective and cap the retry raise; verify "Bench veteran", "Veteran keeping the depth", "Greedy bench player" with unit tests and keep the existing renewal integration tests green.
- [x] 2.2 Add `common.GetTeamLoanListingTimes`, `pickVeteranToRelease` and `releaseUnsoldVeteranIfManager`; verify "Nobody wanted him", "Squad at its size" and "Listed for 10 days" with unit tests and one integration test.

## 3. Specs and checks

- [x] 3.1 Update the Code maps of `cpu-team-management` and `cpu-transfer-market`; verify `openspec validate add-cpu-financial-discipline --strict` passes.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
