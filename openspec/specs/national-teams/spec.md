# national-teams Specification

## Purpose

Every active country fields a national team drawn from its best players by birth country. This
covers how those teams exist, how their friendlies are scheduled and staffed, how caps and
call-ups are counted and notified, and where they are shown.

## Requirements

### Requirement: National team entity
Every active country SHALL have one unowned national team with zero budget, zero stadium capacity, no league and AI level 0, so no club system (user assignment, AI market, CPU friendlies, team search) touches it.

#### Scenario: Gate receipts
- **WHEN** a national-team friendly is played
- **THEN** no gate money is earned, because the stadium capacity is zero

### Requirement: Eligible squad pool
A country's pool SHALL be its 40 highest-rated players by sum of the seven core skills, born in that country and currently in a club's senior squad (no academy prospects or free agents).

#### Scenario: Free agent
- **WHEN** a country's best player is a free agent
- **THEN** he is not in the pool

### Requirement: Friendly pairing
Every 5 minutes, while no World Cup is running, the engine SHALL pair every active country that has no pending national-team fixture, preferring same-region opponents, leaving at most one country unpaired. Each fixture MUST kick off next Saturday 21:00 in the home side's local timezone.

#### Scenario: Previous friendly played
- **WHEN** a country's friendly has just been played
- **THEN** it is re-paired for its next fixture within minutes

#### Scenario: World Cup running
- **WHEN** a World Cup is active
- **THEN** no new national-team friendlies are scheduled

#### Scenario: Two engine replicas
- **WHEN** two engine replicas run the pairing pass at once
- **THEN** an advisory lock lets only one of them create fixtures

### Requirement: Call-up shortly before kickoff
Within 24 hours of kickoff the engine SHALL select each national side's best XI from its current pool and save it as that match's lineup, once per side. National-team cup ties get their lineup the same way when each knockout round is created.

#### Scenario: Fixture three days away
- **WHEN** a national-team friendly is three days away
- **THEN** no lineup has been selected yet

#### Scenario: Retry on failure
- **WHEN** one side's call-up succeeded and the other's failed
- **THEN** only the failed side is retried on later ticks

### Requirement: Call-up notification
When a national lineup is saved, each selected player's club owner SHALL receive a `NOTIFICATION_CALL_UP` linking to that fixture.

#### Scenario: Owner's player selected
- **WHEN** an owner's player is called up for a friendly
- **THEN** the owner is notified with a link to that friendly

### Requirement: Ordinary friendly mechanics
A national-team fixture SHALL be simulated by the regular friendly job and counts for skill growth like any friendly.

#### Scenario: Player features for his country
- **WHEN** a player plays a national-team friendly
- **THEN** his skills may drift afterwards as after any match

### Requirement: International caps
A player's caps SHALL be the number of played national-team friendlies and national-team cup ties in which he was in his country's saved starting lineup. His profile MUST show caps and his recent call-ups, each linking to the fixture.

#### Scenario: World Cup knockout start
- **WHEN** a player starts a World Cup knockout tie for his country
- **THEN** his caps increase by one once it is played

### Requirement: National teams page
The frontend SHALL offer `/national-teams` with a global feed of upcoming and recent national-team friendlies (`GET /v1/national-teams/friendlies/{upcoming,recent}`, excluding World Cup group matches) and the World Cup section.

#### Scenario: Manager browses internationals
- **WHEN** a manager opens `/national-teams`
- **THEN** they see upcoming and recent friendlies for every country

## Code map

- `backend/common/nationalteam.go` (`NationalSquadPoolSize`, `GetPlayersByBirthCountry`,
  `ScheduleNationalTeamFriendlies`, `pairNationalTeamCountries`, `CallUpNationalTeamSquads`,
  `nationalTeamCallUpWindow`, `SetNationalTeamLineup`, `SetNationalTeamCupTieLineup`,
  `notifyCalledUpPlayers`, `GetRecentNationalTeamCallUps`).
- Jobs: `StartNationalTeamFriendlyScheduler`, `StartNationalTeamCallUpScheduler`
  (`engine/scheduler.go`).
- Caps: `common.GetPlayerCareerStats`. Frontend: `NationalTeamsPage.tsx`.
