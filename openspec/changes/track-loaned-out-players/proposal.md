# Proposal

## Why

A loan moves the player's `team_id` to the borrower, so the moment a manager loans a player out he
vanishes from every view of the lender: the squad tab, the team page and the dashboard read
`players.team_id` only. The manager loses track of where his player is, until the return
notification arrives. The only place that still knows is the player's own page ("On loan from…"),
and nothing links to it.

Loans are also missing from the recent-transfers panel of each division (division page and
dashboard). `common.GetRecentLeagueTransfers` rebuilds transfers from paired
`TRANSACTION_TRANSFER_IN/OUT` ledger rows, and a loan writes `TRANSACTION_LOAN_FEE` rows (or none at
all when it is free), so no loan can ever show there. The structured `player_transfers` table
already records every loan (`PLAYER_TRANSFER_LOAN`) and buy option, it is just not read.

## What Changes

- **Loaned-out section on the team page.** The squad tab of every club (own and others) gets a
  "Loaned out" section below the squad list: one row per player the club has out on loan, with the
  borrowing club (link), its division, the season he returns, the wage share the lender still pays
  and the buy-option fee when the loan carries one. Hidden when the club has nobody out.
- **Loaned-in marker.** Players on loan *to* a club get a "Loan" chip in its squad list with the
  parent club in the tooltip, so both ends of a loan read the same way.
- **Dashboard tile.** The manager's dashboard shows a compact "Loaned out" card (count + up to 3
  names with their club) linking to the team page squad tab, only when he has players out.
- **Recent transfers read `player_transfers`.** The division panel lists bought listings,
  accepted offers, buy options, pre-table permanent moves, free-agent signings that paid a premium
  and **loans**, newest first. Each row carries its `kind`; loans show a "Loan" chip and the loan fee,
  or "Free" when there was none.
- **API.** `TeamDetails` gains `loanedOut` and `loanedIn` (both `/v1/user/team` and the public team
  endpoint); `LeagueTransfer` gains `kind`.

## Capabilities

### New Capabilities

- `player-loans`: how a club sees the players it has out on loan and the players it has borrowed.

### Modified Capabilities

- `league-page`: the recent-transfers panel's content (now including loans) becomes a stated
  requirement.

## Impact

- Backend: `common/loans.go` (`GetLoanedOutPlayers`, `LoanedOutPlayer`), `common/database.go`
  (`GetFullTeamDetailsByUser`, `GetTeamDetailsById`), `common/fmstructs.go` (`TeamDetails`),
  `common/leaguetransfers.go` (query rewritten on `player_transfers`, `Kind` field).
- Migration `backend/resources/sql/init93_player_transfers_recent_index.sql`: an index on
  `player_transfers(created_at DESC)` for the league panel. No new `TRANSACTION_*` reason, no new
  notification key.
- Frontend: `features/team/SquadList.tsx`, new `features/team/LoanedOutSection.tsx`,
  `features/team/TeamDetailsPage.tsx`, `features/dashboard/DashboardPage.tsx`,
  `features/leagues/RecentLeagueTransfers.tsx`, `api/types.ts`, en/es locales.

## Out of scope

- Recalling a loaned player early or renegotiating a running loan.
- Detailed per-match stats of the loanee at the borrower (only goals for the borrowing club, which
  `GetPlayerCareerStats` already computes).
- Recording loan *returns* as a `player_transfers` row (see open question 2).
- Changing loan rules, fees or AI lending behaviour (`cpu-loan-market`).

## Open questions for the owner

1. Should another club's "Loaned out" section be public, or visible only to its owner? Proposed:
   public, like the player's transfer history and the loan market, which already expose it.
2. Should a loan ending add a `PLAYER_TRANSFER_LOAN_RETURN` row so the player's history and the
   division panel show "back from loan"? Proposed: no, the return notification already covers the
   lender, and returns at every rollover would flood the panel.
3. Free-agent signings without a premium have never shown in the panel. Keep them out? Proposed:
   keep them out (no money changed hands, no club was left).
