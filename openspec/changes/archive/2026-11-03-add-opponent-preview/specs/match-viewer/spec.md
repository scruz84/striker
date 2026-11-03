# Spec Delta

## MODIFIED Requirements

### Requirement: Viewer phases
The viewer SHALL show one of three views:
- **Scheduled**, before kickoff: the scoreboard marked as scheduled, plus the lineup section and, for a participating owner before the lineup lock, the opponent preview (see opponent-preview).
- **Live**, from kickoff until the live replay reports completion: the live scoreboard, lineups pitch, chronicle, zone duel, minute-by-minute ratings and head-to-head card.
- **Finished**: the scoreboard and the three tabs.

#### Scenario: Opening a match during its broadcast
- **WHEN** a viewer opens a friendly 20 minutes after kickoff while its replay is still running
- **THEN** the live view is shown, caught up to minute 20, and it switches to the finished view only when the replay completes

#### Scenario: Played flag set before kickoff
- **WHEN** a match is already marked played 10 minutes before its advertised kickoff
- **THEN** the viewer still shows the scheduled view until kickoff

#### Scenario: Participant before kickoff
- **WHEN** a participating owner opens his upcoming league match before the lineup lock
- **THEN** the scheduled view shows his lineup editor and the opponent preview
