# world-cup Specification

## Purpose

The national-team World Cup: when one starts, how its group stage is drawn and played as linked
friendlies, how qualifiers move into a knockout cup that reuses the club-cup machinery, and how
it is shown.

## Requirements

### Requirement: World Cup cadence
A new World Cup SHALL start when the international cup counter (advanced every 60 days by the daily 03:00 UTC international cup job) reaches a multiple of 4, no World Cup is running, and at least 8 countries are active. Every active country enters.

#### Scenario: Fourth international cycle
- **WHEN** the international cup counter reaches 8 and no World Cup is active with 12 active countries
- **THEN** a World Cup starts with all 12

#### Scenario: Too few countries
- **WHEN** only 6 countries are active
- **THEN** no World Cup starts

#### Scenario: Several engine replicas
- **WHEN** the international cup job fires on several replicas at once with a round due
- **THEN** exactly one claims the round, so at most one World Cup starts

### Requirement: Clearing pending friendlies
Before entering countries into a new World Cup the engine MUST cancel every unplayed national-team friendly involving them, together with any saved lineup.

#### Scenario: Friendly already scheduled
- **WHEN** a country has a pending weekly friendly as the World Cup starts
- **THEN** that friendly is deleted so the country is not double-booked

### Requirement: Group stage
Countries SHALL be split into balanced groups of about 4 (sizes differing by at most one), each playing a full round-robin, one matchday a week from the nearest Saturday. Every group match MUST be an ordinary friendly row linked to the group.

#### Scenario: Group match played
- **WHEN** a linked group friendly is simulated
- **THEN** both teams' group record (played, won, drawn, lost, goals) is updated

#### Scenario: Viewing a group match
- **WHEN** a user opens a group match
- **THEN** it renders through the shared match viewer (see match-viewer), labelled with its group and matchday
- **AND** the friendly detail response carries that group name and matchday, which are empty for a friendly not linked to a group

### Requirement: Group ranking
Groups SHALL be ranked by points (3 per win, 1 per draw), then goal difference, then goals for, then team name.

#### Scenario: Level on points and goal difference
- **WHEN** two teams have equal points and goal difference
- **THEN** the one with more goals for ranks higher

### Requirement: Knockout stage
Once every group match is played, the top two of each group SHALL be shuffled into a knockout cup (region `world`) run by the ordinary cup machinery, with each national side's lineup picked automatically per round. When that cup completes, its champion MUST be recorded on the World Cup.

#### Scenario: Group stage complete
- **WHEN** the last group match is played
- **THEN** the next cup job creates the knockout cup from the 2 best of every group
- **AND** only one knockout cup is created even when several engine replicas run that job at once

#### Scenario: Final played
- **WHEN** the knockout cup completes
- **THEN** the World Cup is marked completed with that champion

### Requirement: World Cup display
`GET /v1/national-teams/world-cup/current` SHALL return the latest World Cup with its groups' standings and fixtures; its knockout bracket is read through `GET /v1/cups/:id` and rendered with the cup page's components. Group tables MUST show played, won, drawn, lost, goals for, goals against, goal difference and points, and fixtures MUST be grouped by matchday with their kickoff date.

#### Scenario: Knockout under way
- **WHEN** a user opens the World Cup tab during the knockout stage
- **THEN** they see the final group tables and the bracket

#### Scenario: Group fixtures by matchday
- **WHEN** a user opens a group card during the group stage
- **THEN** its fixtures are listed under "Jornada 1", "Jornada 2", "Jornada 3", each with its date

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

## Code map

- `backend/engine/worldcup.go` (`WorldCupFrequencySeasons`, `StartWorldCupForNewSeason`,
  `startWorldCupGroup`, `AdvanceWorldCups`, `startWorldCupKnockout`), started from
  `StartInternationalCupsIfDue` (`engine/cups.go`, `InternationalCupCadence`) and advanced by
  `CupJob` (`engine/tasks.go`).
- `common/worldcup.go` (`WorldCupGroupSize`, `MinWorldCupCountries`, `AssignWorldCupGroups`,
  `BuildRoundRobinRounds`, `RankGroupStandings`, `UpdateWorldCupStandingsAfterMatch`,
  `HasActiveWorldCup`, `IsWorldCupGroupStageComplete` - every group match played and the last
  kickoff at or before `LiveBroadcastCutoff`, `SetWorldCupKnockoutStarted`),
  `common.CancelPendingNationalTeamFriendlies`.
- Display: `common.GetWorldCupGroups` derives each group's table from its fixtures whose broadcast
  has finished (`deriveWorldCupGroupStandings`, `LiveBroadcastCutoff`) and returns fixtures
  broadcast-safe (`applyNationalMatchStatus` in `common/nationalmatches.go`); the stored
  `world_cup_group_teams` totals are read only by `GetWorldCupGroupQualifiers`.
- History and estimate: `common/worldcuphistory.go` (`GetWorldCupHistory`,
  `GetNextWorldCupEstimate`, `nextWorldCupStart`, `WorldCupEveryInternationalCycles`,
  `InternationalCupCycle`); handlers `CurrentWorldCupHandler`, `WorldCupHistoryHandler`
  (`handlers/nationalTeamHandlers.go`).
- Frontend: `features/nationalteams/WorldCupTab.tsx`.
- Schema: `resources/sql/init72_world_cup.sql`.
- Viewing a group match: it renders through `frontend/src/features/match/MatchViewer.tsx` (route
  adapter `features/friendlies/FriendlyMatchPage.tsx`, `fromFriendly` in `matchViewModel.ts`); the
  "Mundial · Grupo X · Jornada N" label comes from `common.FriendlyMatch.WorldCupGroupName` /
  `WorldCupRound`, joined from `world_cup_group_matches` / `world_cup_groups` in
  `GetFriendlyMatchByIdStmtSql`.
- Multi-replica safety (`common.ClaimInternationalCupSeason`, the World Cup start holding the
  national-team scheduling lock, `advanceWorldCup`'s per-World-Cup lock): see the engine-replicas
  spec.
