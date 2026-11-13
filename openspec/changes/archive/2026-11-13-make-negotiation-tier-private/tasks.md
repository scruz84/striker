# Tasks

## 1. Backend

- [x] 1.1 Remove `RecordNegotiationInsight`, `NegotiationStyleFor` and their SQL, `noteNegotiation` and its calls, `PlayerDetails.NegotiationStyle` and the reset's clear. Remove `handlers/negotiationInsight_test.go`.
- [x] 1.2 Replace `init106_negotiation_insights.sql` with `init109_drop_negotiation_insights.sql` (`DROP TABLE IF EXISTS`).

## 2. Frontend

- [x] 2.1 Remove the "Negotiator" row, `negotiationStyle`, `playerPage.stats.negotiator`, `playerPage.negotiationStyles.*` and its test; reword the FAQ answer in en and es.

## 3. Specs and checks

- [x] 3.1 Update the player-negotiation Code map and `backend/README.md`.
- [x] 3.2 Run backend `go build ./... && go vet ./...` and the unit tests; frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.
