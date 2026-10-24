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
`GET /v1/national-teams/world-cup/current` SHALL return the latest World Cup with its groups' standings and fixtures; its knockout bracket is read through `GET /v1/cups/:id` and rendered with the cup page's components.

#### Scenario: Knockout under way
- **WHEN** a user opens the World Cup tab during the knockout stage
- **THEN** they see the final group tables and the bracket

## Code map

- `backend/engine/worldcup.go` (`WorldCupFrequencySeasons`, `StartWorldCupForNewSeason`,
  `startWorldCupGroup`, `AdvanceWorldCups`, `startWorldCupKnockout`), started from
  `StartInternationalCupsIfDue` (`engine/cups.go`, `InternationalCupCadence`) and advanced by
  `CupJob` (`engine/tasks.go`).
- `common/worldcup.go` (`WorldCupGroupSize`, `MinWorldCupCountries`, `AssignWorldCupGroups`,
  `BuildRoundRobinRounds`, `RankGroupStandings`, `UpdateWorldCupStandingsAfterMatch`,
  `HasActiveWorldCup`), `common.CancelPendingNationalTeamFriendlies`.
- Schema: `resources/sql/init72_world_cup.sql`.
- Viewing a group match: it renders through `frontend/src/features/match/MatchViewer.tsx` (route
  adapter `features/friendlies/FriendlyMatchPage.tsx`, `fromFriendly` in `matchViewModel.ts`); the
  "Mundial · Grupo X · Jornada N" label comes from `common.FriendlyMatch.WorldCupGroupName` /
  `WorldCupRound`, joined from `world_cup_group_matches` / `world_cup_groups` in
  `GetFriendlyMatchByIdStmtSql`.
- Multi-replica safety (`common.ClaimInternationalCupSeason`, the World Cup start holding the
  national-team scheduling lock, `advanceWorldCup`'s per-World-Cup lock): see the engine-replicas
  spec.
