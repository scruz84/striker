# release-clauses Specification

## Purpose
Lets a contract carry a public release clause that any club may pay to sign the player without
his club's consent, in exchange for a cheaper wage when the clause is low.

## Requirements

### Requirement: Setting a clause
A renewal, a free-agent signing, a transfer offer's personal terms and a pre-contract SHALL accept an optional release clause of at least 1.5 × the player's market value on a 3-season contract. A new contract without a clause MUST clear the previous one, and a contract that expires clears it too.

#### Scenario: Clause too low
- **WHEN** a player's market value is 400,000 and a renewal carries a clause of 500,000
- **THEN** the renewal fails with a validation error

#### Scenario: Renewal without a clause
- **WHEN** a player with a 900,000 clause is renewed without one
- **THEN** he has no clause afterwards

### Requirement: Wage discount for a clause
A player SHALL accept less than his expected wage when his contract carries a clause of at most 2 × his market value on a 3-season contract - 5%, 7.5%, 10%, 12.5% or 15% less for a Naive, Casual, Aware, Shrewd or Mastermind player - and half that discount for a clause of at most 4 ×.

#### Scenario: Low clause
- **WHEN** an Aware player whose value is 400,000 and expected wage 2,000 is offered a clause of 800,000
- **THEN** he accepts a wage of 1,800

#### Scenario: Mastermind and a low clause
- **WHEN** the same player is a Mastermind
- **THEN** he accepts 1,700

#### Scenario: High clause
- **WHEN** the clause is 2,000,000
- **THEN** he expects his full 2,000

### Requirement: Paying a clause
While the window is open, a solvent club other than the owner SHALL be able to pay a player's clause in full, 14 days or more after it was set, with personal terms he accepts, and sign him at once without the owner's consent. A player on loan or with a signed pre-contract MUST NOT be signed this way.

#### Scenario: Clause paid
- **WHEN** a club pays a 900,000 clause with acceptable terms
- **THEN** 900,000 moves to the owner, the player joins the buyer, and his pending offers and listings are removed

#### Scenario: Too soon
- **WHEN** the clause was set 10 days ago
- **THEN** the payment fails and nothing changes

### Requirement: Clause visibility and notifications
A player's clause SHALL be visible to every manager. A paid clause MUST notify the selling club and the buyer and tell the player's followers he moved.

#### Scenario: Human loses a player
- **WHEN** a CPU Director pays a human's player's clause
- **THEN** the human is told who paid it and how much he received

### Requirement: Clause on market rows
Every transfer market card (listings, loans, free agents) and every row of the pre-contract market SHALL show the player's release clause when he has one, and the "Last season" marker when he is another club's player in the last season of his contract.

#### Scenario: Listed player with a clause
- **WHEN** a transfer-listed player has a 900,000 clause
- **THEN** his card shows "Release clause 900,000"

#### Scenario: No clause
- **WHEN** a listed player has no clause and 2 seasons left
- **THEN** his card shows neither

## Code map

- Core: `backend/common/releaseclauses.go` (`ClauseBaseValue`, `MinReleaseClause`,
  `ReleaseClauseDiscount`, `DiscountedWage`, `ValidateReleaseClause`, `PayReleaseClause`,
  `CpuMayPayClauseOn`, `ReleaseClauseLockIn`); storage `resources/sql/init105_release_clauses.sql`
  (`players.release_clause`, `release_clause_set_at`, `transfer_offers.offered_release_clause`,
  `pre_contract_offers.release_clause`).
- Contracts: `ApplyRenewalOfferStmtSql` / `SignFreeAgentStmtSql` set the clause,
  `ReleaseFromTeamStmtSql` clears it; `MakeContractRenewalOfferWithClause`,
  `SignFreeAgentWithClause`, `MakeTransferOfferWithClause`, `MakePreContractOfferWithClause`;
  `Player.ReleaseClause` / `ReleaseClausePayableAt` read by `scanPlayer` (`playerColumnsSql`).
- API: `handlers/releaseClauseHandlers.go` (`POST /v1/release-clauses`), `releaseClause` in the
  renewal, free-agent, offer and pre-contract bodies.
- Notifications: `notifications.messages.releaseClausePaid` / `releaseClauseSigned`.
- Market rows: `ContractMeta` in `features/transfers/marketUi.tsx` (on `ListingCard` and the last-season rows),
  string `transfers.releaseClause.meta`.
- Frontend: clause field in `components/ContractTermsFields.tsx` (`clauseValue` in
  `utils/contractTerms.ts`), `components/PayReleaseClauseDialog.tsx`, clause row and button in
  `components/PlayerProfileRail.tsx`, hook `usePayReleaseClause`, strings
  `transfers.releaseClause.*`, `team.contractDialog.releaseClause*`.
- Tests: `common/releaseclauses_test.go`, `engine/releaseclauses_integration_test.go`,
  `handlers/releaseClauseHandlers_test.go`, `components/PayReleaseClauseDialog.test.tsx`.
