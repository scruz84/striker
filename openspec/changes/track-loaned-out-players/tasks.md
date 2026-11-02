# Tasks

## 1. Backend: loaned-out players

- [ ] 1.1 Add `LoanedOutPlayer` and `GetLoanedOutPlayersStmtSql` + `GetLoanedOutPlayers` to `backend/common/loans.go` per design D1 (excluding `release_on_return` loans, ordered by return season then last name, `GoalsForBorrower` via `GetPlayerCareerStats`). Verify with `common/loaned_out_players_integration_test.go`: accept a loan → one row with the borrower, its division, the return season and terms; return it via `ReturnLoan` → empty; exercise a buy option → empty; release-on-return loan → empty.
- [ ] 1.2 Add `LoanedOut` / `LoanedIn` (`LoanedInPlayer`, design D2) to `TeamDetails` (`common/fmstructs.go`, always non-nil arrays) and fill them in `GetFullTeamDetailsByUser` and the club branch of `GetTeamDetailsById`; national teams return empty arrays. Verify with an integration test on both functions and a handler test asserting the JSON keys on the public team endpoint.

## 2. Backend: recent transfers from player_transfers

- [ ] 2.1 Add `backend/resources/sql/init93_player_transfers_recent_index.sql` (`CREATE INDEX IF NOT EXISTS player_transfers_created_at_idx ON player_transfers(created_at DESC)`); verify it applies twice on a fresh test DB.
- [ ] 2.2 Rewrite `getRecentLeagueTransfersStmtSql` / `GetRecentLeagueTransfers` in `common/leaguetransfers.go` on `player_transfers` per design D3 and add `Kind` to `LeagueTransfer`. Verify with an integration test covering the "Recent transfers content" scenarios: free loan in both divisions, loan with fee, listing purchase, buy option, release excluded, premium-free free agent excluded, limit and ordering.

## 3. Frontend

- [ ] 3.1 `frontend/src/api/types.ts`: add `LoanedOutPlayer`, `TeamDetails.loanedOut`, `LoanedInPlayer`, `TeamDetails.loanedIn`, `LeagueTransfer.kind`.
- [ ] 3.2 New `features/team/LoanedOutSection.tsx` rendered under `SquadList` in `TeamDetailsPage` squad tab (hidden when empty); `LoanedOutSection.test.tsx` covers a row with and without wage share / buy option and the empty case.
- [ ] 3.3 `SquadList.tsx`: "Loan" chip with the tooltip "On loan from {parent} until season N" for players in `teamDetails.loanedIn`; test the chip renders only for loaned-in players.
- [ ] 3.4 `DashboardPage.tsx`: "Loaned out" card (count, 3 soonest returns, "See all" → `/team/:id?tab=squad`), hidden when empty; extend `DashboardPage.test.ts`.
- [ ] 3.5 `features/leagues/RecentLeagueTransfers.tsx`: "Loan" chip, "Free" for a zero-fee loan, "{borrower} on loan from {lender}" line; update the header comment; test a loan row and a transfer row.
- [ ] 3.6 Locales `frontend/src/i18n/locales/en.json` and `es.json`: `team.loanedOut.title` ("Loaned out ({{count}})" / "Cedidos ({{count}})"), `team.loanedOut.returns` ("Returns season {{season}}" / "Vuelve la temporada {{season}}"), `team.loanedOut.wageShare` ("Wage share {{percent}}%" / "Pagamos el {{percent}}% de la ficha"), `team.loanedOut.buyOption` ("Buy option {{fee}}" / "Opción de compra {{fee}}"), `team.loanedOut.goals`, `team.loanedInChip` ("Loan" / "Cedido"), `team.loanedInTooltip`, `dashboard.loanedOut.*`, `recentTransfers.loanChip` ("Loan" / "Cesión"), `recentTransfers.onLoanFrom` ("on loan from" / "cedido por"), `recentTransfers.free` ("Free" / "Gratis").

## 4. Specs and docs

- [ ] 4.1 Sync the `player-loans` (new) and `league-page` deltas into `openspec/specs/`, with the Code map sections naming the files above; add `GetRecentLeagueTransfers` and `RecentLeagueTransfers.tsx` to the league-page Code map.
- [ ] 4.2 Mention `TeamDetails.loanedOut` in `ARCHITECTURE.md` if it documents the team-details payload.

## 5. Verification

- [ ] 5.1 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`
- [ ] 5.2 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`
