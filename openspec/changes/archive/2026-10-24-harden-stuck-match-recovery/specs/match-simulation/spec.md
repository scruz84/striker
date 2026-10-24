# Spec Delta

## MODIFIED Requirements

### Requirement: Post-match processing
After a league match the engine SHALL record the result and standings, update finances, charge both sides' matchday wages, and store the match's attendance and weather in the report. The weather MUST be drawn once and used both for attendance and the report.

#### Scenario: League match finishes simulating
- **WHEN** a league match has been simulated
- **THEN** ticket revenue, sponsorship and wages are booked immediately
- **AND** availability updates, skill growth and the result notification wait for the live replay (see match-scheduling)

#### Scenario: Result save fails
- **WHEN** writing the result, events, instants or standings fails
- **THEN** none of them is stored, no budget changes, no replay starts, and the match stays unplayed for the next attempt

#### Scenario: Leftovers from an earlier failed attempt
- **WHEN** a match still has events or instants stored by an earlier failed attempt
- **THEN** the successful save replaces them, so the match keeps exactly one set

## ADDED Requirements

### Requirement: Default result record
A fixture settled after repeated simulation failures SHALL be stored as played 0-0 with a single `MATCH_DEFAULT_RESULT` event (type 7) at minute 0 and no instants, lineups or report flavor. Any events or instants left by failed attempts MUST be removed.

#### Scenario: League match defaulted
- **WHEN** a league match is settled 0-0 by default
- **THEN** both sides get a draw in the standings (1 point, 0 goals) and are charged their matchday wages
- **AND** no ticket revenue or sponsorship is booked, and no skill growth happens
- **AND** the matchday counts against both sides' already-suspended or injured players, and each owner gets the result notification

#### Scenario: Cup tie defaulted
- **WHEN** a cup tie is settled 0-0 by default
- **THEN** it goes to the existing shoot-out (each side wins with probability 0.5), stored as `went_to_penalties = true`, and the winner gets the round prize

#### Scenario: World Cup group match defaulted
- **WHEN** a World Cup group friendly is settled 0-0 by default
- **THEN** both national teams get a draw in the group standings

#### Scenario: Viewer opens a defaulted match
- **WHEN** a viewer opens a match settled by default
- **THEN** the page shows the final 0-0 and the default-result event instead of a live replay
