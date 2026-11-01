# Spec Delta

## MODIFIED Requirements

### Requirement: World Cup display
`GET /v1/national-teams/world-cup/current` SHALL return the latest World Cup with its groups' standings and fixtures; its knockout bracket is read through `GET /v1/cups/:id` and rendered with the cup page's components. Group tables MUST show played, won, drawn, lost, goals for, goals against, goal difference and points, and fixtures MUST be grouped by matchday with their kickoff date.

#### Scenario: Knockout under way
- **WHEN** a user opens the World Cup tab during the knockout stage
- **THEN** they see the final group tables and the bracket

#### Scenario: Group fixtures by matchday
- **WHEN** a user opens a group card during the group stage
- **THEN** its fixtures are listed under "Jornada 1", "Jornada 2", "Jornada 3", each with its date

## ADDED Requirements

### Requirement: Broadcast-safe group standings
World Cup group standings SHALL count only group matches whose broadcast has finished (kickoff at least 2 hours ago). A group match still inside its broadcast window MUST be returned with status `live`, the current minute and the score as of that minute, never its final score.

#### Scenario: Group match being broadcast
- **WHEN** Spain beat Portugal 2-0 in a group match simulated 20 minutes ago and the broadcast is at minute 18 with 0-0
- **THEN** the group table shows neither team's played count, points or goals changed by that match
- **AND** the fixture shows `live`, minute 18, 0 - 0

#### Scenario: Broadcast finished
- **WHEN** that match kicked off 2 hours ago
- **THEN** the table includes it: Spain +3 points, +2 goal difference

### Requirement: World Cup history
`GET /v1/national-teams/world-cups` SHALL list every World Cup, newest first, with its name, season, stage and champion (empty until completed). The Mundial tab MUST show this list as a roll of honour.

#### Scenario: Third World Cup running
- **WHEN** two World Cups have been completed and a third is in its group stage
- **THEN** the list shows the third as in progress and the first two with their champions

### Requirement: Next World Cup estimate
When no World Cup is running, the current World Cup response SHALL include the estimated start date of the next one: the last international cup check plus 60 days times the number of international cycles until the counter next reaches a multiple of 4.

#### Scenario: Two cycles to go
- **WHEN** no World Cup is active, the international cup counter is 6 and its last check was 1 October
- **THEN** the estimate is 1 October + 120 days, shown as "Next World Cup: around 29 January"

#### Scenario: Too few countries
- **WHEN** fewer than 8 countries are active
- **THEN** no estimate is returned
