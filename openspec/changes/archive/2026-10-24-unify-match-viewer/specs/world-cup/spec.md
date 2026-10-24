# Spec Delta

## MODIFIED Requirements

### Requirement: Group stage
Countries SHALL be split into balanced groups of about 4 (sizes differing by at most one), each playing a full round-robin, one matchday a week from the nearest Saturday. Every group match MUST be an ordinary friendly row linked to the group.

#### Scenario: Group match played
- **WHEN** a linked group friendly is simulated
- **THEN** both teams' group record (played, won, drawn, lost, goals) is updated

#### Scenario: Viewing a group match
- **WHEN** a user opens a group match
- **THEN** it renders through the shared match viewer (see match-viewer), labelled with its group and matchday
- **AND** the friendly detail response carries that group name and matchday, which are empty for a friendly not linked to a group
