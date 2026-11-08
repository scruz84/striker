# Tasks

## 1. Storage and rules

- [ ] 1.1 Add `resources/sql/init105_release_clauses.sql` (re-runnable) and set/clear the clause in every contract statement; verify "Renewal without a clause" and a clause cleared at contract expiry with integration tests.
- [ ] 1.2 Add `common/releaseclauses.go` (`ClauseBaseValue`, `MinReleaseClauseMultiple`, `ReleaseClauseDiscount`) and the clause argument of `PersonalTermsExpectation`; verify "Clause too low", "Low clause" and "High clause" with unit tests.
- [ ] 1.3 Accept `releaseClause` in renewal, free-agent signing, transfer offers and pre-contracts; verify with handler tests.

## 2. Paying

- [ ] 2.1 Add `PayReleaseClause` and its route; verify "Clause paid" and "Too soon", the loan and pre-contract refusals, and a concurrent integration test where two clubs pay the same clause and exactly one succeeds.
- [ ] 2.2 Add `releaseClausePaid` / `releaseClauseSigned` with Spanish templates and the watchlist fan-out; verify "Human loses a player".

## 3. CPU

- [ ] 3.1 Add `aiRenewalClause` to `renewContract`; verify "Manager bench renewal" and "Director key player" with unit tests.
- [ ] 3.2 Add `candidateReleaseClause` and `LastCpuClausePaymentOn`; verify "Cheap clause", "Clause above its ceiling" and "Rate limit" with unit and integration tests.
- [ ] 3.3 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` and record the clause payments per season and the wage-bill change in the PR.

## 4. Frontend

- [ ] 4.1 Add the clause field with its live discount to `ContractTermsFields`, the clause on the player page and market rows, and `PayReleaseClauseDialog`, with hooks and en/es strings; verify with vitest tests.

## 5. Specs and checks

- [ ] 5.1 Add the Code map to the new `release-clauses` spec and update the Code maps of `cpu-transfer-market` and `cpu-team-management` and `backend/README.md`; verify `openspec validate add-release-clauses --strict` passes.
- [ ] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
