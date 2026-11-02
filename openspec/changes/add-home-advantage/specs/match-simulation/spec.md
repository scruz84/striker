# Spec Delta

## ADDED Requirements

### Requirement: Home advantage
Every minute, after the zone blend, each of the home side's nine zone ratings (attack, midfield and defense; centre, left and right) SHALL be multiplied by `1 + 0.08 + 0.14 × fill`, where fill is the match's attendance divided by the home stadium's capacity, 0 when the capacity is 0. World Cup group friendlies and World Cup knockout ties MUST use a multiplier of 1.

#### Scenario: Full stadium
- **WHEN** a league match is played with attendance equal to the home stadium's capacity
- **THEN** every home zone rating is multiplied by 1.22

#### Scenario: Empty stadium
- **WHEN** a home match draws 0 spectators
- **THEN** every home zone rating is multiplied by 1.08

#### Scenario: Half-full stadium
- **WHEN** 6,000 people attend a 12,000-seat stadium
- **THEN** the multiplier is 1 + 0.08 + 0.14 × 0.5 = 1.15

#### Scenario: World Cup group match
- **WHEN** two national teams play a World Cup group friendly
- **THEN** neither side's ratings are multiplied

#### Scenario: Expected effect between even sides
- **WHEN** a large sample of league matches between sides with identical squads is simulated at a half-full stadium
- **THEN** the home side wins 43%–49% of them, the away side 25%–31%, and the home side scores 1.15–1.40 times as many goals

## MODIFIED Requirements

### Requirement: Post-match processing
After a league match the engine SHALL record the result and standings, update finances, charge both sides' matchday wages, and store the match's attendance and weather in the report. The weather and the attendance MUST be drawn once, before the simulation, and used for the home advantage, the gate revenue and the report.

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

#### Scenario: Crowd drawn before kickoff
- **WHEN** a league match is simulated and 9,500 attendance is drawn before it
- **THEN** the home advantage uses 9,500, the gate revenue is for 9,500 spectators and the report shows 9,500

#### Scenario: Retried after a failed save
- **WHEN** a match's first simulation attempt fails to save
- **THEN** no gate is booked for it, and the next attempt draws its weather and attendance again
