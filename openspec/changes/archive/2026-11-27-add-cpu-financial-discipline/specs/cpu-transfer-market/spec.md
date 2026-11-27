# Spec Delta

## MODIFIED Requirements

### Requirement: Transfer pass
Every CPU team SHALL run one shopping pass per management tick, after roster upkeep, with no signing quota, except while in financial crisis (budget below 0), when it runs none. A Caretaker or a Manager signs at most 1 player per pass; a Director up to 3, of which at most 2 new offers. Accepting a loan and exercising a buy option each count as one signing.

#### Scenario: Nothing worth buying
- **WHEN** no candidate clears the team's minimum score
- **THEN** it signs nobody

#### Scenario: Caretaker fills a hole
- **WHEN** a Caretaker's squad is under 18 players, or a position is under its minimum depth
- **THEN** it may sign one free agent or listed player that fills it (any position when the squad is short), with a score of at least 0 and at most 1.5 × its valuation, keeping half its budget in reserve

#### Scenario: Caretaker with no hole
- **WHEN** a Caretaker's squad is full and every position at its minimum depth
- **THEN** it signs nobody, however good the candidate

#### Scenario: Loan beats a transfer
- **WHEN** a Manager's best-scoring candidate is a loan listing and the next one a listed player
- **THEN** it accepts the loan and signs nobody else that pass

#### Scenario: Club in crisis
- **WHEN** a Director's budget is −1
- **THEN** it runs no shopping pass, so it makes no signing, loan or offer
