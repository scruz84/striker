# Spec Delta

## ADDED Requirements

### Requirement: CPU season objective
A CPU club SHALL derive a season objective from its league table once it has played a third of its league matches (`open` before): `relegation_fight` within 3 points of the relegation zone, else `promotion_push` within 3 points of the last promotion place (the title in a top division), else `nothing_at_stake` when it can neither reach that place nor be caught by the zone at 3 points per remaining match, else `open`.

#### Scenario: Early season
- **WHEN** a club has played 9 of its 30 league matches
- **THEN** its objective is `open` whatever its position

#### Scenario: Just above the drop
- **WHEN** with 2 relegation places a club has 20 points and the best side in the zone has 18, after 12 of 30 matches
- **THEN** its objective is `relegation_fight`

#### Scenario: Chasing promotion
- **WHEN** with 2 promotion places a club in 4th has 40 points and 2nd place has 42
- **THEN** its objective is `promotion_push`

#### Scenario: Dead rubber
- **WHEN** with 3 league matches left a club has 45 points, 2nd place has 60 and the best side in the relegation zone 30
- **THEN** its objective is `nothing_at_stake`, since 45 + 9 < 60 and 30 + 9 < 45

#### Scenario: Points compared
- **WHEN** the objective is computed
- **THEN** "within 3 points of the zone" means points ≤ those of the best-placed side in the relegation zone + 3 (only with a tier below), and "within 3 points of the place" means points ≥ that place's points − 3

#### Scenario: Top division
- **WHEN** a tier-1 club with no tier above is 2 points behind the leader after 20 of 30 matches
- **THEN** its objective is `promotion_push` (the title race)

#### Scenario: Playoff places count
- **WHEN** a club plays in a middle tier of a pyramid that follows the league-pyramid ratio rule
- **THEN** its promotion places are 1st and 2nd (direct and playoff) and its relegation zone is the bottom 4 (playoff and direct), so 2 promotion and 4 relegation places

### Requirement: Youth minutes when nothing is at stake
At the kickoff of a league match a Director whose objective is `nothing_at_stake` SHALL start up to 2 fit players aged 21 or under from its bench: for each of its outfield starters from the lowest-valued up, the best such player at his slot replaces him when worth at least 80% of him. Its substitutions, orders and captain are then planned for that XI.

#### Scenario: Two prospects get a start
- **WHEN** a Director with nothing at stake has bench players aged 19 and 20 worth 85% and 90% of its two weakest outfield starters at their slots
- **THEN** both start in their place

#### Scenario: Prospect too raw
- **WHEN** its only bench player under 22 is worth 60% of the weakest starter at his slot
- **THEN** the XI is unchanged

#### Scenario: Still something to play for
- **WHEN** the same Director's objective is `relegation_fight`
- **THEN** it plays its usual XI
