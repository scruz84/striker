# international-cup Specification

## Purpose
The regional (international) club cups - European Cup, Copa America, Asian Cup, African Cup:
when a region starts a new edition from its countries' new league champions, who enters (each
country's top two), how the first round is drawn, how editions are numbered, and how the start
stays safe on several engine replicas.

## Requirements

### Requirement: Regional cup start
Every 5 minutes the engine SHALL check each region of the active countries. A region MUST start a new edition when none of its editions is running, at least 2 of its countries with a tier-1 league are fresh (latest tier-1 champion has not entered yet), and either all of them are fresh or one has 2 or more unused champions.

#### Scenario: Every country has a new champion
- **WHEN** Spain, France and Italy are in europe, all three have finished a season since the last European Cup, and no European Cup is running
- **THEN** the next check starts a European Cup with the three latest champions

#### Scenario: Waiting for the last country
- **WHEN** Spain and France have new champions but Italy is still playing its season and no country has finished two seasons since it last entered
- **THEN** no European Cup starts yet

#### Scenario: A slow country does not hold the region back
- **WHEN** Italy is still playing and Spain finishes a second season since it last entered while France has one new champion
- **THEN** a European Cup starts with Spain's latest champion and France's, and Italy enters a later edition

#### Scenario: Edition still running
- **WHEN** every country has a new champion but the current European Cup has not finished
- **THEN** no new edition starts until it is completed

#### Scenario: One country only
- **WHEN** only one country in a region has a new champion, however many seasons it has finished
- **THEN** no edition starts in that region

### Requirement: Regional cup entrants
Each fresh country SHALL enter once, with the champion and the runner-up (final positions 1 and 2) of its most recent tier-1 season (lowest league id when it has several tier-1 groups). With no recorded runner-up it enters with its champion alone. The engine MUST record, per edition, each entrant team with its country, final position and that domestic season; a country is fresh again only once it has a champion from a newer season.

#### Scenario: Champion already played
- **WHEN** France's season-4 top two played the last European Cup and France has not finished season 5
- **THEN** France is not fresh and does not enter the next edition

#### Scenario: Two titles since the last edition
- **WHEN** Spain finished seasons 5 and 6 since it last entered with its season-4 top two
- **THEN** Spain enters with its season-6 champion and runner-up, and its season-5 top two never play the cup

#### Scenario: Top two of each country
- **WHEN** a European Cup starts with Spain and France fresh
- **THEN** it has 4 entrants: Spain's 1st and 2nd and France's 1st and 2nd of their latest season

#### Scenario: Missing runner-up
- **WHEN** Italy's latest tier-1 season has a champion but no recorded 2nd place
- **THEN** Italy enters with its champion only and the edition still starts

#### Scenario: Existing editions on upgrade
- **WHEN** the migration runs on a database whose last European Cup had Spain's season-3 champion
- **THEN** that entry is kept as Spain's 1st place and Spain is fresh only after finishing season 4

### Requirement: Regional cup editions
An edition SHALL be created with its entries and first round in one transaction, numbered 1, 2, 3... within its region, and run as a single-elimination cup, one round a day, each tie kicking off at 20:45 in its home side's country (see match-scheduling). The cups are European Cup, Copa America, Asian Cup and African Cup.

#### Scenario: Third edition
- **WHEN** the americas region already had two Copa America editions and a new one starts
- **THEN** it is shown as "Copa America - Edition 3"

#### Scenario: Failure while writing
- **WHEN** writing the first round fails
- **THEN** no cup row or entry remains, and the next check tries again

### Requirement: Regional cup start on several replicas
The regional cup check SHALL run under one global session lock taken without waiting, and re-read each region's state inside it.

#### Scenario: Three replicas tick together
- **WHEN** three replicas run the check at once with a European Cup due
- **THEN** exactly one European Cup edition is created

### Requirement: Regional cup page
The International Cup page SHALL show every region's latest edition to every user, one tab per region in the order European Cup, Copa America, Asian Cup, African Cup. It MUST open on the user's own region (from their team's country) and show an empty state for a region with no edition yet.

#### Scenario: Watching another region's cup
- **WHEN** a user whose team is in Argentina opens the page and picks the European Cup tab
- **THEN** the page shows the latest European Cup edition and its bracket

#### Scenario: Region without an edition
- **WHEN** a user picks the Asian Cup tab and no Asian Cup has started yet
- **THEN** the page says that region has no edition yet

### Requirement: Regional cup title in the champion's history
When a regional cup edition completes, its champion SHALL get the title in its history (see team-history "International cup title"), in the league season its country is playing at that moment. The edition number MUST never be used as a league season.

#### Scenario: Champion of edition 3
- **WHEN** a Spanish team wins European Cup edition 3 while Spain plays season 7
- **THEN** the team's season 7 history line shows "European Cup winner (Ed. 3)"

### Requirement: Regional cup page opens a given edition
The International Cup page SHALL read `region` and `cup` from its URL. With a valid `region` it opens that tab; with `cup` it shows that edition of the tab's region instead of the latest one. Switching tabs MUST drop `cup` and show that region's latest edition.

#### Scenario: Link from the history tab
- **WHEN** the user opens `/international-cup?region=europe&cup=41`
- **THEN** the European Cup tab shows edition cup 41 and its bracket

#### Scenario: Back to the latest
- **WHEN** the user then picks the Copa America tab
- **THEN** the page shows the latest Copa America edition

### Requirement: Regional cup draw
With N entrants and P the largest power of two ≤ N, round 1 SHALL give 2P − N byes when N ≠ P. Byes go to champions first, then runners-up, each by current league points (ties at random), unless moving a bye to a team of the same position avoids a same-country tie. The other teams are paired champion vs a runner-up of another country where possible, and never two teams of one country when another pairing exists. Byes alternate with real ties so two byes never meet in round 2.

#### Scenario: Two countries
- **WHEN** Spain (A1, A2) and France (B1, B2) enter
- **THEN** round 1 is A1 vs B2 and B1 vs A2, with no byes

#### Scenario: Three countries
- **WHEN** Spain, France and Italy enter with 6 teams and Spain's and France's champions have the most points among the champions
- **THEN** A1 and B1 get byes, C1 plays A2 or B2, and the other two runners-up play each other

#### Scenario: Four countries
- **WHEN** four countries enter with 8 teams
- **THEN** there are no byes and every round-1 tie is a champion against another country's runner-up

#### Scenario: Bye that avoids a same-country tie
- **WHEN** Spain (A1, A2) and France (B1, no runner-up) enter and B1 has more points than A1
- **THEN** A1 gets the bye and B1 plays A2, instead of A1 playing A2

#### Scenario: Round 2 after byes
- **WHEN** A1 and B1 got byes in a 6-team edition
- **THEN** in round 2 each of them plays a round-1 winner, not each other

## Code map

- Engine: `backend/engine/cups.go` (`StartRegionalCupsIfDue`, `regionalCupEntrants`,
  `regionalCupTeams`, `RegionalCupEntrantsPerCountry`, `MinRegionalCupCountries`,
  `RegionalCupMaxUnusedChampions`, `internationalCupNames`, `cupTieKickoff`), the draw in
  `engine/regionalcupdraw.go` (`regionalCupFirstRound`), run by `RegionalCupJob`
  (`engine/tasks.go`, `StartRegionalCupScheduler` in `engine/scheduler.go`). Rounds advance through `AdvanceCups` like a domestic cup; a completed
  edition is recorded in its champion's history by `recordCupTitle` (see team-history).
- Data: `backend/common/regionalcups.go` (`GetRegionalCupCountryStates`, `RegionalCupEntrant`,
  `HasActiveRegionalCup`, `CreateRegionalCup`, `GetInternationalCupEntries`), `common/cups.go`
  (`GetLatestCupForRegion`); regions in `common/regions.go` (`RegionForCountry`); lock `LockNamespaceRegionalCupStart` (`common/enginelocks.go`).
- Schema: `resources/sql/history/init103_international_cup_entries.sql`
  (`international_cup_entries`), `init112_international_cup_runners_up.sql` (`final_position`,
  primary key `(cup_id, team_id)`), `history/init52_international_cup.sql` (`cups.region`).
- API: `LatestInternationalCupHandler` (`handlers/cupHandlers.go`).
- Frontend: `features/cup/InternationalCupPage.tsx` (`INTERNATIONAL_CUP_REGIONS`, one tab per
  region, `?region=` / `?cup=` URL parameters); test `features/cup/InternationalCupPage.test.tsx`.
- Tests: `engine/cups_test.go` (`regionalCupEntrants`), `engine/regionalcupdraw_test.go`,
  `engine/cups_integration_test.go`, `engine/engine_replicas_integration_test.go`.
