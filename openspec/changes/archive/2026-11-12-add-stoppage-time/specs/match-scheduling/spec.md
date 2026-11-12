# Spec Delta

## MODIFIED Requirements

### Requirement: Live replay of a simulated result
The engine SHALL replay each simulated match over a WebSocket channel at one simulated minute per real minute, added minutes included, never publishing minute 1 before the advertised kickoff, so a viewer sees it unfold rather than a finished score. The half-time break SHALL last 15 minutes minus the first-half added time, so minute 46 is always published 59 minutes after kickoff.

#### Scenario: Viewer opens the match mid-way
- **WHEN** a viewer connects 30 minutes after kickoff
- **THEN** they see the state as of minute 30 and the remaining minutes as they are paced out
- **AND** the starting lineups are published once before the first minute

#### Scenario: Publish schedule with added time
- **WHEN** a match that kicks off at 14:00 has 3 first-half and 5 second-half added minutes
- **THEN** minute 45 is published at 14:44 and the added-time board for 3 minutes comes with it, 45+1 to 45+3 at 14:45 to 14:47, and half time at 14:47
- **AND** minute 46 is published at 14:59, minute 90 at 15:43 with the board for 5 minutes, 90+1 to 90+5 at 15:44 to 15:48, and the final whistle right after 90+5

#### Scenario: Shortest break
- **WHEN** the first half has the maximum 5 added minutes
- **THEN** the half-time break lasts 10 minutes and minute 46 is still published 59 minutes after kickoff

#### Scenario: Late simulation catches up
- **WHEN** a match is simulated 50 minutes after its kickoff
- **THEN** every minute up to the one matching 50 real minutes after kickoff, added minutes included, is published at once, and the rest follow the schedule above

### Requirement: Spoiler-free side effects
Everything a team owner could see outside the match page MUST wait until the live replay publishes its final whistle, after the last second-half added minute: the result notification, new injuries and suspensions, and post-match skill growth.

#### Scenario: Match simulated before kickoff
- **WHEN** a league match is simulated 15 minutes before kickoff
- **THEN** no result notification, injury/suspension or skill change becomes visible until its replay finishes

#### Scenario: Injury in added time
- **WHEN** a player is injured at 90+3 of a match with 5 added minutes
- **THEN** the injury becomes visible outside the match page only after the final whistle that follows 90+5
