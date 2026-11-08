# Tasks

## 1. Backend

- [x] 1.1 Add `common.GetPreContractMarket` and `GET /v1/pre-contracts/market`; verify "Own players" and the redacted, public fields with `TestPreContractMarketHandler`.

## 2. Frontend

- [x] 2.1 Add the Last year tab (`LastSeasonTab`, `usePreContractMarket`) and `ContractMeta` on `ListingCard`, with en/es strings; verify "Not open yet", "Already offered" and "Listed player with a clause" with `LastSeasonTab.test.tsx`.

## 3. Specs and checks

- [x] 3.1 Update the Code maps of `pre-contracts` and `release-clauses`; verify `openspec validate add-pre-contract-market --strict` passes.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
