# Tasks

## 1. Transfer snapshot

- [ ] 1.1 Add `backend/resources/sql/init87_player_transfer_snapshot.sql`: nullable `position`, `age`, `average_skill` smallint columns on `player_transfers` (`ADD COLUMN IF NOT EXISTS`) and the partial index `player_transfers_comparables_idx` from design §1, with a header comment explaining why old rows stay NULL; verify it applies twice in a row on a scratch database without error
- [ ] 1.2 Make `common.RecordPlayerTransfer` read the player's seven ratings and age through its `sqlExecutor` and insert `BestCorePosition`/`AverageSkill`/age with the row (signature unchanged); verify with an integration test that a bought listing, an accepted offer and a release each store the snapshot taken before any later skill change

## 2. Comparables query

- [ ] 2.1 Add `TransferComparable`, `TransferComparablesResult`, the band/window/limit constants and `const getTransferComparablesStmtSql` plus `GetTransferComparables` in `common/playertransfers.go` (design §3-4); verify `go vet ./...`
- [ ] 2.2 Add a unit test for the summary (median of 4 fees = 135,000 from the spec scenario, odd count, empty → nil fields); verify `go test ./common -run Comparables`
- [ ] 2.3 Add `common/transfer_comparables_integration_test.go` covering every spec scenario: eligible kinds only (free-agent premium, loan, release excluded), fee 0 excluded, 61-day-old excluded, target's own transfer excluded, other position excluded, NULL snapshot excluded, narrow band, broadening below 3 results with `Broadened` set, newest-first with id tiebreak and cap at 8; verify the test passes against the integration database
- [ ] 2.4 Assert in a test that the JSON of a `TransferComparablesResult` contains no rating, trait, stamina, form, wage or contract key; verify it passes

## 3. Endpoint

- [ ] 3.1 Add `handlers.TransferComparablesHandler` (400 on bad id, 404 on unknown player, 500 logged on query error) and register `GET /player/:id/transfer-comparables` in `v1TeamGroup` in `service/fmservice.go`; verify with a handler test for 200/400/404
- [ ] 3.2 Check the route appears correctly in the observability route templates (one series per route, see `service/fmservice.go` comment near line 101); verify by running the service tests

## 4. Offer dialog

- [ ] 4.1 Add `TransferComparable`/`TransferComparablesResult` types to `frontend/src/api/types.ts` and `useTransferComparables(playerId, enabled)` (query key `['transferComparables', id]`, 60 s stale time) to `api/hooks.ts`; verify `npx tsc --noEmit`
- [ ] 4.2 Build `frontend/src/components/TransferComparables.tsx` (loading, error, empty, broadened note, summary line, rows with name link, position, age, average, fee, from → to, date); verify with `TransferComparables.test.tsx` rendering each state
- [ ] 4.3 Render it in `TransferOfferDialog.tsx` under the fee field, query enabled only while `open`, hidden once `accepted`; verify by opening the dialog from the market, the squad list and the player profile in the running app, including at 360 px width
- [ ] 4.4 Add every new string under `transfers.offerDialog.comparables.*` in `frontend/src/i18n/locales/en.json` and `es.json` (title, summary, empty "No similar players have been transferred in the last 60 days", broadened note, error, column labels); verify no missing-key warnings in the browser console in both languages

## 5. Specs and docs

- [ ] 5.1 Update the "Code map (for the main spec)" section in this change's `design.md` to the final file and function names; mention the snapshot columns in the `init78`/`player_transfers` description in `ARCHITECTURE.md` if that file documents the table; verify by reading both
- [ ] 5.2 Run `openspec validate add-transfer-comparables --strict`; verify it passes

## 6. Final checks

- [ ] 6.1 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`; verify all pass
- [ ] 6.2 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass

## Workflow follow-up

- Resolve the proposal's two open questions with the owner before or during review.
- Archive the change (`/opsx:archive`) once merged, which creates `openspec/specs/transfer-comparables/spec.md`, then append the Code map from `design.md` to it.
