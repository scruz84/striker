# Spec Delta

## ADDED Requirements

### Requirement: Recent transfers content
The recent-transfers panel of a division (division page and dashboard) SHALL list the 10 most recent player moves in which the buying or selling club belongs to that division, newest first: bought listings, accepted offers, exercised buy options, permanent moves recorded before structured history, free-agent signings that paid a premium to a previous club, and loans. Releases and premium-free free-agent signings MUST NOT appear.

#### Scenario: Free loan
- **WHEN** a club in division "2A" takes a player on a 1-season loan with no fee from a club in "1A"
- **THEN** the panels of both "2A" and "1A" show the move with a "Loan" chip, "<borrower> on loan from <lender>" and "Free" as the amount

#### Scenario: Loan with a fee
- **WHEN** the loan carried a loan fee of 150,000
- **THEN** the row shows the "Loan" chip and 150,000

#### Scenario: Permanent transfer
- **WHEN** a club in the division buys a listed player for 2,000,000
- **THEN** the row has no chip, reads "<buyer> from <seller>" and shows 2,000,000

#### Scenario: Buy option
- **WHEN** a borrower exercises a 900,000 buy option
- **THEN** a row for 900,000 appears with no "Loan" chip, newer than the original loan row

#### Scenario: Release
- **WHEN** a club in the division releases a player
- **THEN** nothing is added to the panel

### Requirement: Recent transfers endpoint
`GET /v1/team/division/:id/recent-transfers` SHALL read `player_transfers` and return, per row, the player, `amount` (the fee, ≥ 0), `kind` (a `PLAYER_TRANSFER_*` value), buyer and seller team ids and names, and `occurredAt`.

#### Scenario: Loan row
- **WHEN** the latest move in the division is a free loan
- **THEN** the first element has `kind` 3 and `amount` 0, with the borrower as buyer and the lender as seller
