# Spec Delta

## MODIFIED Requirements

### Requirement: International cups and World Cup start
The World Cup cycle SHALL be claimed by one statement that checks the 60-day cadence and increments the counter together. Starting a World Cup MUST hold the national-team scheduling lock (waiting for it), so no friendlies pass can pair a country while its group fixtures are created. Regional club cup editions SHALL be started under one global lock taken without waiting, with each region's state re-read inside it.

#### Scenario: Daily job on two replicas
- **WHEN** the 03:00 UTC job fires on two replicas with a cycle due
- **THEN** exactly one claims it

#### Scenario: Regional cup check on three replicas
- **WHEN** the 5-minute regional cup check runs on three replicas with a European Cup due
- **THEN** exactly one European Cup edition and one set of entries are created
