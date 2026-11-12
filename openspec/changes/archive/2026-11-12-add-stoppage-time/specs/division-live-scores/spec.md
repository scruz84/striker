# Spec Delta

## MODIFIED Requirements

### Requirement: Live score entry content
Each entry SHALL carry `matchId`, `weekNumber`, `matchDate`, the home and away team ids and names, `status`, `minute`, `addedMinute`, `homeGoals` and `awayGoals`. `status` is one of `pending`, `live`, `halftime` or `finished`. `addedMinute` is 0 outside added time.

#### Scenario: Entry shape
- **WHEN** a match of matchday 12 is at minute 63 with a 1-0 scoreline
- **THEN** its entry has `weekNumber` 12, `status` "live", `minute` 63, `addedMinute` 0, `homeGoals` 1 and `awayGoals` 0

#### Scenario: Entry in added time
- **WHEN** a match is at 90+2 with a 2-2 scoreline
- **THEN** its entry has `status` "live", `minute` 90, `addedMinute` 2, 2-2, and the client labels it `90+2'`

### Requirement: Scores come only from the broadcast
`homeGoals`, `awayGoals`, `minute` and `addedMinute` SHALL be what that match's live broadcast is showing at the time of the request, never the stored final result, so the endpoint reveals nothing a viewer of that match's live page would not already see.

#### Scenario: Final result already stored
- **WHEN** a match was simulated as 3-2, and its broadcast is at minute 63 showing 1-2
- **THEN** its entry reads `minute` 63, 1-2, `status` "live"

#### Scenario: Broadcast finished
- **WHEN** the broadcast has published its final whistle after 90+4 with a 3-2 result, 1 hour 47 minutes after kickoff
- **THEN** the entry reads `status` "finished", `minute` 90, `addedMinute` 4, 3-2

#### Scenario: Winner scored in added time
- **WHEN** a match stored as 2-1 has its winner at 90+3 and its broadcast is at 90+1 showing 1-1
- **THEN** the entry reads `status` "live", 90+1, 1-1, and not 2-1

### Requirement: Live score statuses
`pending` SHALL mean kickoff has not arrived or the match is not simulated yet (minute 0, 0-0). `halftime` SHALL mean the broadcast is in its half-time pause (minute 45 plus the first-half added time, score at the end of the first half). `finished` SHALL mean the broadcast has reported full time. Any other match SHALL be `live`, added minutes included.

#### Scenario: Simulated ahead of kickoff
- **WHEN** a match was simulated 10 minutes before its kickoff, which is still 5 minutes away
- **THEN** its entry is `pending`, minute 0, 0-0

#### Scenario: Engine running late
- **WHEN** a match's kickoff was 3 minutes ago but it has not been simulated yet
- **THEN** its entry is `pending`, minute 0, 0-0

#### Scenario: Half-time
- **WHEN** the broadcast is in its half-time pause after 2 first-half added minutes with a 0-1 scoreline
- **THEN** the entry is `halftime`, minute 45, `addedMinute` 2, 0-1

### Requirement: Estimate before the first broadcast message
When a simulated match's kickoff has passed but its broadcast has not published a minute yet, or the broadcast state cannot be read, the entry SHALL be `live` with the minute estimated from the time since kickoff and the match's stored added time, on the replay's publish schedule (see match-scheduling). The score SHALL be the one at that minute. The estimate MUST NOT report `finished`.

#### Scenario: Broadcast state unreadable
- **WHEN** the live broadcast store is unreachable and a match kicked off 20 real minutes ago with a goal at minute 12
- **THEN** its entry is `live`, minute 21, with that goal counted

#### Scenario: Estimate in first-half added time
- **WHEN** the broadcast store is unreachable and a match with 3 first-half added minutes kicked off 46 real minutes ago
- **THEN** its entry is `live`, minute 45, `addedMinute` 2

#### Scenario: Estimate past the last added minute
- **WHEN** the broadcast store is unreachable and a match with 4 second-half added minutes kicked off 115 real minutes ago
- **THEN** its entry is `live`, 90+4, with the score as of 90+4, until the live-broadcast window closes
