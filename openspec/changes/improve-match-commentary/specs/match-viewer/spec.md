# Spec Delta

## ADDED Requirements

### Requirement: Half-time line in the chronicle
The live chronicle and the finished report SHALL show one half-time line after the last first-half event (first-half added time included), with the score at the break: a goalless, level, home-leads or away-leads line, 3 or more per case. Live, it MUST appear only once the broadcast is in the break or past minute 45. A match abandoned in the first half has none.

#### Scenario: Finished match
- **WHEN** a viewer opens a finished match that was 1-0 at half time
- **THEN** a home-leads half-time line with "1-0" sits between the last first-half event and the first second-half event

#### Scenario: Live before the break
- **WHEN** the replay is at 45+1 and the half is not over
- **THEN** no half-time line is shown

#### Scenario: Live during the break
- **WHEN** the broadcast is paused for half time
- **THEN** the half-time line is shown with the score at the break

#### Scenario: Abandoned early
- **WHEN** a match was abandoned in minute 30
- **THEN** the chronicle has no half-time line
