# Tasks

## 1. Scoring loans (common)

- [x] 1.1 Add `SearchesLoanList` to `AiTransferProfile` (true for Manager and Director) in `common/aimanager.go`; verify with a case in `aimanager_test.go`.
- [x] 1.2 Extend `TargetContext` with `Loan`, `LoanSeasons`, `LoanWageContributionPercent`, `BuyOptionFee` and implement the loan branch of `ScoreTarget` (wage share in the payroll check, no projection/reputation, fair fee = value × 0.10 × seasons, +2 fair buy option) per design D2; verify with `transferscoring_test.go` cases for the spec scenarios "Fair loan fee", "Free loan", "Fair buy option" and "Lender pays half the wage".
- [x] 1.3 Add `AiMaxLoanedIn = 2`, `AiMaxOpenLoanListings = 2`, `AiMaxLoanedOut = 3` next to `AiMaxOpenOffers*`; add `LoanedOut` to `TeamMarketCommitments` (`GetTeamLoanedOutPlayerIdsStmtSql`) and a `GetLoansToTeam` helper returning return season and buy-option fee; verify with an integration test in `common/loans_test.go` or a new `*_integration_test.go`.
- [x] 1.4 Expose transfer-listing `listed_at` and asking price for a team's own listings (helper or commitments field) and verify with an integration test.

## 2. Borrowing and buy options in the transfer pass (engine)

- [x] 2.1 Add `candidateLoan` / `candidateBuyOption` kinds, `transferState.LoanedIn`, `playingSquad()`, drop loan-listed players from `available()`, and the cap/loan-limit checks in `pickNextTarget` (D3); verify with pure tests in `aitransfers_test.go` for "Loaned-in players fill the cap", "Two loanees already" and "Already loaned in at his position".
- [x] 2.2 Gather loan listings in `gatherCandidates` (skip own listings, age > 36, seen players) and buy-option candidates in the loan's last season only; make `scoreOne` build the loan / buy-option context; verify "Own loan listing", "Not the last season" and "Starter worth keeping" in `aitransfers_test.go`.
- [x] 2.3 Execute via `common.AcceptLoan` / `common.ExerciseLoanBuyOption` in `executeTransferMove`, silencing the expected errors and recording `loaned_in` / `exercised_buy_option`; keep `planSellToBuy` off loan candidates; verify with an engine integration test where a Manager takes a human's loan listing (human receives `loanedOut`) and a Director exercises an option.
- [x] 2.4 Integration test: two CPU teams run their passes concurrently against one loan listing and exactly one `player_loans` row results.

## 3. Lending in roster upkeep (engine)

- [x] 3.1 Implement the pure `pickPlayerToLoanOut` (eligibility: not listed/international/recent signing, contract >= duration + 1, depth and squad-size floors; surplus listing older than 7 days first, else Director development loanee aged <= 21 outside the best XI, youngest then lowest id) and `planLoanListingWithdrawals`; verify every scenario of "Loan-out eligibility", "Development loans", "Surplus loans" and "Withdrawing loan listings" in `aiteams_test.go`.
- [x] 3.2 Add `manageLoanListings` after `listSurplusPlayer` in `manageRosterIfManager` (limits 1 per tick / 2 open / 3 out; surplus loan = cancel transfer listing then `CreateLoanListing(1, round(MarketValue × 0.10), 0, oldAsking)`, re-list on failure; development loan = `CreateLoanListing(1, 0, 50, 0)`), recording `listed_for_loan` / `loan_listing_withdrawn`; verify with an engine integration test including "Window closed" (no loan listing while the window is closed).

## 4. Simulation, specs and docs

- [x] 4.1 Extend `engine/ai_transfers_simulation_integration_test.go` to count CPU loans in/out and buy options over the simulated days and assert the per-team limits hold; verify the test passes.
- [x] 4.2 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` before and after; record CPU budget medians in this change and confirm no division drifts towards insolvency.
- [x] 4.3 Update the Code map of `openspec/specs/cpu-transfer-market/spec.md` and `openspec/specs/cpu-team-management/spec.md` (new functions, constants and helpers); no en/es locale or `notificationtranslations.go` change is needed since no new strings or notification keys are added — verify with `grep` that none were introduced.
- [x] 4.4 Run `openspec validate cpu-loan-market --strict` and verify it passes.

## 5. Final checks

- [x] 5.1 Backend: `go build ./... && go vet ./... && go test ./...` passes.
- [x] 5.2 Frontend: `npx tsc --noEmit && npx oxlint && npx vitest run` passes.

## Workflow follow-up

- Resolve the proposal's open questions with the owner before starting group 3.
- Archive the change once merged (`/opsx:archive`).
