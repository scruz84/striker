# Spec Delta

## Purpose

Lets a contract carry a public release clause that any club may pay to sign the player without
his club's consent, in exchange for a cheaper wage when the clause is low.

## ADDED Requirements

### Requirement: Setting a clause
A renewal, a free-agent signing, a transfer offer's personal terms and a pre-contract SHALL accept an optional release clause of at least 1.5 × the player's market value on a 3-season contract. A new contract without a clause MUST clear the previous one, and a contract that expires clears it too.

#### Scenario: Clause too low
- **WHEN** a player's market value is 400,000 and a renewal carries a clause of 500,000
- **THEN** the renewal fails with a validation error

#### Scenario: Renewal without a clause
- **WHEN** a player with a 900,000 clause is renewed without one
- **THEN** he has no clause afterwards

### Requirement: Wage discount for a clause
A player SHALL accept 10% less than his expected wage when the clause is at most 2 × his market value on a 3-season contract, and 5% less when it is at most 4 ×.

#### Scenario: Low clause
- **WHEN** a player whose value is 400,000 and expected wage 2,000 is offered a clause of 800,000
- **THEN** he accepts a wage of 1,800

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
