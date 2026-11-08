# Spec Delta

## MODIFIED Requirements

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
