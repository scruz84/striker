# Spec Delta

## ADDED Requirements

### Requirement: Conditional match orders
A lineup that can schedule substitutions SHALL also hold up to 3 conditional orders, each with a from-minute from 1 to 89, a score condition (losing, losing by 2+, drawing, winning, winning by 2+, from the side's own point of view) and one action: set the mentality, or substitute a starter for a bench player. A saved order MUST pass the same player checks as a scheduled substitution.

#### Scenario: Valid orders
- **WHEN** a manager saves "from 60, if losing, go Attacking" and "from 75, if winning, striker A off for defender B"
- **THEN** both orders are stored with the lineup

#### Scenario: Too many orders
- **WHEN** a manager saves 4 orders
- **THEN** the lineup is rejected with a validation error

#### Scenario: Invalid substitution order
- **WHEN** an order's incoming player is also a starter
- **THEN** the lineup is rejected, the same as an invalid scheduled substitution

### Requirement: Order firing
In the substitution stage of each minute, after scheduled substitutions, every unfired order whose from-minute is at most the current minute SHALL be checked, in slot order, against the score at the start of that minute. An order whose condition holds fires once and is never checked again. An order that never meets its condition by minute 90 MUST NOT fire.

#### Scenario: Going behind late
- **WHEN** an order "from 60, if losing, go Attacking" exists and the side goes 0–1 down in minute 72
- **THEN** the side plays Attacking from minute 73 to the end

#### Scenario: Condition met before the from-minute
- **WHEN** a side is losing from minute 20 and its order is "from 60, if losing, go Attacking"
- **THEN** the order fires in minute 60

#### Scenario: Condition never met
- **WHEN** a side wins 2–0 and its only order is "from 60, if losing, go Attacking"
- **THEN** the order never fires and the side stays at its kickoff mentality

#### Scenario: Two orders on the same minute
- **WHEN** orders in slots 1 and 2 both become due and hold in the same minute
- **THEN** slot 1 fires first, then slot 2

### Requirement: Substitution orders
A substitution order that fires SHALL be applied with the scheduled-substitution rules: it needs its outgoing player on the pitch and fewer than 3 substitutions used, counting injury replacements and scheduled substitutions, and an ineligible incoming player is replaced by the best eligible bench player for that position. A substitution order that cannot be applied MUST be spent without counting against the 3.

#### Scenario: Cap already reached
- **WHEN** a side has used 3 substitutions and a substitution order's condition holds
- **THEN** the order is spent and no substitution happens

#### Scenario: Outgoing player sent off
- **WHEN** a substitution order's outgoing player was sent off earlier
- **THEN** the order is spent without a substitution

### Requirement: Mentality change event
A fired mentality order that changes the side's mentality SHALL be recorded as a `MENTALITY_CHANGE` event (type 8) at that minute, naming the team and the new mentality. An order that sets the mentality the side already has is spent without an event.

#### Scenario: Switch to Defensive
- **WHEN** a Balanced side's order "from 75, if winning, go Defensive" fires in minute 75
- **THEN** a `MENTALITY_CHANGE` event for minute 75 with mentality Defensive is stored

#### Scenario: Already Attacking
- **WHEN** an Attacking side's order "go Attacking" fires
- **THEN** no event is stored

## MODIFIED Requirements

### Requirement: Per-minute simulation pipeline
The simulator SHALL run, for every minute from 1 to 90, the stages in this order: substitutions (scheduled substitutions, then conditional orders), stamina (fatigue), team qualification, scorer, discipline, injury.

#### Scenario: A minute is simulated
- **WHEN** minute N is simulated
- **THEN** due scheduled substitutions are applied first, then due conditional orders
- **AND** fatigue-adjusted player copies are rebuilt before qualification and scoring read them
- **AND** cards and injuries are rolled after the scoring stage
