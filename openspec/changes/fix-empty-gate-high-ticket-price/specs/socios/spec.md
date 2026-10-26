# Spec Delta

## MODIFIED Requirements

### Requirement: Attendance factors
The factors SHALL be: price `1 − (price − 10) / 30` clamped to [0, 1.3]; form `0.8 + 0.4 × points in the last 5 league matches / 15` (1.0 without history); rival `0.85 + 0.30 × visitor Fanbase / 99`, × (1 + 0.1 × cup round); weather sunny 1.05, cloudy 1.0, windy 0.92, rain 0.82; noise ±10%.

#### Scenario: Price 39
- **WHEN** a team charges 39, the highest price allowed
- **THEN** the price factor is 1/30 (about 3% of the crowd at the default price)

#### Scenario: Price 40
- **WHEN** the price factor is evaluated at 40 or more (no team can be charging that)
- **THEN** it is 0 and nobody would attend

#### Scenario: Five straight wins
- **WHEN** the home side won its last five league matches
- **THEN** the form factor is 1.2

#### Scenario: Cup round 2
- **WHEN** a cup tie in round 2 is played
- **THEN** the rival factor is multiplied by 1.2

### Requirement: Optimal ticket price
The optimal price SHALL be the integer price from 1 to 39 that maximises expected `price × min(capacity, demand)` against an average visitor in average weather, ties going to the lower price.

#### Scenario: Demand below capacity
- **WHEN** a club's socios cannot fill its ground
- **THEN** the optimal price is moderate (around 20)

#### Scenario: Demand above capacity
- **WHEN** a club's socios exceed its ground
- **THEN** the optimal price is the one that just fills it

#### Scenario: Shown to the owner
- **WHEN** an owner opens the finance page
- **THEN** the recommended ticket price shown is this optimal price for their team
