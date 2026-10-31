# Spec Delta

## MODIFIED Requirements

### Requirement: AI levels
Each unowned club SHALL have an AI level, Caretaker (1), Manager (2) or Director (3), each a strict superset of the one below. New CPU teams draw it by tier: tier 1 10/30/60%, tier 2 30/50/20%, others 60/30/10% (Caretaker/Manager/Director). Admins can override it.

#### Scenario: Tier-1 team created
- **WHEN** a CPU team is created in a tier-1 league
- **THEN** it is a Director with 60% probability

#### Scenario: Released club becomes CPU
- **WHEN** a club released from a human owner finishes its reset in a tier-2 league
- **THEN** it draws a new AI level with the tier-2 weights (30/50/20%) and CPU routines start managing it

#### Scenario: Released club before its reset
- **WHEN** a released club is still waiting for its reset
- **THEN** it has no AI level and no CPU routine acts on it

#### Scenario: AI-managed gate
- **WHEN** any CPU routine checks a team
- **THEN** it only acts if the team has no owner and an AI level of at least Caretaker (national teams have level 0 and are never managed)
