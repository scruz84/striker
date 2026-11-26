# Spec Delta

## MODIFIED Requirements

### Requirement: Lineup and formation
Every CPU team SHALL re-pick its best XI from the current squad whenever it acts and again at the kickoff of each of its matches, ranking the six formations by the projected expected goal difference of each formation's best XI against the same squad's best 4-4-2 XI: a Manager or Director takes the first, a Caretaker the third or fourth at random; a Director then re-picks against its opponent at competitive kickoffs (see "Director formation against the opponent"). A Manager or Director MUST pick only from players available for that match; a Caretaker picks from the whole squad.

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
- **WHEN** a Manager's best 3-4-3 XI projects a higher expected goal difference against its own best 4-4-2 XI than any other formation's best XI
- **THEN** it plays 3-4-3

#### Scenario: Ranking ties
- **WHEN** two formations project the same expected goal difference within 0.000000001
- **THEN** the one whose XI has the higher summed slot value ranks first, and after that the earlier one in the order 4-4-2, 5-3-2, 3-4-3, 4-3-3, 4-5-1, 3-5-2


#### Scenario: Best XI by optimal assignment
- **WHEN** a squad's only flank players are a wide midfielder (Lateral, Midfield and Pass 75, the rest 25) and a full-back (Lateral and Defense 60, the rest 40), alongside a keeper, two centre-backs and weak reserves
- **THEN** the full-back plays full-back (slot value 57) and the wide midfielder wide midfield (60), because the XI is the assignment of players to slots with the highest total slot value
- **AND** the slot-by-slot pick, which reaches the back line first, would have put the wide midfielder at full-back (58) and a reserve in wide midfield
