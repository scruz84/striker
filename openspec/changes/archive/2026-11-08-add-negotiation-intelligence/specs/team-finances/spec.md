# Spec Delta

## MODIFIED Requirements

### Requirement: Contract renewal and release
An owner SHALL be able to renew a player for 1–3 seasons; a renewal meeting the player's expected wage and acceptable term is accepted. The expected wage MUST be at least 100% of his current wage while he has 2 or more seasons left and, in his last season, 75%, 80%, 85%, 90% or 95% of it for a Naive, Casual, Aware, Shrewd or Mastermind player. Releasing a player early MUST cost a severance of 10 × his wage.

#### Scenario: Veteran renewal
- **WHEN** a 33-year-old is offered a renewal
- **THEN** he accepts at most 1 season (2 seasons from 30 to 32, 3 under 30, longer only for a declining player)

#### Scenario: No free wage cut
- **WHEN** a player earning 2,000 with 3 seasons left has a market wage of 1,500 and is offered 1,600
- **THEN** he declines and reports an expected wage of 2,000

#### Scenario: Last-season veteran
- **WHEN** an Aware player earning 2,000 in his last season has a market wage of 1,500
- **THEN** his expected wage is 1,700 (85% of 2,000)

#### Scenario: Shrewd last-season veteran
- **WHEN** the same player is Shrewd
- **THEN** his expected wage is 1,800 (90%)

#### Scenario: Improving player
- **WHEN** a player earning 1,000 has a market wage of 1,400 including his improvement raise
- **THEN** his expected wage is 1,400
