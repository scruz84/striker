# Spec Delta

## ADDED Requirements

### Requirement: CPU mentality
At the kickoff of a league match or cup tie a CPU team SHALL pick its mentality from the ratio R of its kickoff XI's strength to the opponent's kickoff XI's strength (sum of each player's value at his slot). A Caretaker is always Balanced. A Manager is Attacking when R ≥ 1.15, Defensive when R ≤ 0.87, Balanced otherwise. A Director first multiplies R by 1.07 at home and by 1/1.07 away, then uses 1.10 and 0.90. In friendlies every CPU team is Balanced.

#### Scenario: Manager favourite
- **WHEN** a Manager's XI is worth 1,150 against an opponent's 1,000
- **THEN** it plays Attacking

#### Scenario: Director away underdog
- **WHEN** a Director's XI is worth 950 against 1,000 away from home
- **THEN** R becomes 0.95 / 1.07 ≈ 0.888 and it plays Defensive

#### Scenario: Caretaker
- **WHEN** a Caretaker plays any match
- **THEN** it plays Balanced
