# Spec Delta

## ADDED Requirements

### Requirement: Team of the week
Once every match of a league matchday is played and past its replay, the league SHALL store a 4-4-2 of that matchday's rated players: 1 goalkeeper, 4 defenders, 4 midfielders and 2 forwards by the position each played most, the highest rating first, then more minutes, more goals, lower player id. A line short of players leaves its slots empty.

#### Scenario: Ties within a line
- **WHEN** three defenders are rated 7.0, one of them with 90 minutes and a goal and one with 90 minutes
- **THEN** the one with the goal comes first, then the other 90-minute one

#### Scenario: Matchday not over
- **WHEN** one match of the matchday is still waiting for its post-match effects
- **THEN** no team of the week is stored yet

### Requirement: Picked once
A team of the week or of the season SHALL be stored exactly once, however many engine replicas close the matchday at the same time, and only the one that stores it notifies.

#### Scenario: Four replicas
- **WHEN** four replicas apply the last match's effects and close the matchday at once
- **THEN** one round of at most 11 players is stored and each owner is notified once

### Requirement: Team of the season
At rollover, alongside the season awards, each league SHALL store a 4-4-2 by average rating of its league matches, among players with at least 36 minutes per match of the team that played the most, each on the line he played most (the lower line on a tie).

#### Scenario: Not enough minutes
- **WHEN** a forward averaged 9.0 over 20 minutes in a 1-match season
- **THEN** he is not in the team of the season

### Requirement: Best XI notification
Each club owner whose players made a team of the week or of the season SHALL get one `NOTIFICATION_AWARD` naming them (`notifications.messages.teamOfTheWeek` or `teamOfTheSeason`), linking to that round on the division page.

#### Scenario: Two players picked
- **WHEN** two players of a user's club make the matchday 5 team of the week
- **THEN** the user gets one notification "Team of the week (matchday 5): A, B" linking to `/division/<id>?xi=5`

### Requirement: Best XI on the league page
The division page SHALL show the latest team of the week on a pitch with each player's rating and club, with arrows through the season's matchdays and its team of the season, and an empty state before the first matchday ends. A past season opens on its team of the season. A player's profile SHALL count his teams of the week and of the season.

#### Scenario: Open from a notification
- **WHEN** a user opens `/division/7?xi=5`
- **THEN** the page opens on the team of matchday 5
