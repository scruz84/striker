# Spec Delta

## ADDED Requirements

### Requirement: Minute labels with added time
Every minute the viewer shows (the live clock, goal list, chronicle, events report and scrubber labels) SHALL read `N'` for a regulation minute and `45+N'` or `90+N'` for an added minute, in every competition and in both the live and finished views.

#### Scenario: Goal in first-half added time
- **WHEN** a goal is scored at 45+2
- **THEN** the scoreboard's goal list reads `45+2'` next to the scorer, with `(pen.)` or `(f.k.)` after it for a set piece

#### Scenario: Live clock in added time
- **WHEN** the replay is at the third second-half added minute
- **THEN** the live clock reads `90+3'`

#### Scenario: Old match
- **WHEN** a match played before this change is opened
- **THEN** every minute reads as before, with no added-time label

### Requirement: Added-time board in the chronicle
The live chronicle and the finished report SHALL list every `ADDED_TIME` event at minute 45 or 90 as the fourth official's board ("+N minutes" / "+N minutos" in the viewer's language). During the live view it MUST appear only once the replay reaches that minute.

#### Scenario: Board shown at minute 90
- **WHEN** a viewer watches a match whose second half gets 5 added minutes
- **THEN** at minute 90 the chronicle shows "+5 minutes", after any other event of minute 90

#### Scenario: Before minute 90
- **WHEN** the replay is at minute 88
- **THEN** the second-half board is not shown, so the viewer cannot yet tell how long the match will last

### Requirement: Scrubbers cover added minutes
The minute scrubbers of the zone duel and the ratings by minute SHALL have one step per simulated minute, added minutes included, in match order. The last step is the final minute, so the finished view's last step shows the stored ratings.

#### Scenario: Scrubbing into first-half added time
- **WHEN** a finished match has 2 first-half added minutes and the viewer steps forward from 45
- **THEN** the next steps are 45+1, 45+2, then 46

#### Scenario: Final minute
- **WHEN** a finished match has 4 second-half added minutes
- **THEN** the scrubber's last step is labelled `90+4'` and shows the stored ratings
