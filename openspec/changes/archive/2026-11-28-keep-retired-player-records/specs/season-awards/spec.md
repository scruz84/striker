# Spec Delta

## MODIFIED Requirements

### Requirement: Award delivery and record
Each award SHALL be stored permanently in `player_season_awards` and notified to the winner's team owner as `NOTIFICATION_AWARD` (distinguished by message key) linking to the player's page. The player's profile MUST show his Golden Boot and MVP counts. The record MUST outlive the player: a winner who retires keeps his awards (see retired-players).

#### Scenario: Winner plays for an owned team
- **WHEN** a player wins the Golden Boot
- **THEN** his owner gets a `notifications.messages.goldenBoot` notification linking to `/player/{id}`
- **AND** his profile's Golden Boot counter increases by one

#### Scenario: Winner retires
- **WHEN** a Golden Boot winner retires two seasons later
- **THEN** his profile still counts that Golden Boot
