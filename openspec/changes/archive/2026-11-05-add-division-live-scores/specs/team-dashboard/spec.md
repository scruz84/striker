# Spec Delta

## ADDED Requirements

### Requirement: Rest of the matchday in the live hero
When the live hero shows a league match, it SHALL include a "Rest of the matchday" section listing the division live scores entries of the same `weekNumber`, excluding the owner's own match, in kickoff then match id order. Each row SHALL show both teams, the score and the state ("63'", "Half-time", "FT", or the kickoff time while `pending`) and link to `/match/:id`.

#### Scenario: League match live
- **WHEN** the owner's league match of matchday 12 is live and the other 7 matches of matchday 12 are broadcasting
- **THEN** the hero lists those 7 matches with their broadcast scores and minutes, and not the owner's own match

#### Scenario: Score as broadcast
- **WHEN** another match of the matchday is at minute 63 with 1-2 and its stored result is 3-2
- **THEN** its row reads "1 - 2" and "63'", never "3 - 2"

#### Scenario: Cup tie live
- **WHEN** the owner's live match is a cup tie or a friendly
- **THEN** the hero shows no "Rest of the matchday" section and makes no live scores request

### Requirement: Rest of the matchday when there is nothing to list
The section SHALL be hidden when the division live scores list no other match of the same matchday, while the first response is loading, and when the request fails. The owner's own score, minute and "Watch live" button SHALL never depend on it.

#### Scenario: Endpoint down
- **WHEN** the division live scores request fails while the owner's league match is live
- **THEN** the hero shows the own match as before and no "Rest of the matchday" section

### Requirement: Rest of the matchday layout
On phone-width screens the section SHALL start collapsed behind a toggle that reads the summary "N matches · G goals", N being the listed matches and G the sum of their broadcast goals. On desktop it SHALL start expanded. Rows MUST fit at 360 px wide without horizontal scroll and stay readable on the hero's error-container surface.

#### Scenario: Phone summary
- **WHEN** an owner on a 360 px phone has a live league match and the other 7 matches have 12 goals between them
- **THEN** the hero shows a collapsed toggle reading "7 matches · 12 goals", and tapping it lists the 7 matches

#### Scenario: Desktop
- **WHEN** the same owner opens the dashboard on a 1280 px screen
- **THEN** the 7 matches are listed without tapping anything

#### Scenario: Spanish summary
- **WHEN** the language is Spanish and there is 1 other match with 0 goals
- **THEN** the toggle reads "1 partido · 0 goles"
