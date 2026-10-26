# Tasks

## 1. Price bound

- [ ] 1.1 Set `common.MaxTicketPrice = 39` in `backend/common/fmconstants.go` and update its comment (39 = `engine.SociosPriceZeroAt − 1`); verify `common/stadium_test.go` rejects 40 and accepts 39 with `ErrInvalidTicketPrice` / nil.
- [ ] 1.2 Add `TestMaxTicketPriceBelowZeroDemand` in `backend/engine/stadium_test.go` asserting `common.MaxTicketPrice == SociosPriceZeroAt-1`, and change `TestCalculateAttendance_NobodyPaysAbsurdPrices` to iterate `SociosPriceZeroAt, 70, 100` (literal, no longer `MaxTicketPrice`); verify `go test ./engine -run 'TicketPrice|Attendance'` passes.
- [ ] 1.3 Update `SetTicketPriceHandler`'s and `optimalTicketPrice`'s doc comments for the new range; verify the existing optimal-price tests still pass.

## 2. Legacy prices

- [ ] 2.1 Add `backend/resources/sql/init87_cap_ticket_price.sql` with a header comment and `UPDATE teams SET ticket_price = 10 WHERE ticket_price > 39;`; verify with an integration test that seeds a team at 69 and one at 25, applies migrations twice, and finds 10 and 25.

## 3. Recommended price

- [ ] 3.1 Add exported `engine.RecommendedTicketPrice(dbCtx, teamId)` in `engine/stadium.go` (stadium, socios, recent form → `optimalTicketPrice`) and make `maximizeTicketPrice` call it; verify the AI ticket-pricing tests still pass and a unit/integration test shows 1,000 seats / 3,000 socios recommends > 10 and 30,000 seats / 500 socios recommends ≤ 25.
- [ ] 3.2 Add `recommendedTicketPrice` and `maxTicketPrice` to `FinancesResponse` in `handlers/financeHandlers.go`, filled from 3.1 and `common.MaxTicketPrice`; verify with a handler test (or the existing finances handler test) that both fields are present.

## 4. Frontend price dialog

- [ ] 4.1 Add `recommendedTicketPrice: number` and `maxTicketPrice: number` to `FinancesResponse` in `frontend/src/api/types.ts`; verify `npx tsc --noEmit` passes.
- [ ] 4.2 Pass both to `StadiumCard` in `FinancesPage.tsx`; give the price field `error`/`helperText` (range + recommended), a "Use recommended" button, and disable Save for a non-integer or out-of-range price; verify with a Vitest component test (typing 50 disables Save and shows the range; "Use recommended" fills the field).
- [ ] 4.3 Add `finances.stadiumCard.priceHelper`, `priceRange` and `useRecommended` to `frontend/src/i18n/locales/en.json` and `es.json`; verify the i18n key-parity test passes.

## 5. Specs and docs

- [ ] 5.1 Update the Code map of `openspec/specs/team-finances/spec.md` (`engine.RecommendedTicketPrice`, `init87_cap_ticket_price.sql`, price dialog in `FinancesPage.tsx`) and `openspec/specs/socios/spec.md` (`RecommendedTicketPrice`); verify `openspec validate fix-empty-gate-high-ticket-price --strict` passes.

## 6. Final checks

- [ ] 6.1 Economy: `cd backend && ECON_SIM=1 go test ./engine -run TestEconomySimulation -v`; verify the passive/optimal human and CPU results match the pre-change run (the optimum never exceeded 39).
- [ ] 6.2 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`
- [ ] 6.3 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`

## Workflow follow-up

- Archive the change after review, syncing the delta specs into `team-finances` and `socios`.
