# Spec Delta

## ADDED Requirements

### Requirement: Mentality change in the chronicle
The live chronicle and the finished report SHALL list every `MENTALITY_CHANGE` event at its minute, naming the team and its new mentality in the viewer's language. During the live view it MUST appear only once the replay reaches its minute.

#### Scenario: Live replay
- **WHEN** a viewer watches a match whose away side switches to Attacking in minute 70
- **THEN** at minute 70 the chronicle shows that the away side moves to an attacking approach

#### Scenario: Before the minute
- **WHEN** the replay is at minute 50
- **THEN** that minute-70 change is not shown
