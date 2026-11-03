# rivalries Specification

## Purpose
Gives two clubs of the same country a rivalry that grows out of their shared history (close
results, thrashings, red cards, cup eliminations, title races and key-player sales), fades over
time, and raises the stakes of their league matches and cup ties.

## Requirements

### Requirement: Rivalry score
Every pair of clubs of the same country SHALL have one rivalry score, from 0 to 40, starting at 0 and shared by both sides. National teams have no rivalry. A score is changed only by the events in "Rivalry events" and by "Rivalry decay", and it MUST never exceed 40 or drop below 0.

#### Scenario: Two clubs that never met
- **WHEN** two clubs of the same country have never played each other
- **THEN** their rivalry score is 0 and neither lists the other as a rival

#### Scenario: Cap
- **WHEN** a pair at 39 earns 5 more points
- **THEN** its score is 40

#### Scenario: Clubs of different countries
- **WHEN** two clubs of different countries meet in an international cup
- **THEN** no rivalry score is created or changed

### Requirement: Rivalry events
A league match or cup tie between two clubs SHALL add points to their score: +1 for a draw or a one-goal margin, +1.5 for a margin of 3 or more, +1 per red card shown in the match (at most +2), and +3 to the pair for a cup tie that eliminates one of them (+5 for a final). Friendlies MUST add nothing.

#### Scenario: Tight league win with a red card
- **WHEN** a league match ends 2–1 and one player is sent off
- **THEN** the pair's score rises by 1 + 1 = 2

#### Scenario: Thrashing
- **WHEN** a league match ends 4–0 with no red cards
- **THEN** the pair's score rises by 1.5

#### Scenario: Comfortable win
- **WHEN** a league match ends 2–0 with no red cards
- **THEN** the pair's score does not change

#### Scenario: Cup final decided on penalties
- **WHEN** a cup final ends 1–1 and one side wins the shoot-out
- **THEN** the pair's score rises by 1 (draw) + 5 (final) = 6

#### Scenario: Friendly
- **WHEN** two clubs play a friendly
- **THEN** their score does not change

### Requirement: Season and transfer rivalry events
At each season rollover the champion and the runner-up of every division SHALL add 4 to their pair's score. When a club sells a player who was one of its key players to another club of the same country (listing purchase or accepted offer), the pair SHALL add 3. Loans add nothing.

#### Scenario: Title race
- **WHEN** a division's season ends with A champion and B runner-up
- **THEN** the A–B score rises by 4 before the season's decay is applied

#### Scenario: Key player sold to a domestic club
- **WHEN** a club sells one of its key players to a club of the same country
- **THEN** the pair's score rises by 3

### Requirement: Human rivalry bonus
Every point a pair earns from "Rivalry events" or "Season and transfer rivalry events" SHALL be multiplied by 1.5 when both clubs had a human owner when the event happened.

#### Scenario: Two human managers in a cup final
- **WHEN** two human-owned clubs play a cup final that ends 2–1
- **THEN** the pair's score rises by (1 + 5) × 1.5 = 9

### Requirement: Rivalry decay
At each season rollover of a country, after the season's events are added, every pair score of that country SHALL be multiplied by 0.6. A pair whose score falls below 0.5 MUST be reset to 0.

#### Scenario: Old feud
- **WHEN** a pair at 20 earns nothing during a season
- **THEN** it is at 12 after the rollover and at 7.2 after the next one

### Requirement: Rivalry levels
A pair SHALL be at level Rival when its score is at least 8 and below 16, and at level Clásico when it is at least 16. Below 8 it has no level. A match's level is the level of its pair when the match kicks off, and it MUST NOT change during or after that match.

#### Scenario: Thresholds
- **WHEN** a pair's score is 7.9, 8, 15.9 and 16
- **THEN** its level is none, Rival, Rival and Clásico respectively

#### Scenario: Level fixed at kickoff
- **WHEN** a match between two clubs at 15 ends with 2 red cards and a one-goal margin
- **THEN** the match itself is played and shown as a Rival match, and the pair is a Clásico from the next match on

### Requirement: Rivalry notification
When a pair's level rises (none to Rival, or Rival to Clásico) each human owner of the two clubs SHALL get one `NOTIFICATION_RIVALRY` naming the other club and the new level. A level that falls is not notified. A rise caused by a match MUST be notified only after that match's live replay finishes.

#### Scenario: New Clásico
- **WHEN** a cup final takes a human club's pair from 13 to 19
- **THEN** after the final's replay ends that owner is told the other club is now a Clásico rival

#### Scenario: Decay
- **WHEN** a season rollover drops a pair from Clásico to Rival
- **THEN** no notification is sent

### Requirement: Rivalry visibility
A team's page SHALL list its top 3 pairs with a level (highest score first) with the other club, the level, the score as a 0–40 bar and a link to their head-to-head. Fixture rows, the dashboard's next-match hero and the match viewer scoreboard SHALL show a Rival or Clásico chip for a match with a level. A match's chip MUST NOT reveal a level change it caused before its replay ends.

#### Scenario: Team page
- **WHEN** a manager opens a club with pairs at 22, 9 and 5
- **THEN** its Rivals section lists the Clásico at 22 and the Rival at 9, and not the pair at 5

#### Scenario: Upcoming Clásico
- **WHEN** a manager's next league match is against a Clásico rival
- **THEN** the dashboard hero and the fixture row show a "Clásico" chip

## Code map

- Storage: `backend/resources/sql/init100_team_rivalries.sql` (`team_rivalries`, `rivalry_level` on
  `matches` and `cup_ties`).
- Core: `common/rivalries.go` (`RIVALRY_*`, thresholds and point constants, `RivalryLevel`,
  `DecayRivalryScore`, `AddRivalryPoints` - row created then locked `FOR UPDATE`, so concurrent
  additions are exact -, `AddRivalryEvent` / `AddRivalryEventQuietly` with the human bonus and the
  same-country, club-only check `rivalryPair`, `NotifyRivalryRise`, `GetRivalryLevel`,
  `KickoffRivalryLevel`, `GetTopRivalries`, `DecayCountryRivalries`, `IsCupFinal`,
  `fixtureRivalrySql` / `GetFixtureRivalry`).
- Match events: `engine/rivalries.go` (`rivalryPointsForMatch`, `applyRivalryForMatch`, called from
  `applyPostMatchEffects` in `engine/postmatch.go`, so after the replay); `PostMatchPayload.CupFinal`
  set by `simulateCupTie`.
- Season: `applyTitleRaceRivalries` (`engine/season.go`), from `applyPromotionRelegation` under the
  rollover lock: title race, then the decay, then the rise notifications that survived it.
- Transfers: `isKeyPlayerOf` judged before the move in `ExecuteTransfer` (`common/transfers.go`) and
  `executeOfferTx` (`common/offers.go`), applied after the commit (`finishOfferTransfer`).
- Effects: `cardIncidentProbability` (`engine/discipline_simulation.go`), `rivalryAttendanceFactor`
  (`engine/stadium.go`), `fanbaseResultDriftProbability` (`engine/fanbase.go`).
- Notification: `NOTIFICATION_RIVALRY`, keys `notifications.messages.rivalryRival` /
  `rivalryClasico`, Spanish templates in `common/notificationtranslations.go`.
- API: `GET /v1/teams/team/:id/rivalries` (`handlers/rivalryHandlers.go`); `rivalry` on
  `TeamMatchSummary`, `WeekMatch` and `CupTieDetails`.
- Frontend: `src/components/RivalryChip.tsx` (fixture rows in `TeamMatchesSection`, the dashboard's
  `NextMatchHero`, the viewer's `Scoreboard`), `src/features/team/RivalsSection.tsx` (team page),
  hook `useTeamRivalries`; strings `rivalries.*`.
- Tests: `common/rivalries_test.go`, `common/rivalries_integration_test.go`,
  `engine/rivalries_test.go` (with the calibration), `engine/rivalries_integration_test.go`,
  `handlers/rivalryHandlers_test.go`, `src/features/team/RivalsSection.test.tsx`.
