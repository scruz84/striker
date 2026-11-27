# Spec Delta

## ADDED Requirements

### Requirement: Market urgency
A Manager or Director whose season objective is `relegation_fight` SHALL shop with its minimum score lowered by 2 (never below 1) and half its usual budget reserve; a Director whose objective is `promotion_push` SHALL lower its minimum score by 1. Every other objective leaves its profile unchanged.

#### Scenario: Manager fighting relegation
- **WHEN** a Manager (minimum score 5, reserve 50%) is in a relegation fight
- **THEN** it signs a candidate scoring 3 that leaves 30% of its budget

#### Scenario: Director in a promotion push
- **WHEN** a Director (minimum score 3) is in a promotion push
- **THEN** a candidate scoring 2 clears its minimum

#### Scenario: Manager in a promotion push
- **WHEN** a Manager is in a promotion push
- **THEN** its minimum score stays 5
