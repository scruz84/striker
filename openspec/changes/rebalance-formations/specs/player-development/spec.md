# Spec Delta

## MODIFIED Requirements

### Requirement: Match-triggered skill drift
After every league match, cup tie and friendly, each player who featured SHALL get an independent chance for each skill used by the position he played most minutes at in that match. A skill is used by a position when it weighs at least 15% in that position's slot value. The chance per skill is minutes played in that match ÷ 900.

#### Scenario: Full match as a striker
- **WHEN** a striker plays all 90 minutes
- **THEN** each of Shot and Dribbling has a 10% chance to drift and his Pass and Defense never move

#### Scenario: Friendly
- **WHEN** a player features in a friendly
- **THEN** his skills can drift exactly as after a competitive match

#### Scenario: Position skill sets
- **WHEN** the most-played position is decided
- **THEN** goalkeeper drifts GoalKeeper; central defender Defense; full-back Lateral, Defense and Midfield; central midfielder Midfield, Pass and Shot; wide midfielder Lateral, Shot and Midfield; striker Shot and Dribbling; wide striker Shot, Lateral and Dribbling

#### Scenario: Slot weights
- **WHEN** a slot value is computed for drift, lineups and transfers
- **THEN** it uses these weights (GoalKeeper/Defense/Lateral/Pass/Midfield/Dribbling/Shot, in %): goalkeeper 100/0/0/0/0/0/0; central defender 0/90/0/0/10/0/0; full-back 0/35/50/0/15/0/0; central midfielder 0/10/0/20/50/5/15; wide midfielder 0/0/40/10/20/5/25; striker 0/0/5/10/0/20/65; wide striker 0/0/30/5/0/15/50
