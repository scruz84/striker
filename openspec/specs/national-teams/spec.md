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
Every 5 minutes, while no World Cup is running, the engine SHALL pair every active country that has no pending national-team fixture, preferring same-region opponents, leaving at most one country unpaired. Each fixture MUST kick off at 21:00 in the home side's local timezone on the first Saturday that is in the future and at least 6 days after both countries' latest national-team kickoff.

#### Scenario: Previous friendly played
- **WHEN** a country's friendly has just been played
- **THEN** it is re-paired for its next fixture within minutes
- **AND** that fixture kicks off the following Saturday, not the same evening

#### Scenario: Re-paired inside the early-simulation window
- **WHEN** Nigeria's friendly kicks off Saturday 20:00 UTC and São Tomé's friendly (21:00 UTC) is simulated at 20:46 UTC, and the pass at 20:50 UTC pairs São Tomé (home) with Nigeria
- **THEN** the new fixture kicks off the next Saturday 21:00 São Tomé time, 7 days later
- **AND** neither country has two national-team kickoffs on the same Saturday

#### Scenario: Country that has never played
- **WHEN** a newly activated country with no previous national-team match is paired
- **THEN** its fixture kicks off the next local Saturday 21:00 that is still in the future, subject only to its opponent's rest

#### Scenario: World Cup running
- **WHEN** a World Cup is active
- **THEN** no new national-team friendlies are scheduled

#### Scenario: Two engine replicas
- **WHEN** two engine replicas run the pairing pass at once
- **THEN** an advisory lock lets only one of them create fixtures

#### Scenario: World Cup starting
- **WHEN** a World Cup starts while a pairing pass is running
- **THEN** the World Cup start waits for the pass's lock, so no country gets both a friendly and a group match that week

### Requirement: Call-up shortly before kickoff
Within 24 hours of kickoff the engine SHALL select each national side's best XI from its current pool and save it as that match's lineup, once per side. National-team cup ties get their lineup the same way when each knockout round is created.

#### Scenario: Fixture three days away
- **WHEN** a national-team friendly is three days away
- **THEN** no lineup has been selected yet

#### Scenario: Retry on failure
- **WHEN** one side's call-up succeeded and the other's failed
- **THEN** only the failed side is retried on later ticks

#### Scenario: Two engine replicas
- **WHEN** two engine replicas run the call-up pass at once
- **THEN** each side is called up, and each club notified, once

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
The frontend SHALL offer `/national-teams` with three tabs: Jornada (the current round), Resultados (paginated history) and Mundial (the World Cup section). Jornada is the default tab and the tab is kept in the URL (`?tab=round|results|world-cup`).

#### Scenario: Manager browses internationals
- **WHEN** a manager opens `/national-teams`
- **THEN** they see the Jornada tab with every live, upcoming and just-finished national-team match of every country

#### Scenario: Shared link to the World Cup
- **WHEN** a manager opens `/national-teams?tab=world-cup`
- **THEN** the Mundial tab is selected

### Requirement: Broadcast-safe national-team scores
No national-team endpoint SHALL return the final score of a match whose kickoff is less than 2 hours ago (or that was simulated before kickoff). Such a match MUST be reported with status `live`, the current minute and the score as of that minute; a match not yet simulated MUST report status `scheduled` and no score.

#### Scenario: Live friendly that ends 2-3
- **WHEN** a friendly that ends Ivory Coast 2 - 3 Tanzania is at minute 40 with the broadcast showing 1 - 0
- **THEN** every national-team feed reports it as `live`, minute 40, score 1 - 0
- **AND** the match page and the feed show the same score

#### Scenario: Simulated before kickoff
- **WHEN** a friendly was simulated 10 minutes before its 21:00 kickoff and it is 20:55
- **THEN** feeds report it as `live` at minute 0 with score 0 - 0

#### Scenario: Broadcast finished
- **WHEN** the match kicked off 2 hours ago or more
- **THEN** feeds report it as `finished` with its final score

### Requirement: Current round feed
`GET /v1/national-teams/round` SHALL return, uncapped, every national-team match (friendly, World Cup group match or World Cup knockout tie) that is unplayed, live, or finished within the last 24 hours, ordered by kickoff, each with its competition label, status, minute and broadcast-safe score.

#### Scenario: Saturday evening with 100 fixtures
- **WHEN** 104 national-team friendlies kick off this Saturday
- **THEN** the round feed returns all 104, and the Jornada tab lists them grouped by kickoff time in the viewer's timezone

#### Scenario: Live section
- **WHEN** 30 of those matches are live
- **THEN** the Jornada tab shows all 30 first, each with its live minute and broadcast-safe score

### Requirement: Results history
`GET /v1/national-teams/results?page=N&country=XX` SHALL return finished national-team matches (friendlies, World Cup group matches and knockout ties) whose broadcast has ended, newest first, 25 per page, with a flag telling whether another page exists. `country` limits it to that country's matches.

#### Scenario: Second page
- **WHEN** a manager taps "Load more" on the Resultados tab
- **THEN** the next 25 older results are appended

#### Scenario: Live match not in results
- **WHEN** a match kicked off 30 minutes ago
- **THEN** it is absent from the results history until its broadcast ends

### Requirement: Country search and region filter
The Jornada and Resultados tabs SHALL offer a country search (matching the localized country name) and region chips (the four game regions: Europe, Africa, Asia, Americas); a filter keeps a match when either side matches.

#### Scenario: Searching for Spain
- **WHEN** a manager types "Esp" on the Jornada tab
- **THEN** only matches involving España are listed

### Requirement: My players badge
For a manager who owns a club, every national-team match in the round feed whose saved call-up includes players of the owner's club SHALL carry that count; the page shows it as a badge and offers a "Only my players" toggle.

#### Scenario: Two players called up
- **WHEN** two of an owner's players are in Nigeria's saved XI for Saturday
- **THEN** that fixture shows a badge "2 of your players"

#### Scenario: Call-up not yet made
- **WHEN** a fixture is more than 24 hours away and no lineup is saved
- **THEN** it carries no badge

#### Scenario: Manager without a club
- **WHEN** a signed-in manager with no club assigned opens the page
- **THEN** no badge or toggle is shown

### Requirement: Compact match rows
National-team match rows SHALL fit a 360 px wide screen without wrapping a team name over more than one line: the competition label and kickoff (or live minute) on a first line, flags, ellipsized names and the score on a second, the whole row being one tap target of at least 44 px.

#### Scenario: Long names on a phone
- **WHEN** "Irlanda del Norte" plays "Santo Tomé y Príncipe" on a 360 px screen
- **THEN** both names stay on one line, ellipsized if needed, and the score stays centred

### Requirement: National team matches tab
The team page of a national team SHALL offer a Partidos tab with its next fixture, its last 10 results (broadcast-safe, from the results history) and a form strip of its last 5 results (W/D/L).

#### Scenario: Viewing Spain
- **WHEN** a manager opens Spain's team page and selects Partidos
- **THEN** they see Spain's next friendly, its last 10 results and a five-letter form strip such as W W D L W

## Code map

- `backend/common/nationalteam.go` (`NationalSquadPoolSize`, `GetPlayersByBirthCountry`,
  `ScheduleNationalTeamFriendlies`, `pairNationalTeamCountries`, `NationalTeamFriendlyRest`,
  `nationalTeamFriendlyEarliestKickoff`, `GetLatestNationalTeamKickoffs`
  (`getLatestNationalTeamKickoffsStmtSql`), `CallUpNationalTeamSquads`,
  `nationalTeamCallUpWindow`, `SetNationalTeamLineup`, `SetNationalTeamCupTieLineup`,
  `notifyCalledUpPlayers`, `GetRecentNationalTeamCallUps`).
- Kickoff: `common.NextLocalSaturdayKickoffAfter` (`common/timezones.go`).
- Jobs: `StartNationalTeamFriendlyScheduler`, `StartNationalTeamCallUpScheduler`
  (`engine/scheduler.go`).
- Feeds: `common/nationalmatches.go` (`NationalMatch`, `applyNationalMatchStatus` - the one place
  deciding what a feed may reveal, using `liveScoreAsOfNow` in `common/livematches.go` -
  `GetNationalTeamRound`, `GetNationalTeamResults`, `NationalTeamResultsPageSize`,
  `SetMyPlayersInNationalMatches`); handlers `NationalTeamRoundHandler`,
  `NationalTeamResultsHandler` (`handlers/nationalTeamHandlers.go`), routes `/v1/national-teams/round`
  and `/v1/national-teams/results` (`service/fmservice.go`).
- Caps: `common.GetPlayerCareerStats`.
- Frontend: `features/nationalteams/` - `NationalTeamsPage.tsx` (tabs in `?tab=`), `RoundTab.tsx`,
  `ResultsTab.tsx`, `WorldCupTab.tsx`, `NationalMatchRow.tsx` (compact row),
  `NationalMatchFilters.tsx`, `nationalMatches.ts` (grouping, filters, labels, form strip),
  `NationalTeamMatchesTab.tsx` (shown by `features/team/TeamDetailsPage.tsx`, tabs in
  `features/team/teamTabs.ts`); hooks `useNationalTeamRound`, `useNationalTeamResults`,
  `useWorldCupHistory` (`api/hooks.ts`).
- Multi-replica safety (locks on pairing, World Cup start and call-ups): see the engine-replicas
  spec.
