# Tasks

## 1. Personal terms

- [x] 1.1 Add `common.PersonalTermsExpectation` with `TermsKind` and the floor constants (`RenewalWageFloorShare`, `LastSeasonWageFloorShare`, `TransferWageFloorShare`); verify "No free wage cut", "Last-season veteran", "Improving player" and "Free agent" with unit tests.
- [x] 1.2 Use it in `MakeContractRenewalOffer`, `SignFreeAgent` / `SignFreeAgentAtMarketRate` and `MakeTransferOffer`; verify "Pay cut refused" and "Improving target" with integration tests and keep the existing contract and offer tests green.

## 2. Valuations

- [x] 2.1 Add `contractValueFactor` to `MarketValue`; verify "Last season" and "Fresh contract" in `transferscoring_test.go` and update tests whose fixtures relied on the old value.
- [x] 2.2 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v`: the simulator models no transfers, so before and after differ only by its run-to-run noise (noted in the PR).

## 3. CPU

- [x] 3.1 Start `renewContract` and `makeAiOffer` from `PersonalTermsExpectation`; verify "Overpaid target" with a unit test.
- [x] 3.2 Add `pickUnrenewedToList` (action `listed_expiring`) to `manageRosterIfManager`; verify "Bench veteran with one season left", "Nobody buys him" and "Caretaker" with unit and integration tests.
- [x] 3.3 Add `contractsToRenewEarly` for Directors; verify "Rising star", "Bench player with two seasons" and "Manager" with unit tests.

## 4. Frontend

- [x] 4.1 Add a hint under the renewal and offer dialogs' wage field explaining that a player does not accept a pay cut (85% floor in his last season); add the en/es strings; verify with a vitest test.

## 5. Specs and checks

- [x] 5.1 Update the Code maps of `team-finances`, `cpu-transfer-market` and `cpu-team-management`; update `backend/README.md` "Player contracts & free agency"; verify `openspec validate rework-contract-terms --strict` passes.
- [x] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
