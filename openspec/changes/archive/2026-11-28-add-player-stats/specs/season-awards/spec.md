# Spec Delta

## MODIFIED Requirements

### Requirement: Golden Boot
At season rollover each league SHALL award the Golden Boot to its top scorer in its league matches for the whole season (including matches still being replayed live when the season closes), from the player match stats, ties to the lower player id. A scorer who retired in that same rollover can win it. It MUST be skipped when nobody scored.

#### Scenario: Top scorer exists
- **WHEN** a league's season ends and its top scorer has 14 goals
- **THEN** that player wins the Golden Boot

#### Scenario: No goals
- **WHEN** no goal was scored in the league all season
- **THEN** no Golden Boot is awarded

### Requirement: Player of the Season
Each league SHALL award Player of the Season to the player with the highest score from that season's league match stats: +3 per goal, −1 per yellow card, −3 per red card, ties to the lower player id. A player who retired in that same rollover can win it. It MUST be skipped when no score is positive.

#### Scenario: Goals and cards
- **WHEN** a player scored 5 goals and got 2 yellow cards
- **THEN** his score is 13

#### Scenario: Retired in the rollover
- **WHEN** the season's best scorer of 9 points retires before the awards are handed out
- **THEN** he is still Player of the Season
