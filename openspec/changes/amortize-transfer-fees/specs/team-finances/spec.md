## MODIFIED Requirements

### Requirement: Transfers
A fixed-price listing purchase or an accepted offer SHALL create an installment schedule for the agreed fee over the player's new contract length, charged to the buyer (and paid to the seller) once per league round alongside wages, each installment as its own ledger entry. A free-agent signing costs only the wage, plus a premium of 4× the agreed wage to his last club if signed within his first season of free agency.

#### Scenario: Three-season contract
- **WHEN** a team buys a player for 300,000 on a three-season contract
- **THEN** no 300,000 debit is made at signing
- **AND** the fee is charged in equal installments over the rounds of those three seasons

#### Scenario: Installment ledger entries
- **WHEN** a league round's wages are charged
- **THEN** each active schedule also charges one installment with its own transaction reason

#### Scenario: Competing offers
- **WHEN** several teams have pending offers on one player (at most one per team)
- **THEN** the seller's Offers tab groups them under that player, sorted by amount
