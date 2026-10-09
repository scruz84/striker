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
- **Fanbase: DONE.** Every team now has a 0-99 Fanbase index (`teams.fanbase`) that drifts slowly
  from league/cup results and feeds a multiplier into matchday attendance, sponsorship, and season
  tickets/merchandising - closing the gap `engine/stadium.go`'s `demandRate` doc comment used to
  name explicitly ("no model for what else should drive attendance"). See `GAME_MECHANICS.md`'s
  "Fanbase" section and `backend/engine/fanbase.go`. A match-simulation effect (home advantage/
  crowd noise) was deliberately left out - the simulator has no such concept today, and adding one
  is a separate, bigger design decision.
- **Phase 4, "loan money": DONE.** `AcceptLoan` used to move no money at all. A lender can now
  attach a one-time loan fee, a wage-contribution split (the lender keeps covering a percentage of
  the player's wage for the loan's duration), and/or a buy-option fee (the borrower can make the
  move permanent at any time) to a loan listing - all independent and optional, zero/off by
  default. See `GAME_MECHANICS.md`'s "Loan money" bullet, `backend/common/loans.go`, and
  `backend/engine/wages.go`'s `deductWages`.
- **Phase 4, "auction/bidding": DONE.** `GetOffersReceived` already returned every pending offer
  on a player from every interested buyer; the Offers tab now groups a player's competing offers
  together (sorted by amount) instead of interleaving them with every other player's by date, so
  a seller can actually compare before accepting one. See `TransferMarketPage.tsx`'s
  `ReceivedOffersByPlayer` - no backend/schema change was needed. "Amortized transfer fees" (the
  remaining Phase 4 item) is still proposed, not started.

## Existing economy building blocks (for context)

- **Budget/wages/ledger**: `backend/common/database.go` (`AdjustTeamBudget`, `GetTeamWageBill`),
  reason codes in `backend/common/fmconstants.go` (`TRANSACTION_*`), wage formula in
  `backend/common/fmcommon.go` (`PlayerWageFromSkills`).
- **Transfers**: `backend/common/transfers.go` (fixed-price listings) and
  `backend/common/offers.go` (negotiated offers) - both follow the same atomic tx pattern (lock
  rows `FOR UPDATE`, validate inside the lock, mutate, two ledger inserts, commit, then
  best-effort notify). `common.contracts.go`'s `SignFreeAgent`/`RenewPlayerContract`/
  `ReleasePlayer` mirror this pattern and are the most recent example to copy from.
- **Loans**: `backend/common/loans.go` - fixed duration, mutually exclusive with an active
  transfer listing, with an optional one-time fee, wage-contribution split, and buy-option fee
  (see "Phase 4, loan money" in Status above).
- **Sponsorship**: `backend/engine/sponsorship.go` - per-match amount by league tier, shaped by
  each team's chosen `common.SponsorDeal` (`common/sponsordeal.go`).
- **Stadium/tickets**: `backend/common/stadium.go` + `backend/engine/stadium.go` - capacity,
  ticket price, attendance-driven matchday revenue.
- **Prize money**: `backend/engine/prizemoney.go` - league position + cup round payouts.
- **Transfer windows**: `backend/common/database.go`'s `IsTransferWindowOpen` - one global flag,
  admin-toggled.
- **Fanbase**: `backend/common/fanbase.go` (storage) + `backend/engine/fanbase.go` (drift,
  attendance/commercial multipliers) - a 0-99 per-team index that feeds into attendance,
  sponsorship, and season tickets/merchandising.

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
  wages. Still proposed, not started.
- **Loan money: DONE**, see Status above.
- **Auction/bidding: DONE**, see Status above.

## Phase 5 - Planning tools (frontend-heavy) - DONE, see Status above

## Suggested order

Phases 1, 2, 3, and 5 are done. Phase 4's "loan money" and "auction/bidding" items are done too;
only "amortized transfer fees" is left, independent of everything else - take it next or skip if
the user prefers to prioritize non-economy features instead.
