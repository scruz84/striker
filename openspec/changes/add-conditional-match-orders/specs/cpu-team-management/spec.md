# Spec Delta

## ADDED Requirements

### Requirement: CPU conditional orders
At the kickoff of a league match or cup tie a CPU team SHALL add conditional orders by AI level. A Caretaker adds none. A Manager adds "from 70, if losing, go Attacking". A Director adds "from 65, if losing, go Attacking" and "from 75, if winning, go Defensive". In friendlies CPU teams add no orders.

#### Scenario: Director protecting a lead
- **WHEN** a Director leads 1–0 at the start of minute 75
- **THEN** it switches to Defensive in minute 75

#### Scenario: Manager chasing the game
- **WHEN** a Manager is losing 0–1 at the start of minute 80 and was drawing until then
- **THEN** it switches to Attacking in minute 80
