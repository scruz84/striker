# Spec Delta

## MODIFIED Requirements

### Requirement: World Cup cadence
A new World Cup SHALL start when the international cup counter (advanced every 60 days by the daily 03:00 UTC World Cup job, which no longer starts regional club cups) reaches a multiple of 4, no World Cup is running, and at least 8 countries are active. Every active country enters.

#### Scenario: Fourth international cycle
- **WHEN** the international cup counter reaches 8 and no World Cup is active with 12 active countries
- **THEN** a World Cup starts with all 12

#### Scenario: Too few countries
- **WHEN** only 6 countries are active
- **THEN** no World Cup starts

#### Scenario: Several engine replicas
- **WHEN** the World Cup job fires on several replicas at once with a cycle due
- **THEN** exactly one claims the cycle, so at most one World Cup starts

#### Scenario: Regional cups are independent
- **WHEN** the daily job claims a new cycle
- **THEN** no regional club cup is started by it
