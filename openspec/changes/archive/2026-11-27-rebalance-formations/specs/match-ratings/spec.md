# Spec Delta

## MODIFIED Requirements

### Requirement: Zone contribution per minute
Every simulated minute, the engine SHALL record, for each player on the pitch, his position, his own value in the zone duels his position feeds, and the rival's per-player value in the mirrored duels. Both are computed from the same fatigue-adjusted skills and the same position weights the minute's zone strengths use. A player who feeds two duels gets the mean of each figure. Partial line cover counts toward the rival's per-player value but is not a duel of its own.

#### Scenario: Central defender
- **WHEN** a central defender with effective Defense 58 faces two central strikers with attacking values 40 and 50, backed by two central midfielders who each lend 0.1 of their attacking value 40 to the attack
- **THEN** his contribution that minute is 58 against 44.5 (98 over a weight of 2.2)

#### Scenario: Full-back feeds two duels
- **WHEN** a left full-back's flank-defense value is 50 against the rival right attack's 40, and his flank-midfield value is 54 against the rival right midfield's 48
- **THEN** his contribution that minute is 52 against 44

#### Scenario: Wide midfielder feeds two duels
- **WHEN** a 4-4-2's left midfielder has flank-attack value 40 against the rival right defense's 44, and flank-midfield value 40 against the rival right midfield's 48
- **THEN** his contribution that minute is 40 against 46

#### Scenario: Goalkeeper
- **WHEN** a goalkeeper with effective GoalKeeper 62 faces a rival whose attacking players' per-player values average 50
- **THEN** his contribution that minute is 62 against 50

#### Scenario: Empty rival zone
- **WHEN** the rival has no player left in the mirrored duel after a red card
- **THEN** the rival value is 0 and that minute's edge is +1
