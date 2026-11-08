# pre-contracts Specification

## Purpose
Lets clubs agree a free move for next season with a player in the last season of his contract,
so expiring contracts carry risk for the owning club and opportunity for everyone else.

## Requirements

### Requirement: Last-season marker
A player owned by a club and in the last season of his contract SHALL carry a public "last season" marker visible to every manager; his exact contract years stay visible only to his own club.

#### Scenario: Rival's player
- **WHEN** a manager views another club's player with 1 season left
- **THEN** he sees the "last season" marker, but still not the exact number of seasons on other players' contracts

#### Scenario: Two seasons left
- **WHEN** the player has 2 seasons left
- **THEN** no marker is shown

### Requirement: Making a pre-contract offer
While the transfer window is open, a solvent club SHALL be able to offer a contract for next season (wage and 1–3 years, no fee) to another club's player in his last season, not on loan and with no signed pre-contract, once his club's league has played at least half its matches. A club MUST have at most one pending offer per player and at most 3 signed pre-contracts.

#### Scenario: Too early
- **WHEN** the player's league has played 14 of its 30 rounds
- **THEN** the offer fails with a validation error

#### Scenario: Offer made
- **WHEN** the league has played 15 of 30 rounds and the terms reach the player's demands
- **THEN** the offer is pending and his club is told the buyer, the offered wage and years, and when the player will decide

### Requirement: Player demands
A player SHALL only consider a pre-contract offer whose wage is at least 1.10 × his expected wage and whose length is at most his age's maximum term; an offer below that MUST be refused at once, reporting the wage he would accept.

#### Scenario: Below the premium
- **WHEN** a player's expected wage is 2,000 and the offer is 2,100
- **THEN** no offer is created and he reports 2,200

### Requirement: Signing a pre-contract
72 hours after a player's first pending offer, he SHALL sign the best pending offer that still meets his demands: highest wage, then most years, then the earliest. The other offers are declined. The decision MUST happen once even when several engine replicas run at the same time.

#### Scenario: Two suitors
- **WHEN** after 72 hours a player has offers of 2,400 for 2 years and 2,400 for 3 years
- **THEN** he signs the 3-year offer and the other buyer is told he declined

### Requirement: Right to counter-offer
Before the player decides, his club SHALL be told each offer's buyer, wage and years and the decision time, and MAY counter by renewing him. While offers are pending, his renewal expectation is at least the best pending wage ÷ 1.10, rounded up (he stays for 10% less than the best rival offer). A renewal, sale or loan cancels every pending offer and tells each buyer, without the renewal's terms.

#### Scenario: Matching a rival
- **WHEN** a player's expected wage is 2,000 and the best pending offer is 3,300
- **THEN** his club must offer at least 3,000 to renew him

#### Scenario: Renewed in time
- **WHEN** his club renews him at 3,000, 48 hours after the first offer
- **THEN** every pending offer is cancelled and each buyer is told he stayed

#### Scenario: Better offer during the wait
- **WHEN** a second club offers 3,520 after the first offered 3,300
- **THEN** his club is told, the bar to keep him rises to 3,200 and the decision time does not move

#### Scenario: No counter
- **WHEN** his club does not renew him within the 72 hours
- **THEN** he signs the best pending offer

### Requirement: Pre-contracted player
A player with a signed pre-contract MUST NOT be renewed, released, transfer-listed, loan-listed or receive transfer offers. At his country's next season rollover he SHALL join the buying club on the agreed wage and years, with no fee and no premium to his old club, instead of becoming a free agent.

#### Scenario: Rollover
- **WHEN** his country's season rolls over
- **THEN** he belongs to the buyer with the agreed terms and his transfer history shows a pre-contract move with fee 0

#### Scenario: Listing refused
- **WHEN** his club tries to transfer-list him after he signed
- **THEN** the request fails and nothing changes

#### Scenario: Retirement
- **WHEN** he retires at the rollover
- **THEN** the pre-contract lapses and the buyer is told

### Requirement: Pre-contract notifications
Every pre-contract event SHALL notify the clubs involved with a `NOTIFICATION_PRE_CONTRACT`: an offer received, a reminder to the owning club 24 hours before the decision, signed, signed away, declined, cancelled, and the move at the rollover; followers of the player get the usual "moved" watchlist notification at the move.

#### Scenario: Reminder
- **WHEN** 48 hours have passed since the first offer and the player is still not renewed
- **THEN** his club is reminded that he decides in 24 hours, with the wage it must offer to keep him

#### Scenario: Signed away
- **WHEN** a human's player signs a pre-contract with a CPU club
- **THEN** the human is told the player will leave at the end of the season

## Code map

- Core: `backend/common/precontracts.go` (`MakePreContractOffer` / `MakePreContractOfferWithClause`,
  `WithdrawPreContractOffer`, `DecidePreContracts` with `remindPreContractDecisions` /
  `decidePreContract`, `PreContractCounterBar`, `PreContractMinimumWage`, `PreContractKeepWage`,
  `completePreContractTx`, `LapsePreContractsOnRetirement`, `cancelPendingPreContracts`, the
  `PreContract*` constants); storage `resources/sql/init104_pre_contracts.sql`.
- Wiring: `common.MakeContractRenewalOffer` (counter bar, cancellation), `ReleasePlayer`,
  `CreateTransferListing`, `CreateLoanListing`, `MakeTransferOffer` (refuse a signed player),
  `ExecuteTransfer`, `executeOfferTx`, `AcceptLoan` (cancel pending offers),
  `AdvancePlayerContract` (completion), `ResetReleasedTeam` (clears a reset club's offers),
  `Player.ContractExpiring` set by `RedactFinancials`; `engine/tasks.go` `OfferResponseJob`;
  `engine/progression.go` (retirement).
- API: `handlers/preContractHandlers.go`, routes `/v1/pre-contracts` in `service/fmservice.go`.
- Notifications: `NOTIFICATION_PRE_CONTRACT`, keys `notifications.messages.preContract*` with
  Spanish templates in `common/notificationtranslations.go`.
- Frontend: `components/PreContractDialog.tsx`, `features/transfers/PreContractCards.tsx` (Offers
  tab), `utils/contractTerms.ts` (`isLastSeason`), chips in `features/team/SquadList.tsx`, button in
  `components/PlayerProfileRail.tsx`; hooks `usePreContractsReceived`, `usePreContractsSent`,
  `useMakePreContractOffer`, `useWithdrawPreContractOffer`; strings `transfers.preContract.*`,
  `team.lastSeason*`, `team.signedFor*`.
- Tests: `common/precontracts_test.go`, `engine/precontracts_integration_test.go`,
  `handlers/preContractHandlers_test.go`, `features/transfers/PreContractCards.test.tsx`,
  `utils/contractTerms.test.ts`.
