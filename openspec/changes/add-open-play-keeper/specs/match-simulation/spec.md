# Spec Delta

## ADDED Requirements

### Requirement: Defending keeper on every chance
Every GOAL and NO_GOAL event, open play included, SHALL name the defending side's goalkeeper on the pitch at that minute as the other team's player (id and full name). League matches, cup ties and friendlies MUST all store it and return it with the match's events. Events stored before this change return no keeper.

#### Scenario: Open-play save in a cup tie
- **WHEN** an away open-play chance is missed in a cup tie whose home goalkeeper is Ruiz
- **THEN** the stored missed-chance event lists Ruiz as the other team's player, and the cup tie's events endpoint returns him

#### Scenario: Friendly goal
- **WHEN** a home open-play goal is scored in a friendly
- **THEN** the stored goal event lists the away goalkeeper as the other team's player

#### Scenario: Old event
- **WHEN** a cup tie event stored before this change is read
- **THEN** its other-team players and names are empty
