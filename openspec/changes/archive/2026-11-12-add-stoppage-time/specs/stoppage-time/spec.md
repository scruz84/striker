# Spec Delta

## Purpose

How much time is added at the end of each half of a match, how it is worked out from the
time lost to stoppages, how it is announced and recorded, and the bound that keeps the
longest broadcast inside the live-broadcast window.

## ADDED Requirements

### Requirement: Time lost per half
At the end of minute 45 and of minute 90, the engine SHALL add up the time lost in that half's regulation minutes (1–45 or 46–90): a base of 30 s (first half) or 60 s (second half), 40 s per goal, 30 s per substitution (injury replacements included), 60 s per injury, 15 s per yellow card, 40 s per red card and 45 s per penalty awarded. Mentality changes cost nothing.

#### Scenario: Quiet first half
- **WHEN** a first half has no goal, card, injury, substitution or penalty
- **THEN** the time lost is 30 s

#### Scenario: Busy second half
- **WHEN** a second half has 2 goals, 5 substitutions, 1 yellow card and 1 injury replaced by a bench player
- **THEN** the time lost is 60 + 2 × 40 + 6 × 30 + 15 + 60 = 395 s, because the injury replacement is a sixth substitution

#### Scenario: Penalty scored
- **WHEN** a penalty is awarded and scored in minute 30
- **THEN** it counts 45 s for the penalty and 40 s for the goal

#### Scenario: Second yellow
- **WHEN** a player is shown a second yellow and sent off
- **THEN** it counts as one red card, 40 s

### Requirement: Added minutes per half
The added time of a half SHALL be the time lost divided by 60 and rounded half up to whole minutes. It is then clamped to 1–5 minutes, in either half.

#### Scenario: Rounding
- **WHEN** a second half lost 250 s
- **THEN** 250 / 60 = 4.17 rounds to 4 added minutes

#### Scenario: Minimum
- **WHEN** a first half lost 30 s
- **THEN** 0.5 rounds to 1 added minute

#### Scenario: First-half clamp
- **WHEN** a first half lost 330 s
- **THEN** 5.5 rounds to 6, clamped to 5 added minutes

#### Scenario: Second-half clamp
- **WHEN** a second half lost 395 s
- **THEN** 6.58 rounds to 7, clamped to 5 added minutes

### Requirement: Added time is computed from regulation only
Events that happen during a half's added minutes MUST NOT change that half's added time, which is fixed when its regulation minutes end.

#### Scenario: Goal in added time
- **WHEN** a goal is scored at 90+2 of a second half with 4 added minutes
- **THEN** the match still ends after 90+4

### Requirement: Added-time board event
When a half's added time is fixed, the engine SHALL record an `ADDED_TIME` event (type 9) at minute 45 or 90, added minute 0, carrying the number of added minutes and naming the home side as team owner and the away side as other team, like the default-result event. It comes after every other event of that minute.

#### Scenario: Board at the end of the first half
- **WHEN** a first half gets 2 added minutes
- **THEN** an `ADDED_TIME` event for minute 45 carrying 2 is stored, after any goal or card of minute 45

#### Scenario: Forfeit before the board
- **WHEN** a match is abandoned in minute 61 because a side dropped to 7 players
- **THEN** it has the first half's `ADDED_TIME` event and no second-half one

### Requirement: Added time stored on the fixture
Every simulated league match, cup tie and friendly SHALL store the added minutes of its first and its second half. A half that was never reached (forfeit), a defaulted fixture, and any match played before this change store 0.

#### Scenario: Cup tie
- **WHEN** a cup tie gets 3 and 5 added minutes
- **THEN** its fixture stores first-half added time 3 and second-half added time 5

#### Scenario: Old match
- **WHEN** a match played before this change is read
- **THEN** both added times are 0 and it has no `ADDED_TIME` event

### Requirement: Added minutes notation
A minute in added time SHALL be identified by its clock minute (45 or 90) plus an added minute from 1 to that half's added time, written `45+N` / `90+N`. Ordering is by clock minute, then added minute. A regulation minute has added minute 0.

#### Scenario: Ordering across halves
- **WHEN** events happen at 44, 45+2 and 46
- **THEN** they are ordered 44, 45+2, 46

#### Scenario: Counted as first half
- **WHEN** a goal is scored at 45+1
- **THEN** it counts as a first-half goal everywhere a half is distinguished, for example the half-time score

### Requirement: Longest broadcast fits the live-broadcast window
The longest possible live replay, from kickoff to the final whistle with both halves at their maximum added time, MUST end at least 5 minutes before the live-broadcast window (2 hours after kickoff) closes. Any change to the added-time clamps, the replay pacing or the window MUST keep this true.

#### Scenario: Current constants
- **WHEN** the second half has the maximum 5 added minutes
- **THEN** 90+5 and the final whistle are published 108 minutes after kickoff, 12 minutes inside the 120-minute window

#### Scenario: Clamp raised too far
- **WHEN** a change raises the second-half clamp to 13 minutes
- **THEN** the bound check fails, because the final whistle would fall 116 minutes after kickoff, less than 5 minutes inside the window

### Requirement: Results stay hidden until the final whistle
No result, standings change, scorer tally, head-to-head entry, cup or World Cup progression, notification, injury, suspension or skill change of a match SHALL be visible outside its own match page before its replay's final whistle. A gate based on time MUST use the full live-broadcast window and never assume the match ends at minute 90.

#### Scenario: Goal at 90+5
- **WHEN** a league match with 5 added minutes is won by a goal at 90+5, published 108 minutes after kickoff
- **THEN** the league table, the dashboard and the head-to-head do not show that result before the final whistle that follows it
- **AND** the result notification is sent only after the final whistle

#### Scenario: Live table during added time
- **WHEN** a match is at 90+2 on its broadcast
- **THEN** the division live scores show it `live`, not `finished`
