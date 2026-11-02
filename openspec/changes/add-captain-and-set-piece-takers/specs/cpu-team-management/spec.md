# Spec Delta

## ADDED Requirements

### Requirement: CPU captain
At kickoff of every match a CPU team SHALL name as captain its oldest outfield starter, ties going to the lowest player id, and leave both set-piece takers automatic.

#### Scenario: Veteran captain
- **WHEN** a CPU team's outfield starters are aged 24, 31 and 33 among others, with 33 the oldest
- **THEN** the 33-year-old is its captain

#### Scenario: Takers
- **WHEN** a CPU team wins a penalty
- **THEN** its taker is chosen by the automatic rule
