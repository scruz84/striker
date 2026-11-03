# Tasks

## 1. Storage, validation and API

- [x] 1.1 Add `resources/sql/init96_lineup_orders.sql` (four orders tables, `mentality` column on the three event tables), re-runnable; verify by applying it twice on the test DB.
- [x] 1.2 Add `common.LineupOrder`, `Lineup.Orders`, the `ORDER_*` constants and read/write orders with each lineup kind's substitutions and in the kickoff snapshot; verify with integration tests that the default, match, cup-tie and friendly lineups round-trip their orders.
- [x] 1.3 Add `ValidateOrders` and call it from every lineup handler; verify "Valid orders", "Too many orders" and "Invalid substitution order" with handler tests.

## 2. Firing (engine)

- [x] 2.1 Implement `applyConditionalOrders` after `applyScheduledSubstitutions` (conditions from the side's point of view, from-minute, once, slot order) and the `MENTALITY_CHANGE` event with its persisted mentality; verify "Going behind late", "Condition met before the from-minute", "Condition never met", "Two orders on the same minute", "Switch to Defensive" and "Already Attacking" with unit tests in `substitution_simulation_test.go`.
- [x] 2.2 Reuse the scheduled-substitution helper for substitution orders; verify "Cap already reached" and "Outgoing player sent off", and that injury replacements and orders share the 3-substitution cap.
- [x] 2.3 Verify a fixed-seed match without orders produces the same events as before (golden test).
  - The order stage draws no random numbers, so it can't shift the seeded stream; `TestConditionalOrders_NoOrdersPlaysAsBefore` checks that, without orders, the substitution stage gives exactly the pitch, counter and events of the scheduled substitutions alone, minute by minute. (Team mentality's `TestBalancedSidesPlayExactlyAsBefore` covers the rest of the pipeline.)

## 3. CPU orders

- [x] 3.1 Return Manager/Director orders from `PlanAiMatchLineup` (none in friendlies or for a Caretaker) and validate them in `planAiKickoffs`; verify "Director protecting a lead" and "Manager chasing the game" with tests in `ailineup_test.go` and an engine test.

## 4. Frontend

- [x] 4.1 Add `OrdersEditor` to `MatchLineupForm` wherever `substitutionsEnabled` is set, with the `orders` field in the lineup types and en/es strings; verify with vitest tests that an order is added, validated (from-minute 1–89, at most 3) and sent on save.
- [x] 4.2 Render `MENTALITY_CHANGE` in `utils/matchCommentary.tsx` and `MatchEventsReport`, live only once the replay reaches its minute; verify "Live replay" and "Before the minute" with vitest tests.

## 5. Specs and checks

- [x] 5.1 Update the Code maps of `match-simulation`, `cpu-team-management` and `match-viewer`; verify `openspec validate add-conditional-match-orders --strict` passes.
- [x] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
