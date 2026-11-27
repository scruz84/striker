# Spec Delta

## MODIFIED Requirements

### Requirement: Lineup and formation
Every CPU team SHALL re-pick its best XI from the current squad whenever it acts and again at the kickoff of each of its matches, choosing among the six formations ranked by best-XI strength: a Manager or Director takes the strongest (rank 0), a Caretaker the third or fourth at random; a Director then re-picks against its opponent at competitive kickoffs (see "Director formation against the opponent"). A Manager or Director MUST pick only from players available for that match; a Caretaker picks from the whole squad.

#### Scenario: New signing
- **WHEN** a CPU team signs a better striker
- **THEN** its next lineup refresh can include him

#### Scenario: Injured starter, Manager
- **WHEN** a Manager's best striker is injured at kickoff of a league match
- **THEN** its formation and XI are chosen from the squad without him, so a different formation can win

#### Scenario: Injured starter, Caretaker
- **WHEN** a Caretaker's best striker is injured at kickoff
- **THEN** its XI still names him and the match-day backfill replaces him with the best available player for that position

#### Scenario: Suspended player in a friendly
- **WHEN** a Manager's player serving a suspension is picked for a friendly
- **THEN** he may play, because suspensions only bar league and cup matches

#### Scenario: Manager's shape
- **WHEN** a Manager's 3-4-3 gives a stronger XI than any other formation
- **THEN** it plays 3-4-3

## ADDED Requirements

### Requirement: Director formation against the opponent
At the kickoff of a league match or cup tie a Director SHALL rate the best XI it can field in each of the six formations against its opponent's XI by expected goal difference, computed from the minute-1 zone ratings with the engine's chance-rate, chance-share and conversion formulas and no randomness, and play the highest; ties go to the stronger XI. Its substitutions, orders, captain and mentality are then planned for that XI.

#### Scenario: Exploiting a weak flank
- **WHEN** an opponent's XI has a far weaker right side than left and a Director's 4-3-3 expects a goal difference of +0.4 against it while its strongest-by-value 4-4-2 expects +0.2
- **THEN** the Director plays 4-3-3

#### Scenario: Friendly
- **WHEN** a Director plays a friendly
- **THEN** it keeps its friendly rotation and does not re-pick its formation

#### Scenario: Projection matches the engine
- **WHEN** two identical XIs are projected on a neutral ground
- **THEN** both expect the same goals, about 1.2 open-play goals each per match
