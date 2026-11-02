# Spec Delta

## MODIFIED Requirements

### Requirement: Crowd influence
Fanbase SHALL affect attendance only through socios targets and as the visiting side's rival factor (see socios). It MUST NOT affect the match simulation directly; it reaches it only through the attendance used by the home advantage (see match-simulation).

#### Scenario: Big visitor
- **WHEN** a Fanbase-99 team visits
- **THEN** the home crowd is ×1.15 compared with ×0.85 for a Fanbase-0 visitor

#### Scenario: Big visitor fills the stadium
- **WHEN** a Fanbase-99 visitor's pull takes the crowd from 80% to 92% of capacity
- **THEN** the home side's multiplier rises from 1.192 to about 1.209, and nothing else about fanbase touches the simulation
