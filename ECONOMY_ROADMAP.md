# Team economy roadmap

This tracks a multi-phase plan for deepening Striker's team-economy mechanics (budget, wages,
transfers, contracts, income). Written so a fresh session can pick up any phase without needing
the conversation that produced it - each phase lists the mechanic, why it matters, and where to
start reading in the code.

## Status

- **Wage/cadence rescale bug: FIXED.** `PayMatchdayWages` started charging a team's full wage
  bill once per round instead of once a real calendar week, but `WagePerSkillPoint`/
  `TrainerWagePerSkillPoint` were never scaled down to match - under the default daily league
  schedule (`common.DefaultMatchScheduleWeekdaysMask` = every weekday) that left payroll up to 7x
  its intended size relative to per-match ticket/sponsorship income, steadily draining
  `teams.budget` across the whole game (visible as `striker.economy.total_budget` trending down -
  see `backend/common/telemetry.go`). Fixed by splitting both constants into a numerator/
  denominator pair (`common/fmcommon.go`, `common/trainers.go`) so the per-round rate is the old
  weekly rate ÷ 7, plus a migration (`resources/sql/init65_rescale_wages_to_match_cadence.sql`)
  recalculating every already-existing player's/trainer's stored wage the same way. See the
  "Wages" bullet in `GAME_MECHANICS.md`'s Economy section for the full explanation.
- **Phase 1 - Contracts & free agency: DONE.** See `backend/README.md`'s "Player contracts &
  free agency" section, `backend/common/contracts.go`, and commit `e902b50` on
  `claude/repo-overview-4di58f`. Players now have a contract length that expires to free agency;
  any team can sign a free agent for just their wage; owners can renew (free, capped) or release
  early (paid severance) their own players.
- **Phase 2 - Financial consequences (insolvency): DONE.** See the "Insolvency" paragraph in
  `GAME_MECHANICS.md`'s Economy section, `common.MaxDebt`/`common.IsTeamSolvent` in
  `common/fmconstants.go`/`common/database.go`, and PR #51. A team crossing `-MaxDebt` (-200,000)
  is blocked from `ExecuteTransfer`, `MakeTransferOffer`, `AcceptLoan`, and `SignFreeAgent` via
  `ErrTeamInsolvent`, exposed to the frontend as `insolvent` on the finances response. The
  "optional harsher tier" (forced sale / points penalty) below was deliberately skipped - the
  spend-side lockout alone was judged enough teeth for now.
- **Phase 3 - Richer income streams: DONE.** Season tickets and merchandising shipped
  (`engine/seasonincome.go`, PR mentioned in git history as "Add season tickets and merchandising
  income (Economy Phase 3)"). Sponsor tiers shipped too: `common.SponsorDeal`
  (`common/sponsordeal.go`, `teams.sponsor_deal`) lets each team pick between
  `SponsorDealSteady`/`SponsorDealBalanced`/`SponsorDealPerformance`, applied in
  `engine.sponsorshipAmount` (`engine/sponsorship.go`) and changeable any time via a
  `SponsorDealCard` on `FinancesPage.tsx`, the same way ticket price is. See the "Sponsorship"
  bullet in `GAME_MECHANICS.md`'s Economy section for the exact multipliers.
- **Phase 5 - Planning tools: DONE.** `FinancesPage.tsx` now has a cash-flow forecast card
  (`features/finances/CashFlowForecastCard.tsx`/`cashFlowForecast.ts`), an income-breakdown chart
  (`IncomeBreakdownChart.tsx`), and a budget-over-time chart (`BudgetOverTimeChart.tsx`), all pure
  frontend computation from the existing transactions list - plus a small backend addition
  (`FinancesResponse.maxDebt`) so the forecast doesn't have to hardcode the debt ceiling a second
  time.
- Phase 4 is proposed, not started.

## Existing economy building blocks (for context)

- **Budget/wages/ledger**: `backend/common/database.go` (`AdjustTeamBudget`, `GetTeamWageBill`),
  reason codes in `backend/common/fmconstants.go` (`TRANSACTION_*`), wage formula in
  `backend/common/fmcommon.go` (`PlayerWageFromSkills`).
- **Transfers**: `backend/common/transfers.go` (fixed-price listings) and
  `backend/common/offers.go` (negotiated offers) - both follow the same atomic tx pattern (lock
  rows `FOR UPDATE`, validate inside the lock, mutate, two ledger inserts, commit, then
  best-effort notify). `common.contracts.go`'s `SignFreeAgent`/`RenewPlayerContract`/
  `ReleasePlayer` mirror this pattern and are the most recent example to copy from.
- **Loans**: `backend/common/loans.go` - free, fixed duration, mutually exclusive with an active
  transfer listing.
- **Sponsorship**: `backend/engine/sponsorship.go` - per-match amount by league tier, shaped by
  each team's chosen `common.SponsorDeal` (`common/sponsordeal.go`).
- **Stadium/tickets**: `backend/common/stadium.go` + `backend/engine/stadium.go` - capacity,
  ticket price, attendance-driven matchday revenue.
- **Prize money**: `backend/engine/prizemoney.go` - league position + cup round payouts.
- **Transfer windows**: `backend/common/database.go`'s `IsTransferWindowOpen` - one global flag,
  admin-toggled.

## Phase 2 - Financial consequences (insolvency) - DONE, see Status above

The optional harsher tier (forced-sale flag after N weeks in debt, or a league-points penalty at
rollover) was not built - the spend-side lockout was judged sufficient. Revisit only if playtesting
shows the debt ceiling alone isn't enough friction.

## Phase 3 - Richer income streams - DONE, see Status above

## Phase 4 - Transfer economics depth

- **Amortized transfer fees**: instead of `ExecuteTransfer`/`AcceptOffer` debiting the full fee
  in one ledger line, spread it over the buyer's contract length (now that Phase 1 gives every
  player one). Needs a new table (e.g. `transfer_amortization`) tracking remaining
  fee-per-season, debited during `engine/database.go`'s `UpdateTeamFinancesAfterMatch` alongside
  wages.
- **Loan money**: `common/loans.go`'s `AcceptLoan` currently moves no money at all. Add an
  optional loan fee (paid once, like a mini transfer) and/or the parent club covering a
  percentage of wages during the loan (would need `UpdateTeamFinancesAfterMatch` to split the
  wage debit between `player_loans.from_team_id` and `to_team_id`). Also consider a buy-option
  clause recorded on `player_loans` that the borrower can exercise like a fixed-price
  `ExecuteTransfer`.
- **Auction/bidding**: `transfer_offers` already supports arbitrary counter-prices from any team
  on any player - a real auction would mean the seller can see and compare multiple simultaneous
  offers before accepting one, which the current `GetOffersReceived` listing already supports
  without schema changes; this is mostly a frontend grouping/comparison change in
  `TransferMarketPage.tsx`'s `OffersTab`.

## Phase 5 - Planning tools (frontend-heavy) - DONE, see Status above

## Suggested order

Phases 1, 2, 3, and 5 are done. What's left is Phase 4 (transfer economics depth) - its three
items are independent of each other, take in any order, or skip if the user prefers to prioritize
non-economy features instead.
