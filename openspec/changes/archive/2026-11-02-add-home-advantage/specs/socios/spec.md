# Spec Delta

## MODIFIED Requirements

### Requirement: Weather drawn once
The weather SHALL be drawn once per match (league match, cup tie or friendly), before the simulation, and used for attendance and the match report. The attendance is drawn right after it, also before the simulation, so the same crowd sets the home advantage (see match-simulation) and the gate revenue.

#### Scenario: Rainy match
- **WHEN** a match's weather is rain
- **THEN** the report shows rain and attendance used ×0.82

#### Scenario: Rainy cup tie
- **WHEN** a cup tie's weather is drawn as rain
- **THEN** its attendance uses ×0.82 and its stored report weather is rain

#### Scenario: Sunny friendly
- **WHEN** a friendly's weather is drawn as sunny
- **THEN** its attendance uses ×1.05 and its stored report weather is sunny

#### Scenario: Form of the match itself
- **WHEN** the attendance of a league match is drawn
- **THEN** its form factor uses the home side's last 5 league matches before this one
