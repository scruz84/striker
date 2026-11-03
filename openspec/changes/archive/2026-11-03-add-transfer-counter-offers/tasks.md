# Tasks

## 1. Counter-offers (common and API)

- [x] 1.1 Add `resources/sql/init98_offer_counters.sql` (counter columns and partial index), re-runnable; verify by applying it twice on the test DB.
- [x] 1.2 Extract `executeOfferTx` from `AcceptOffer` and verify the existing `offers_test.go` and offer integration tests pass unchanged.
- [x] 1.3 Implement `CounterOffer`, `AcceptCounterOffer`, `DeclineCounterOffer` and the `ErrOfferCountered` guard in `AcceptOffer`; verify "Counter", "Counter not above the offer", "Second counter", "Counter accepted", "Buyer cannot afford it" and "Counter declined" with integration tests, plus a concurrent test where two accept calls on one counter move the money once.
- [x] 1.4 Add the three endpoints and the new `TransferOffer` fields; verify with handler tests (only the seller can counter, only the buyer can accept/decline).
- [x] 1.5 Implement `ExpireCounterOffers` in the offer-response job; verify "Ignored counter" and a concurrent test where two replicas expire the same counter and each club gets exactly one notification.
- [x] 1.6 Add `NOTIFICATION_COUNTER_OFFER` and the counter keys with Spanish templates in `notificationtranslations.go`; verify "CPU counters a human" with an integration test.

## 2. CPU behaviour (engine)

- [x] 2.1 Counter near-miss offers in `respondToReceivedOffers` (Manager/Director, ≥ 70%, threshold rounded up to 1,000, never for thin squads); verify "Near-miss offer to a Director", "Low offer", "Caretaker near miss" and "Thin position near miss" in `aiteams_test.go`.
- [x] 2.2 Implement `answerCountersAsBuyer` (≤ 1.5 × valuation and affordable, decline → rejection cooldown on the original bid); verify "Acceptable counter" and "Counter too high" with an engine integration test.
- [x] 2.3 Extend `ai_transfers_simulation_integration_test.go` to count counters and their outcomes and verify the offer limits still hold.

## 3. Frontend

- [x] 3.1 Add `CounterOfferDialog` (with `TransferComparables`) to received offers and Accept/Decline with the expiry time to countered sent offers in `TransferMarketPage`; add the hooks, types and en/es strings; verify "Buyer's view" and the seller's countered state with vitest tests.

## 4. Specs and checks

- [x] 4.1 Add the Code map to the new `transfer-negotiation` spec and update the Code maps of `cpu-team-management`, `cpu-transfer-market` and `team-finances` (transfers section); verify `openspec validate add-transfer-counter-offers --strict` passes.
- [x] 4.2 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` and verify transfer volumes and CPU budgets stay within their previous ranges.
- [x] 4.3 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
