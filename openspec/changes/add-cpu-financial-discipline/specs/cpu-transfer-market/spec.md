# Spec Delta

## MODIFIED Requirements

### Requirement: Transfer pass
Every Manager and Director SHALL run one shopping pass per management tick, after roster upkeep, with no signing quota, except while in financial crisis (budget below 0), when it runs none. A Manager signs at most 1 player per pass; a Director up to 3, of which at most 2 new offers. Accepting a loan and exercising a buy option each count as one signing.

#### Scenario: Nothing worth buying
- **WHEN** no candidate clears the team's minimum score
- **THEN** it signs nobody

#### Scenario: Loan beats a transfer
- **WHEN** a Manager's best-scoring candidate is a loan listing and the next one a listed player
- **THEN** it accepts the loan and signs nobody else that pass

#### Scenario: Club in crisis
- **WHEN** a Director's budget is −1
- **THEN** it runs no shopping pass, so it makes no signing, loan or offer
