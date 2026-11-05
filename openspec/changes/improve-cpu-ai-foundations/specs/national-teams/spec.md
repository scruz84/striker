# Spec Delta

## MODIFIED Requirements

### Requirement: Call-up shortly before kickoff
Within 24 hours of kickoff the engine SHALL select each national side's XI from its current pool and save it as that match's lineup, once per side. Injured players MUST NOT be selected. The side plays the strongest of the six formations for its fit pool, names a captain by the CPU captain rule, starts Balanced and carries the Director's conditional mentality orders. National-team cup ties get their lineup the same way when each knockout round is created.

#### Scenario: Fixture three days away
- **WHEN** a national-team friendly is three days away
- **THEN** no lineup has been selected yet

#### Scenario: Retry on failure
- **WHEN** one side's call-up succeeded and the other's failed
- **THEN** only the failed side is retried on later ticks

#### Scenario: Two engine replicas
- **WHEN** two engine replicas run the call-up pass at once
- **THEN** each side is called up, and each club notified, once

#### Scenario: Injured star
- **WHEN** a country's best striker is injured for 2 more matches at call-up time
- **THEN** he is not selected and his club is not notified

#### Scenario: Pool full of strikers
- **WHEN** a country's fit pool has 5 strong strikers and 2 good defenders
- **THEN** the lineup uses whichever of the six formations gives the strongest XI, not always 4-4-2

#### Scenario: Orders
- **WHEN** a national side is losing at the start of minute 65
- **THEN** it switches to Attacking, and it switches to Defensive if winning at the start of minute 75
