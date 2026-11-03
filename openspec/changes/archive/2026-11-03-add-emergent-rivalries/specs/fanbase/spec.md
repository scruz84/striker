# Spec Delta

## MODIFIED Requirements

### Requirement: Result drift
After each league match or cup tie (never a friendly) a team SHALL have a chance of a result drift: +1 for a win, −1 for a loss, nothing for a draw. The chance is 15%, or 30% in a Rival match and 45% in a Clásico (see rivalries).

#### Scenario: Win with a successful roll
- **WHEN** a team wins and its 15% roll succeeds
- **THEN** its Fanbase rises by 1

#### Scenario: Clásico defeat
- **WHEN** a team loses a Clásico
- **THEN** its Fanbase drops by 1 with probability 45%
