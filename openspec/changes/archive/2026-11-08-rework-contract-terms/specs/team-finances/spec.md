# Spec Delta

## MODIFIED Requirements

### Requirement: Contract renewal and release
An owner SHALL be able to renew a player for 1–3 seasons; a renewal meeting the player's expected wage and acceptable term is accepted. The expected wage MUST be at least 100% of his current wage while he has 2 or more seasons left and 85% of it in his last season. Releasing a player early MUST cost a severance of 10 × his wage.

#### Scenario: Veteran renewal
- **WHEN** a 33-year-old is offered a renewal
- **THEN** he accepts at most 1 season (2 seasons from 30 to 32, 3 under 30, longer only for a declining player)

#### Scenario: No free wage cut
- **WHEN** a player earning 2,000 with 3 seasons left has a market wage of 1,500 and is offered 1,600
- **THEN** he declines and reports an expected wage of 2,000

#### Scenario: Last-season veteran
- **WHEN** a player earning 2,000 in his last season has a market wage of 1,500
- **THEN** his expected wage is 1,700 (85% of 2,000)

#### Scenario: Improving player
- **WHEN** a player earning 1,000 has a market wage of 1,400 including his improvement raise
- **THEN** his expected wage is 1,400

## ADDED Requirements

### Requirement: Personal terms of a transfer offer
A transfer offer's personal terms SHALL be judged with the same intelligence- and trend-aware expected wage and term as a renewal, and the expected wage MUST be at least 100% of the player's current wage. A free agent's expected wage has no floor.

#### Scenario: Pay cut refused
- **WHEN** a buyer offers 1,500 to a player earning 2,000 whose market wage is 1,500
- **THEN** no offer is created and the player reports an expected wage of 2,000

#### Scenario: Improving target
- **WHEN** an improving player's renewal expectation is 1,400 and his current wage 1,000
- **THEN** a transfer offer must also carry at least 1,400

#### Scenario: Free agent
- **WHEN** a free agent last paid 3,000 has a market wage of 2,000
- **THEN** he accepts 2,000
