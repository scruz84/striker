# Spec Delta

## MODIFIED Requirements

### Requirement: Per-minute simulation pipeline
The simulator SHALL run, for every minute from 1 to 45, then each first-half added minute (45+1 … 45+A1), then 46 to 90, then each second-half added minute (90+1 … 90+A2), the stages in this order: substitutions (scheduled substitutions, then conditional orders), stamina (fatigue), team qualification, scorer, discipline, injury. A1 and A2 are fixed as described in stoppage-time.

#### Scenario: A minute is simulated
- **WHEN** minute N is simulated
- **THEN** due scheduled substitutions are applied first, then due conditional orders
- **AND** fatigue-adjusted player copies are rebuilt before qualification and scoring read them
- **AND** cards and injuries are rolled after the scoring stage

#### Scenario: Added minutes play like any other
- **WHEN** a match has 2 and 4 added minutes
- **THEN** 96 minutes are simulated in the order 1…45, 45+1, 45+2, 46…90, 90+1…90+4
- **AND** chances, set pieces, cards, injuries, injury replacements and the forfeit check apply in every added minute

#### Scenario: Scheduled substitutions in added time
- **WHEN** a first half is in its added minutes
- **THEN** no scheduled substitution fires, because scheduled substitutions only have minutes 1 to 90; one scheduled for minute 46 fires at 46

### Requirement: In-match fatigue
Each minute, every player's seven core skills SHALL be scaled by `1 - P × (min(minutes played, 90) / 90) × ((99 - Stamina) / 99)`, where P is 0.30 for a Balanced side, 0.33 for an Attacking side and 0.27 for a Defensive side at that minute, and every later stage MUST read these effective skills. Added minutes count as minutes played.

#### Scenario: Stamina 99
- **WHEN** a Stamina-99 player has played 90 minutes
- **THEN** his effective skills equal his raw skills

#### Scenario: Stamina 0 at full time
- **WHEN** a Stamina-0 player of a Balanced side has played 90 minutes
- **THEN** each of his core skills is reduced by 30%

#### Scenario: Not yet on the pitch
- **WHEN** a player has played 0 minutes this match
- **THEN** he is unaffected whatever his Stamina

#### Scenario: Attacking side at full time
- **WHEN** a Stamina-0 player of an Attacking side has played 90 minutes
- **THEN** each of his core skills is reduced by 33%

#### Scenario: Deep into added time
- **WHEN** a Stamina-0 player of a Balanced side has played 95 minutes, added minutes included
- **THEN** each of his core skills is reduced by 30%, the same as at 90

### Requirement: Order firing
In the substitution stage of each minute, after scheduled substitutions, every unfired order whose from-minute is at most the current clock minute SHALL be checked, in slot order, against the score at the start of that minute. An order whose condition holds fires once and is never checked again. Orders are also checked in added minutes. An order that never meets its condition by the last second-half added minute MUST NOT fire.

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

#### Scenario: Going behind in added time
- **WHEN** an order "from 80, if losing, go Attacking" exists and the side goes behind at 90+1 of a match with 4 added minutes
- **THEN** the order fires at 90+2 and the side plays Attacking until 90+4

#### Scenario: From-minute in the first-half added time
- **WHEN** an order has from-minute 45 and its condition first holds at 45+2
- **THEN** it fires at 45+2

### Requirement: Stored match record for every competition
Every simulated match SHALL store the same record, whatever its competition (league match, cup tie or friendly): its events, its per-minute zone instants (one per simulated minute, added minutes included), its added time per half and its report flavor. The report flavor is the weather, attendance, referee and two assistant referees. Flavor generation is best-effort: if a name can't be drawn, the flavor is skipped and the match result stands.

#### Scenario: Cup tie simulated
- **WHEN** a cup tie finishes simulating with 2 first-half and 5 second-half added minutes
- **THEN** it has 97 stored instants (1 to 45, 45+1 and 45+2, 46 to 90, 90+1 to 90+5) and a stored weather, attendance, referee and two assistants

#### Scenario: Friendly simulated
- **WHEN** a friendly (club, national-team or World Cup group) finishes simulating
- **THEN** its stored weather and attendance are the same values that set its ticket revenue

#### Scenario: Forfeit
- **WHEN** a cup tie with 3 first-half added minutes stops in minute 61 because one side dropped to 7 players
- **THEN** it stores the instants for minutes 1 to 45, 45+1 to 45+3 and 46 to 61, a second-half added time of 0, and its flavor, the same as a league match does

#### Scenario: Referee names unavailable
- **WHEN** drawing a referee name fails after a cup tie was simulated
- **THEN** the tie's result, events and instants are still stored, and its flavor stays empty

## ADDED Requirements

### Requirement: Cup tie level after added time
A cup tie that is level after the last second-half added minute SHALL go to the existing penalty shoot-out. There is no extra time.

#### Scenario: Equaliser in added time
- **WHEN** a cup tie's away side equalises at 90+3 and no other goal follows
- **THEN** the tie goes to the shoot-out, stored as `went_to_penalties = true`
