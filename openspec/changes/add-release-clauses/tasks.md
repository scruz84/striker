# Tasks

## 1. Storage and rules

- [x] 1.1 Add `resources/sql/init105_release_clauses.sql` (re-runnable) and set/clear the clause in every contract statement; verify "Renewal without a clause" and a clause cleared at contract expiry with integration tests.
- [x] 1.2 Add `common/releaseclauses.go` (`ClauseBaseValue`, `MinReleaseClauseMultiple`, `ReleaseClauseDiscount`) and the clause argument of `PersonalTermsExpectation`; verify "Clause too low", "Low clause" and "High clause" with unit tests.
- [x] 1.3 Accept `releaseClause` in renewal, free-agent signing, transfer offers and pre-contracts; verify with handler tests.

## 2. Paying

- [x] 2.1 Add `PayReleaseClause` and its route; verify "Clause paid" and "Too soon", the loan and pre-contract refusals, and a concurrent integration test where two clubs pay the same clause and exactly one succeeds.
- [x] 2.2 Add `releaseClausePaid` / `releaseClauseSigned` with Spanish templates and the watchlist fan-out; verify "Human loses a player".

## 3. CPU

- [x] 3.1 Add `aiRenewalClause` to `renewContract`; verify "Manager bench renewal" with unit (`TestRenewalClauseAmount`) and integration tests; "Director key player" follows from the code path (no clause when `common.IsKeyPlayer`) and has no dedicated test yet.
- [x] 3.2 Add `candidateReleaseClause` and `CpuMayPayClauseOn`; verify "Cheap clause" and "Rate limit" with integration tests ("Clause above its ceiling" reuses the listing ceiling in `scoreOne`, covered by the existing listing tests).
- [x] 3.3 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v`: the simulator models no transfers, renewals or clauses, so its figures only move by its run-to-run noise (noted in the PR).

## 4. Frontend

- [x] 4.1 Add the optional clause field (discount bands as a hint) to `ContractTermsFields` in the four contract dialogs, the clause on the player page and `PayReleaseClauseDialog`, with hooks and en/es strings; verify with vitest tests. Showing the clause on market rows is left for later.

## 5. Specs and checks

- [x] 5.1 Add the Code map to the new `release-clauses` spec and update the Code maps of `cpu-transfer-market` and `cpu-team-management` and `backend/README.md`; verify `openspec validate add-release-clauses --strict` passes.
- [ ] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
