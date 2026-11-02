# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- `AcceptLoan` sets `players.team_id` to the borrower and inserts a `player_loans` row
  (`from_team_id`, `to_team_id`, `return_season`, `wage_contribution_percent`, `buy_option_fee`).
  `GetPlayersFromTeamId` therefore returns loaned-in players and never loaned-out ones.
- `GetTeamMarketCommitments` already reads `LoanedOut` / `LoanedIn` id sets for the AI, but nothing
  user-facing reads them. `GetPlayerStatus` already reports `onLoanFromTeam*` on the player page.
- `AcceptLoan` and `ExerciseLoanBuyOption` both call `RecordPlayerTransfer` inside their
  transaction, so `player_transfers` holds every loan (kind 3, fee = loan fee, 0 when free) and buy
  option (kind 4). `GetRecentLeagueTransfers` ignores that table and pairs ledger rows instead.
- `player_transfers` only has `(player_id, created_at)` and a partial `fee DESC` index.

## Goals / Non-Goals

**Goals:**
- A lender always sees where each of his loaned players is and when he comes back, from the team
  page and the dashboard, with no extra request beyond the team details already loaded.
- One source of truth for "transfers that happened": `player_transfers`, same as the player history
  and the comparables.

**Non-Goals:**
- No change to loan rules, money flows, notifications or the CPU loan market.

## Decisions

### D1. `LoanedOutPlayer` read in one query
New `common.GetLoanedOutPlayers(dbCtx, teamId) ([]LoanedOutPlayer, error)` in `common/loans.go`
with `GetLoanedOutPlayersStmtSql`:
`SELECT playerColumnsSql, pl.to_team_id, t.name, t.league_id, l.name, pl.return_season,
pl.wage_contribution_percent, pl.buy_option_fee FROM player_loans pl JOIN players p ... JOIN teams t
ON t.team_id = pl.to_team_id JOIN leagues l ON l.league_id = t.league_id WHERE pl.from_team_id = $1
ORDER BY pl.return_season, p.last_name`. `LoanedOutPlayer{Player, BorrowingTeamId,
BorrowingTeamName, BorrowingDivisionId, BorrowingDivisionName, ReturnSeason,
WageContributionPercent, BuyOptionFee, GoalsForBorrower}`; `GoalsForBorrower` comes from
`GetPlayerCareerStats(player, borrower).GoalsForTeam` (at most `AiMaxLoanedOut` rows for a CPU club,
and few for a human, so N small reads are acceptable). A release-on-return loan (reset lender)
never shows: the lender is a fresh CPU club and the player is not coming back to it, so the query
adds `AND NOT pl.release_on_return`.
*Alternative:* a separate `/v1/teams/:id/loans` endpoint. Rejected: the squad tab and dashboard
already load team details; one more round trip for a usually empty list is not worth it.

### D2. Exposing it on `TeamDetails`
`TeamDetails` gains `LoanedOut []LoanedOutPlayer` (`json:"loanedOut"`, always an array) and
`LoanedIn []LoanedInPlayer` (`json:"loanedIn"`: `playerId`, `parentTeamId`, `parentTeamName`,
`returnSeason`, read by the existing `GetLoansToTeamStmtSql` joined with `teams` for the name), so
the squad chip needs no per-player status request. Both are filled by `GetFullTeamDetailsByUser` and
`GetTeamDetailsById` (club branch only; national teams get empty arrays). Visibility is public
(open question 1): loans are already public through the loan market and player histories. The
lender's wage share and buy-option fee are listing terms every visitor could read on the market.

### D3. Recent transfers from `player_transfers`
`getRecentLeagueTransfersStmtSql` becomes:
`SELECT pt.player_id, pt.fee, pt.kind, pt.to_team_id, bt.name, pt.from_team_id, st.name,
pt.created_at FROM player_transfers pt JOIN teams bt ON bt.team_id = pt.to_team_id JOIN teams st ON
st.team_id = pt.from_team_id WHERE (pt.kind IN (0,1,3,4,6) OR (pt.kind = 2 AND pt.from_team_id IS
NOT NULL)) AND (bt.league_id = $1 OR st.league_id = $1) ORDER BY pt.created_at DESC,
pt.transfer_id DESC LIMIT $2` — the inner joins drop releases and premium-free signings (NULL
team). `LeagueTransfer` gains `Kind int` (`json:"kind"`); `Amount` is the fee (≥ 0). The per-row
`GetPlayerById` stays (limit 10). Ledger pairing is gone, so two transfers of the same player at
the same instant can no longer be mis-paired.
*Alternative:* keep the ledger query and union `PLAYER_TRANSFER_LOAN` rows. Rejected: two code
paths for one panel, and free loans have no ledger rows anyway.

### D4. Migration
`backend/resources/sql/init93_player_transfers_recent_index.sql`:
`CREATE INDEX IF NOT EXISTS player_transfers_created_at_idx ON player_transfers(created_at DESC);`
Re-runnable, no data change. No new `TRANSACTION_*` reason, no new notification key.

### D5. Frontend
- `api/types.ts`: `LoanedOutPlayer`, `LoanedInPlayer`, `TeamDetails.loanedOut`, `TeamDetails.loanedIn`,
  `LeagueTransfer.kind`.
- New `features/team/LoanedOutSection.tsx` rendered under `SquadList` in the squad tab when
  `loanedOut.length > 0`: title "Loaned out (n)", rows with player link + flag, position, age,
  borrowing club link and division chip, "Returns season N", optional "Wage share X%" and
  "Buy option €Y", "Goals: G". On narrow screens the terms wrap to a second line.
- `SquadList`: a small "Loan" chip with tooltip "On loan from {parent} until season N" next to players in `loanedIn`; the existing "Loan out"
  action stays hidden for them (already blocked server-side).
- `DashboardPage`: a "Loaned out" card (count, up to 3 names + club, "See all" →
  `/team/:id?tab=squad`) when `loanedOut.length > 0`.
- `RecentLeagueTransfers`: when `kind === PLAYER_TRANSFER_LOAN` show a "Loan" chip and the fee, or
  "Free" when 0; the from/to line reads "{borrower} on loan from {lender}".

## Risks / Trade-offs

- [Panel ordering changes for historic rows] → `player_transfers` was backfilled from the same
  ledger rows (`init78`), so pre-existing transfers keep their timestamps and fees.
- [Team details payload grows] → bounded by the number of active loans of one club (CPU ≤ 3).
- [Team details caching] → `engine.GetTeamDetails` caches the public view for about a minute, so another
  club's page can lag a loan by that long, exactly as its squad already does; the owner's own view
  (`/v1/user/team`) is read uncached.

## Code map (for the new `player-loans` spec)

- Backend: `common/loans.go` (`GetLoanedOutPlayers`, `LoanedOutPlayer`, `GetLoanedOutPlayersStmtSql`),
  `common/loans.go` (`GetLoansToTeamStmtSql`), `common/fmstructs.go`
  (`TeamDetails.LoanedOut`, `TeamDetails.LoanedIn`, `LoanedInPlayer`), `common/database.go`
  (`GetFullTeamDetailsByUser`, `GetTeamDetailsById`, `getNationalTeamDetails`).
- Frontend: `features/team/LoanedOutSection.tsx`, `features/team/SquadList.tsx` (loan chip),
  `features/team/TeamDetailsPage.tsx`, `features/dashboard/DashboardPage.tsx`, `api/types.ts`
  (`LoanedOutPlayer`), locales `team.loanedOut.*`, `dashboard.loanedOut.*`.
- Tests: `common/loaned_out_players_integration_test.go`, `features/team/LoanedOutSection.test.tsx`.
