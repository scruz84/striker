# Spec Delta

## ADDED Requirements

### Requirement: AI level after a division change
At the season rollover a CPU club (unowned, AI level 1 or more) that is promoted or relegated SHALL draw a level with its new tier's weights (see "AI levels") and move one step towards it: up one level if the draw is higher, down one if lower, unchanged if equal. A club with a human owner or no AI level is untouched.

#### Scenario: Caretaker reaches the top flight
- **WHEN** a Caretaker is promoted to tier 1 and the draw gives Director
- **THEN** it becomes a Manager

#### Scenario: Director relegated
- **WHEN** a Director is relegated to tier 3 and the draw gives Caretaker
- **THEN** it becomes a Manager

#### Scenario: Same draw
- **WHEN** a promoted Manager draws Manager
- **THEN** it stays a Manager

### Requirement: Club style
Every CPU club SHALL have a style, assigned once from its team id and stored for its whole life (a club with none reads as balanced): of every 5 consecutive ids, 2 are balanced, 1 youth, 1 frugal and 1 ambitious. A Manager's or Director's transfer profile is adjusted by it; a Caretaker ignores it.

#### Scenario: Youth club
- **WHEN** a Manager's style is youth
- **THEN** it signs players for their projected growth and lends out benched under-22s, as a Director does

#### Scenario: Frugal club
- **WHEN** a Director's style is frugal
- **THEN** its budget reserve is 30% instead of 15% and its key-player price multiplier is 2.0 instead of 2.5

#### Scenario: Ambitious club
- **WHEN** a Manager's style is ambitious
- **THEN** its budget reserve is 40%, its minimum score 4 and its key-player multiplier 2.5

#### Scenario: Stable
- **WHEN** the same club's style is read twice, before and after a restart
- **THEN** it is the same

### Requirement: Prospects by potential
On its management pass a Director, or a Manager whose style is youth, that has not promoted an academy prospect into a thin position SHALL promote the prospect with the highest projected value at his best core position, read through its potential noise 3 seasons on, when that value is at least its weakest starter's value at that position, within the weekly promotion cap.

#### Scenario: Gifted prospect
- **WHEN** a Director's 17-year-old academy striker projects to 64 and its weakest starting striker is worth 60
- **THEN** it promotes him to the senior squad

#### Scenario: Ordinary prospect
- **WHEN** its best prospect projects to 50 against a weakest starter of 60
- **THEN** nobody is promoted

#### Scenario: Balanced Manager
- **WHEN** a balanced Manager has the same gifted prospect
- **THEN** it only promotes for a thin position
