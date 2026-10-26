# Design

## Context

See proposal.md - Why. Relevant current state:

- `common.MinTicketPrice = 1`, `MaxTicketPrice = 100`, `DefaultTicketPrice = 10`
  (`common/fmconstants.go`); `common.SetTicketPrice` rejects anything outside that range with
  `ErrInvalidTicketPrice`, which `SetTicketPriceHandler` maps to HTTP 400.
- `engine.priceFactor` is 0 from `SociosPriceZeroAt = 40` upward, so prices 40–100 always give
  attendance 0. `creditTicketRevenue` returns early when revenue is 0, so no ledger row is written.
- `engine.optimalTicketPrice` scans `[MinTicketPrice, MaxTicketPrice]`; its optimum is always
  below 40 because revenue there is 0. `maximizeTicketPrice` (CPU Manager/Director) calls it daily.
- `teams.ticket_price integer NOT NULL DEFAULT 10` (`resources/sql/init.sql`); the last
  migration is `init86_simulation_failures.sql`. Migrations re-run at every startup.
- The finance page's price dialog (`StadiumCard` in `FinancesPage.tsx`) is a bare number field:
  no range, no hint. The frontend duplicates backend bounds as constants in `api/types.ts`
  (e.g. `MIN_STADIUM_UPGRADE_SEATS`).

## Goals / Non-Goals

**Goals:**
- No team can be on a price where nobody attends, now or after this change ships.
- An owner choosing a price sees what range is valid and what price the model rewards.

**Non-Goals:**
- Any change to the demand model, socios, or to how much a well-priced gate earns.
- Ledger rows for empty houses, or a notification for the reset.

## Decisions

1. **Bound the price at `SociosPriceZeroAt − 1` = 39.** `common.MaxTicketPrice` becomes 39. It
   stays a `common` constant (the engine imports common, not the other way round); a test in
   `engine/stadium_test.go` asserts `common.MaxTicketPrice == SociosPriceZeroAt-1` so the two
   cannot drift apart.
   *Alternative:* keep 100 and only warn in the UI. Rejected: the trap stays reachable by API and
   by any owner who ignores the warning, and the spec would keep a range where 60% of the values
   are meaningless.

2. **Migration `resources/sql/init87_cap_ticket_price.sql`:**
   `UPDATE teams SET ticket_price = 10 WHERE ticket_price > 39;` It is naturally re-runnable (a
   second run matches no rows) and touches CPU and human teams alike; CPU teams re-price to their
   optimum at their next daily tick anyway.
   *Alternative:* reset to each team's optimal price via a Go backfill at engine start. Rejected
   for now: needs a one-shot guard and an engine-replica-safe claim for a one-time job; 10 is the
   documented default and close to the optimum for most teams. Left as an open question for the
   owner (proposal.md).

3. **Recommended price computed server-side.** Add exported
   `engine.RecommendedTicketPrice(dbCtx, teamId) (int, error)`, which reads stadium, socios and
   recent form exactly as `maximizeTicketPrice` does and returns `optimalTicketPrice(...)`;
   `maximizeTicketPrice` is refactored to call it so the CPU and the hint can never disagree.
   `FinancesResponse` gains `recommendedTicketPrice int` and `maxTicketPrice int` (from
   `common.MaxTicketPrice`); `minTicketPrice` is not sent (it is 1 and fixed).
   *Alternative:* compute it in the frontend. Rejected: it would duplicate the demand model and
   need the form, which the finance response does not carry.

4. **Price dialog.** `StadiumCard` receives `recommendedTicketPrice` and `maxTicketPrice`; the
   field gets `error` + `helperText` ("Between 1 and 39 · recommended: 18") and Save is disabled
   for a non-integer or out-of-range value. A small "Use recommended" text button fills the field.
   New i18n keys under `finances.stadiumCard`: `priceHelper`, `priceRange`, `useRecommended`
   (en and es). No new notification keys, no new `TRANSACTION_*` reasons.

## Risks / Trade-offs

- [An owner deliberately priced at 40+ sees their price drop to 10 without notice] → They were
  earning nothing at that price; the recommended-price hint appears on the same page. A
  notification is listed as an open question.
- [API clients sending 40–100 now get 400] → The only client is our frontend, which now enforces
  the same bound.
- [Recommended price uses form and noise-free demand, so the real gate differs per match] → The
  hint is labelled as recommended, not guaranteed; it is the same figure the CPU uses.

## Migration Plan

Deploy backend and frontend together. On startup `init87_cap_ticket_price.sql` resets legacy
prices; from the next home match those teams earn a gate again. Rollback: revert the code; the
reset prices remain valid under the old bound, so no data rollback is needed.

## Open Questions

- Reset value (10 vs. per-team optimum) and whether to notify affected owners - see proposal.md.
  Either answer only changes task 2.1's SQL/backfill and adds an optional notification task.
