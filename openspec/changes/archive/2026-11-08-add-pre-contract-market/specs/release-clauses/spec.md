# Spec Delta

## ADDED Requirements

### Requirement: Clause on market rows
Every transfer market card (listings, loans, free agents) and every row of the pre-contract market SHALL show the player's release clause when he has one, and the "Last season" marker when he is another club's player in the last season of his contract.

#### Scenario: Listed player with a clause
- **WHEN** a transfer-listed player has a 900,000 clause
- **THEN** his card shows "Release clause 900,000"

#### Scenario: No clause
- **WHEN** a listed player has no clause and 2 seasons left
- **THEN** his card shows neither
