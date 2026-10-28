# Proposal

## Why

An owner reported that home matches no longer earn any ticket revenue: the finance ledger has no
"Venta de entradas" rows at all, even on a day they played at home. The gate code itself is
intact (`creditTicketRevenue` runs for every league match and cup tie), but since
`rebalance-economy-socios` (2026-10-19) the crowd is `socios × priceFactor × …` and
`priceFactor` is 0 at a price of 40 or more. Owners can still set any price from 1 to 100, and
under the old model the "best" price was 69, so a human who raised their price before the
rebalance (or does so now) silently gets an empty stadium: attendance 0, revenue 0, and because
`creditTicketRevenue` skips a zero amount, no ledger row explains why. CPU teams are unaffected:
they re-price daily with `optimalTicketPrice`. Nothing in the finance page warns the owner.

## What Changes

- **BREAKING (API bound):** the ticket price range becomes 1–39 (`common.MaxTicketPrice` 100 →
  39, one below `SociosPriceZeroAt`). `PUT /team/stadium/ticket-price` with 40 or more is rejected with
  `ErrInvalidTicketPrice` (HTTP 400), as any out-of-range price already is.
- A one-off migration resets every stored ticket price above 39 to the default 10, so teams stuck
  on a pre-rebalance price earn a gate again from their next home match.
- The finance response gains `recommendedTicketPrice` (the same `optimalTicketPrice` the CPU uses,
  for this team's capacity, socios and form) and `maxTicketPrice`. The price dialog shows the
  allowed range and the recommended price, and blocks a price outside the range client-side.
- Specs updated: the gate's price range, the optimal-price scan range and the attendance-factor
  scenario at the top of the range.

Out of scope:
- Recording a 0-amount ledger row for an empty house, or explaining attendance in the ledger.
- Changing the demand model, `SociosPriceZeroAt` or the optimal-price formula.
- Notifying owners whose price was reset (see open question).
- The ledger in the report also shows no wages or maintenance on 25/10 and 26/10, which only
  happens when those days' matches were cup ties (wages are charged on league matches only). That
  is existing, specified behavior and is not changed here.

Open questions for the owner:
- Reset value for legacy prices: the default 10 (proposed, simple and safe) or the team's
  recommended price (needs a Go backfill instead of a SQL migration)?
- Should affected owners get a notification that their price was reset? (Needs a new key and its
  Spanish template; not included.)

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `team-finances`: "Matchday gate" narrows the ticket price range to 1–39, resets legacy prices
  above it, and the finance page shows the recommended price.
- `socios`: "Attendance factors" scenario at the top of the price range and "Optimal ticket
  price" scan range follow the new bound.

## Impact

- Backend: `common/fmconstants.go` (`MaxTicketPrice`), `common/stadium.go` (`SetTicketPrice`
  validation unchanged, bound changes), `engine/stadium.go` (`optimalTicketPrice` scan, export a
  helper for the handler), `handlers/financeHandlers.go` (`FinancesResponse`), new
  `resources/sql/init87_cap_ticket_price.sql`, tests in `engine/stadium_test.go`,
  `common/stadium_test.go`.
- Frontend: `api/types.ts` (`FinancesResponse`), `features/finances/FinancesPage.tsx` (price dialog),
  `i18n/locales/{en,es}.json`.
- Economy simulation: unchanged in outcome (the optimal price is always below 40); re-run
  `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` to confirm.
