# Team economy roadmap

This tracks a multi-phase plan for deepening Striker's team-economy mechanics (budget, wages,
transfers, contracts, income). Written so a fresh session can pick up any phase without needing
the conversation that produced it - each phase lists the mechanic, why it matters, and where to
start reading in the code.

## Status

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
- Phases 3-5 below are proposed, not started.

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
- **Sponsorship**: `backend/engine/sponsorship.go` - flat per-match amount by league tier.
- **Stadium/tickets**: `backend/common/stadium.go` + `backend/engine/stadium.go` - capacity,
  ticket price, attendance-driven matchday revenue.
- **Prize money**: `backend/engine/prizemoney.go` - league position + cup round payouts.
- **Transfer windows**: `backend/common/database.go`'s `IsTransferWindowOpen` - one global flag,
  admin-toggled.

## Phase 2 - Financial consequences (insolvency) - DONE, see Status above

The optional harsher tier (forced-sale flag after N weeks in debt, or a league-points penalty at
rollover) was not built - the spend-side lockout was judged sufficient. Revisit only if playtesting
shows the debt ceiling alone isn't enough friction.

## Phase 3 - Richer income streams

Adds variety without new failure states - safe to build without touching Phase 2.

- **Season tickets**: a lump sum credited once per season (at rollover, in
  `engine/season.go`/`AdvanceSeasonIfComplete`), based on stadium capacity - independent of the
  existing per-match attendance-driven ticket revenue in `engine/stadium.go`. New reason code
  `TRANSACTION_SEASON_TICKETS` in `fmconstants.go` (next free value after
  `TRANSACTION_CONTRACT_RELEASE = 8`, so `9`) - remember to add the matching frontend
  `TRANSACTION_REASON_KEYS` entry and i18n string in both `en.json`/`es.json`, the same way
  `TRANSACTION_CONTRACT_RELEASE` was wired up in this last phase (a gap existed for
  `TRANSACTION_STADIUM_UPGRADE` for months because someone forgot this step - don't repeat it).
- **Merchandising**: small steady income tied to league position/results streak - a good fit
  inside `engine/prizemoney.go` or a new `engine/merchandising.go` alongside it, run at the same
  point in the season-rollover sequence as `awardLeaguePrizeMoney`.
- **Sponsor tiers**: replace the flat `SponsorshipPerMatchTier1/2/3` constants in
  `engine/sponsorship.go` with a per-team choice between 2-3 sponsor deals (higher guaranteed pay
  vs. performance bonuses), re-offered each season. Needs a new `teams.sponsor_tier` column (or
  similar) and a small settings UI, probably on `FinancesPage.tsx` next to the ticket-price form.

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

## Phase 5 - Planning tools (frontend-heavy)

- **Cash-flow forecast**: on `FinancesPage.tsx`, project income (ticket revenue estimate from
  current attendance trend, sponsorship, upcoming prize money) against the current wage bill
  (`GetTeamWageBill`) for the next few match weeks, so a crunch is visible before it happens. Pure
  frontend computation from data already available via `useFinances`/`useUserTeam` - no new
  backend endpoint needed unless historical attendance isn't exposed yet.
- **Income-breakdown chart**: group `TeamTransaction.reason` values from the existing
  transactions list into a pie/bar chart (ticket/sponsorship/prize money/wages/etc.) - use the
  `dataviz` skill for chart styling/palette when building this.
- **Budget-over-time line chart**: cumulative running balance from the same transactions list,
  same skill guidance applies.

## Suggested order

Phases 1-2 are done. Phases 3-5 are additive and can be done in any order from here, or skipped if the user
prefers to prioritize non-economy features instead.
