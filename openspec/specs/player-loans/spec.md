# player-loans Specification

## Purpose

How a club sees the players it has out on loan and the players it has borrowed: the "Loaned out"
section of the team page, the "Loan" chip in a squad list, the dashboard card and the loan fields of
the team-details API. The loan market itself (listing, accepting, fees, returns) is specified with
the transfer market.

## Requirements

### Requirement: Loaned-out section on the team page
The squad tab of every club SHALL show, below the squad list, a "Loaned out (n)" section listing each player the club has out on loan: name, position, age, borrowing club (link, with its country's flag) and its division, return season, goals for the borrower, and the lender's wage share and buy-option fee when the loan carries them. The return season counts in the borrowing club's country's own seasons, so it MUST name that country and its current season. It MUST be hidden when the club has nobody out, and is visible to every visitor.

#### Scenario: Owner loans a player out
- **WHEN** a manager's 19-year-old striker is loaned for 1 season to an Italian club in division "1B" during Italy's season 4, with a 50% wage share and no buy option
- **THEN** his squad tab shows the striker under "Loaned out (1)" with that club, "1B", "Returns in Italy's season 5 (now in season 4)" and "Wage share 50%", and no buy-option figure

#### Scenario: Loan with a buy option
- **WHEN** the loan carries a buy-option fee of 1,200,000
- **THEN** the row also reads "Buy option 1,200,000"

#### Scenario: Nobody out
- **WHEN** a club has no active loan as lender
- **THEN** no "Loaned out" section is rendered

#### Scenario: Another club's page
- **WHEN** a visitor opens the squad tab of a club that has two players out on loan
- **THEN** the same section lists both players

#### Scenario: Lender was reset
- **WHEN** a loan is flagged to release the player on return because the lending club was reset
- **THEN** the player does not appear in the reset club's "Loaned out" section

### Requirement: Loaned-out list follows the loan's life
A player SHALL appear in his lender's "Loaned out" section from the moment the loan is accepted until it ends: he MUST leave it when he returns at the rollover of the return season and when the borrower exercises the buy option.

#### Scenario: Return at rollover
- **WHEN** the season rollover returns the striker to his parent club
- **THEN** he is back in the parent club's squad list and no longer under "Loaned out"

#### Scenario: Buy option exercised
- **WHEN** the borrower pays the buy option
- **THEN** the player disappears from the former lender's "Loaned out" section and the borrower's "Loan" chip on him is gone

### Requirement: Loaned-in marker
A club's squad list SHALL mark every player on loan to it from another club with a "Loan" chip whose tooltip names the parent club and the return season.

#### Scenario: Borrowed player in the squad
- **WHEN** a club has a midfielder on loan from another club until season 6
- **THEN** his squad row shows a "Loan" chip with the tooltip "On loan from <club> until season 6"

### Requirement: Dashboard loaned-out card
The manager's dashboard SHALL show a "Loaned out" card when his club has at least one player out on loan: the count, up to 3 players with their borrowing club and the return season in that club's country, and a link to the squad tab of his team page. It MUST NOT render when nobody is out.

#### Scenario: Four players out
- **WHEN** the manager has 4 players out on loan
- **THEN** the card reads "Loaned out: 4", lists the 3 returning soonest, and "See all" opens his team page on the squad tab

### Requirement: Loans in the team details API
The own-team and public team-details responses SHALL include `loanedOut` (array, empty when none) with each loaned-out player and the loan terms, and `loanedIn` (array of player id, parent club id and name, return season). National-team responses return both empty.

#### Scenario: Public endpoint
- **WHEN** an anonymous client requests a club's team details while it has one player out on loan
- **THEN** `loanedOut` has one entry with `borrowingTeamId`, `borrowingTeamName`, `borrowingDivisionId`, `borrowingDivisionName`, `borrowingCountry`, `borrowingCountrySeason`, `returnSeason`, `wageContributionPercent`, `buyOptionFee` and `goalsForBorrower`

## Code map

- Backend: `backend/common/loans.go` (`GetLoanedOutPlayers`, `GetLoanedOutPlayersStmtSql`,
  `LoanedOutPlayer`, `GetLoanedInPlayers`, `LoanedInPlayer`), `backend/common/fmstructs.go`
  (`TeamDetails.LoanedOut`, `TeamDetails.LoanedIn`), `backend/common/database.go`
  (`GetFullTeamDetailsByUser`, `GetTeamDetailsById`, `getNationalTeamDetails`);
  `backend/engine/catalog.go` (`GetTeamDetails` caches the public view for about a minute).
- Frontend: `frontend/src/features/team/LoanedOutSection.tsx`, `features/team/SquadList.tsx` (loan
  chip), `features/team/TeamDetailsPage.tsx`, `features/dashboard/DashboardPage.tsx`
  (`LoanedOutCard`, `soonestLoanReturns`), `api/types.ts` (`LoanedOutPlayer`, `LoanedInPlayer`);
  strings under `team.loanedOut.*`, `team.loanedIn*`, `dashboard.loanedOut.*` in
  `i18n/locales/{en,es}.json`.
- Tests: `backend/common/loaned_out_players_integration_test.go`,
  `backend/handlers/teamLoansHandler_test.go`, `frontend/src/features/team/LoanedOutSection.test.tsx`,
  `SquadList.test.tsx`, `frontend/src/features/dashboard/DashboardPage.test.ts`.
