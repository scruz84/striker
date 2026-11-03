# Tasks

## 1. Item rules

- [x] 1.1 Add `features/dashboard/pendingItems.ts` with `buildPendingItems` and the threshold constants; verify every "Pending items" and "Ordering" scenario with unit tests in `pendingItems.test.ts`.
- [x] 1.2 Add the snooze helpers (localStorage in try/catch, content key, local midnight); verify "Snoozed offers" and "Insolvency" with unit tests, including a throwing `localStorage`.

## 2. Panel

- [x] 2.1 Add `PendingPanel.tsx` and replace `AttentionAlerts` in `DashboardPage`, wiring the next fixture's lineup, trainer, forecast and ticket price; add en/es strings; verify "Nothing pending", "Injured starter for the next match" and "Injured reserve" with vitest tests.
- [x] 2.2 Add the phone layout (3 items + "Show all (N)"); verify "Five items on a phone" with a vitest test using the existing phone test helper.

## 3. Specs and checks

- [x] 3.1 Add the pending panel to the Code map of `team-dashboard`; verify `openspec validate improve-dashboard-pending-actions --strict` passes.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
