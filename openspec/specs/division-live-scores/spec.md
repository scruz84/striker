# division-live-scores Specification

## Purpose
Live scores of every league match of one division while its matchday is being broadcast, as the
broadcast shows them and never as the stored final result, for the views that follow the rest of
a matchday (the dashboard's live hero and the league page).

## Requirements

### Requirement: Division live scores endpoint
`GET /v1/teams/division/:id/live` SHALL return the league matches of that division whose kickoff is between 2 hours ago and 15 minutes from now, to any signed-in user, ordered by kickoff then match id. A non-numeric id SHALL return 400. A league with no match in that window, or an unknown league, SHALL return 200 with `[]`.

#### Scenario: Matchday being broadcast
- **WHEN** a 16-team league's 8 matches of matchday 12 kicked off 40 minutes ago and a client requests `/v1/teams/division/7/live`
- **THEN** the response is 200 with those 8 matches

#### Scenario: No matchday in play
- **WHEN** the league's last kickoff was 3 hours ago and the next one is in 2 days
- **THEN** the response is 200 with `[]`

#### Scenario: Invalid id
- **WHEN** a client requests `/v1/teams/division/abc/live`
- **THEN** the response is 400

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

### Requirement: Live scores cache
The response SHALL be cached per league for at most 5 seconds, so two requests for the same league within 5 seconds read the broadcast state at most once.

#### Scenario: Many viewers of one league
- **WHEN** 200 users of the same league poll the endpoint within the same 5 seconds
- **THEN** the broadcast state of that league's matches is read once and every user gets the same response

### Requirement: Client polling
A view showing division live scores SHALL refresh them every 15 seconds while at least one entry is not `finished`, and SHALL stop when every entry is `finished` or the list is empty. If a request fails, the view SHALL fall back to what it shows without live scores, and SHALL NOT show a score from any other source.

#### Scenario: Matchday ends
- **WHEN** the last broadcast of the matchday reports full time
- **THEN** the next response has every entry `finished` and the view stops polling

#### Scenario: Endpoint down
- **WHEN** the live scores request fails
- **THEN** the view shows no live scores and keeps any "LIVE" label it already had

## Code map

- Backend: `backend/common/livematches.go` (`GetLeagueLiveScores`, `liveStateAsOfNow`, `LiveState` and the
  `LiveStatus*` constants, `getLeagueLiveMatchesStmtSql`, `liveEarlySimulationWindow`), `backend/common/cache.go`
  (`SetValueFor`), `backend/handlers/statsHandlers.go` (`LeagueLiveScoresHandler`) and the route
  `GET /v1/teams/division/:id/live` in `backend/service/fmservice.go`.
- Frontend: `frontend/src/api/hooks.ts` (`useDivisionLiveScores`, `divisionLiveScoresRefetchInterval`),
  `frontend/src/api/types.ts` (`LeagueLiveScore`), `frontend/src/components/DivisionLiveScores.tsx`,
  `frontend/src/utils/liveScoreLabel.ts`; strings under `liveScores.*` in `i18n/locales/{en,es}.json`.
- Tests: `backend/common/league_live_scores_integration_test.go`, `backend/common/livematches_test.go`,
  `backend/handlers/statsHandlers_test.go`, `frontend/src/api/divisionLiveScores.test.tsx`,
  `frontend/src/components/DivisionLiveScores.test.tsx`, `frontend/src/utils/liveScoreLabel.test.ts`.
- Added time: `LeagueLiveScore.AddedMinute` and `LiveState.AddedMinute`; `minuteFromLiveMessage` reads it
  from the message (the "halftime" / "complete" `Minute` / `AddedMinute`), and the estimate is
  `CurrentMatchMinute(kickoff, firstHalfAddedTime, secondHalfAddedTime)` (`common/livebroadcast.go`)
  over the fixture's stored added time (`get*AddedTimeStmtSql`) with the score as of
  `(minute, added_minute)` (`get*ScoreAsOfMinuteStmtSql`). The same state feeds the dashboard's
  suggested live match (`LiveMatchSummary.AddedMinute`) and the national-team feeds
  (`NationalMatch.AddedMinute`, `liveScoreLookup`). Frontend: `utils/liveScoreLabel.ts` labels it with
  `matchMinuteText`. Tests: `TestGetLeagueLiveScores_AddedTime`,
  `TestGetLeagueLiveScores_EstimateWithAddedTime`, `TestCurrentMatchMinute`,
  `TestMinuteFromLiveMessage`, `TestLeagueLiveScore_JsonCarriesAddedMinute`.
