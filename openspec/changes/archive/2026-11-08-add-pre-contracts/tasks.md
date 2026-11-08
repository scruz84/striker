# Tasks

## 1. Storage and core

- [x] 1.1 Add `resources/sql/init104_pre_contracts.sql` (re-runnable) and `common/precontracts.go` with the constants (`PreContractSeasonShare`, `PreContractMovePremium`, `PreContractDecisionDelay`, `MaxSignedPreContracts`, `AiMaxSignedPreContracts`) and `MakePreContractOffer` / `WithdrawPreContractOffer`; verify "Too early", "Offer made" and "Below the premium" with integration tests.
- [x] 1.2 Add `DecidePreContracts` and call it from `OfferResponseJob` after `RespondToAllPendingOffers`; verify "Two suitors" and a concurrent integration test where two replicas decide the same player and exactly one pre-contract is signed.
- [x] 1.2b Add `PreContractCounterBar` to `MakeContractRenewalOffer` and the `preContractDecisionSoon` reminder in `DecidePreContracts`; verify "Matching a rival", "Better offer during the wait", "No counter" and "Reminder" with integration tests, plus a concurrent test where two replicas send the reminder only once.
- [x] 1.3 Cancel pending offers in `MakeContractRenewalOffer`, `executeOfferTx`, `ExecuteTransfer` and `AcceptLoan`; refuse a pre-contracted player in renewal, release, listings and offers (`ErrPlayerPreContracted`); verify "Renewed in time" and "Listing refused".
- [x] 1.4 Make `AdvancePlayerContract` complete signed pre-contracts (kind `PLAYER_TRANSFER_PRE_CONTRACT`) and `RetirePlayer` drop them; verify "Rollover" and "Retirement" with integration tests.

## 2. Notifications and API

- [x] 2.1 Add `NOTIFICATION_PRE_CONTRACT` and the eight keys with Spanish templates; verify "Signed away" and the cancellation notice with integration tests.
- [x] 2.2 Add `Player.ContractExpiring` (never redacted) and the three routes with their handlers; verify "Rival's player" and "Two seasons left" and the handler errors with handler tests.

## 3. CPU

- [x] 3.1 Add `GetExpiringCandidatesForPosition` and `candidatePreContract` in `runAiTransferPass`; verify "Director plans next season", "Two signed already" and "Pile-up on a human's player" with unit and integration tests.
- [x] 3.2 Add `answerPreContractOffers`; verify "Manager keeps its starter", "Rival outbids the ceiling", "Director key player", "Director lets a bench veteran go" and "Caretaker bench player".
- [x] 3.3 Extend `ai_transfers_simulation_integration_test.go` so a simulated season shows CPU pre-contracts being offered, signed and completed.

## 4. Frontend

- [x] 4.1 Add the "Last season" chip, `PreContractDialog`, the Pre-contracts section of the Offers tab (received offers show the decision countdown, the wage that keeps the player and a "Renew" action opening the renewal dialog pre-filled with it) and the "Signed for" squad chip, with hooks in `api/hooks.ts` and en/es strings; verify with vitest tests.

## 5. Specs and checks

- [x] 5.1 Add the Code map to the new `pre-contracts` spec, update the Code maps of `cpu-transfer-market` and `cpu-team-management` and `backend/README.md` "Player contracts & free agency"; verify `openspec validate add-pre-contracts --strict` passes.
- [x] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
