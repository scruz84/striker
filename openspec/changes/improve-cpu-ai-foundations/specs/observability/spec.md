# Spec Delta

## ADDED Requirements

### Requirement: CPU level gauges
The engine SHALL export, from one database query per export tick, the gauges `striker.ai.teams`, `striker.ai.average_budget`, `striker.ai.insolvent_teams` (budget below −200,000) and `striker.ai.points_per_match` (current-season league points ÷ matches), each labelled `ai_level` = `human` (owned clubs), `caretaker`, `manager` or `director`. National teams and clubs awaiting a reset are left out. The admin overview MUST return the same figures.

#### Scenario: Two Directors
- **WHEN** the only Director clubs have budgets of 300,000 and −250,000 and have taken 10 points from 5 and 4 points from 5 league matches
- **THEN** the director series read teams 2, average budget 25,000, insolvent 1 and points per match 1.4

#### Scenario: Owned club
- **WHEN** a human owns a club
- **THEN** it counts in the `human` series whatever AI level it had before

### Requirement: CPU level panels
The Simulated world row SHALL show, per `ai_level`, points per match and average budget as time series and the club and insolvent counts as a table, each read with max() across instances.

#### Scenario: Directors sinking
- **WHEN** Directors' average budget falls for a week while their points per match hold
- **THEN** the average budget panel shows the director line falling
