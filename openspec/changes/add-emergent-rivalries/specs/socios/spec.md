# Spec Delta

## MODIFIED Requirements

### Requirement: Attendance factors
The factors SHALL be: price `1 − (price − 10) / 30` clamped to [0, 1.3]; form `0.8 + 0.4 × points in the last 5 league matches / 15` (1.0 without history); rival `0.85 + 0.30 × visitor Fanbase / 99`, × (1 + 0.1 × cup round), × 1.10 for a Rival match or × 1.20 for a Clásico (see rivalries); weather sunny 1.05, cloudy 1.0, windy 0.92, rain 0.82; noise ±10%.

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

#### Scenario: Clásico league match
- **WHEN** a Fanbase-50 visitor is the home side's Clásico rival in a league match
- **THEN** the rival factor is (0.85 + 0.30 × 50/99) × 1.20 ≈ 1.20

#### Scenario: Friendly between rivals
- **WHEN** two Clásico rivals play a friendly
- **THEN** no rivalry multiplier is applied
