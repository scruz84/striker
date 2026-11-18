# team-history Specification

## Purpose
A team's history tab (the trophy cabinet): one line per season with its league, final position,
promotion or relegation and the cups it won, plus a summary of every title the club holds.

## Requirements

### Requirement: Season lines
The history tab SHALL list one line per season the team has a record for, newest first. Each line shows the season, the league and final position (linking to that league's table for that season), a trophy icon for 1st place, and chips for promotion, relegation, a domestic cup win and an international cup win. A season can show both cup chips.

#### Scenario: Double winner
- **WHEN** a team wins its domestic cup and the European Cup edition 3 during its league season 5
- **THEN** the season 5 line shows "Cup winner" and "European Cup winner (Ed. 3)"

#### Scenario: No titles
- **WHEN** a team finished 6th in season 2 and won nothing
- **THEN** the season 2 line shows no cup chip

### Requirement: Domestic cup title
A domestic cup win SHALL be recorded in the season of that cup, which is the season of the country's league it was played in. Its chip SHALL read "Cup winner" / "Campeón de copa".

#### Scenario: Cup final mid-season
- **WHEN** a team wins its country's season-4 cup
- **THEN** its season 4 line shows "Cup winner"

### Requirement: International cup title
A regional cup win (European Cup, Copa America, Asian Cup, African Cup) SHALL be recorded in the league season the champion's country is playing when the final is resolved, never in the season equal to the edition number. Its gold chip names the cup and edition and links to that edition. The World Cup knockout SHALL write nothing to a club's history.

#### Scenario: Edition number differs from the season
- **WHEN** a team whose country is playing season 6 wins European Cup edition 2
- **THEN** its season 6 line shows "European Cup winner (Ed. 2)" and its season 2 line shows no cup chip

#### Scenario: Opening the edition
- **WHEN** the user clicks "European Cup winner (Ed. 2)"
- **THEN** the International Cup page opens on the European Cup tab showing edition 2

#### Scenario: Title before the season ends
- **WHEN** the final is resolved while season 6 is still being played
- **THEN** the season 6 line exists already, showing the chip without a final position until the season ends

### Requirement: Honours summary
Above the season lines the history tab SHALL show the club's honours: league titles (1st place in a tier-1 league), domestic cups, and one count per international cup won. A count of 0 is not shown, and the summary is absent when the club has won nothing.

#### Scenario: Summary
- **WHEN** a team won tier 1 twice, its domestic cup once and the European Cup once, and also won a tier-2 group
- **THEN** the summary reads "League titles 2 · Cups 1 · European Cup 1"

#### Scenario: No honours
- **WHEN** a team has never won a title
- **THEN** no summary is shown

### Requirement: Existing history on upgrade
On upgrade, every recorded cup win that no domestic cup of that country and season backs SHALL be removed, a season line that existed only for it SHALL be deleted, and every completed regional cup SHALL be recorded in its champion's history in the league season of its last league match on or before the final (or the first one after it, if it has none before). Re-running the migration MUST change nothing.

#### Scenario: Wrong season fixed
- **WHEN** the database has "Champion" in a team's season 2 because it won European Cup edition 2 during season 6, and the team did not win its season-2 domestic cup
- **THEN** after the migration season 2 shows no cup chip and season 6 shows "European Cup winner (Ed. 2)"

#### Scenario: Real domestic win kept
- **WHEN** a team won its season-2 domestic cup and European Cup edition 2
- **THEN** season 2 keeps "Cup winner"

## Code map

- Engine: `backend/engine/cups.go` (`advanceCup`, `recordCupTitle`).
- Data: `backend/common/seasonhistory.go` (`SeasonHistoryEntry`, `SeasonHistoryInternationalCup`,
  `RecordSeasonHistoryCupResult`, `RecordSeasonHistoryInternationalCup`, `GetSeasonHistory`).
- Schema: `resources/sql/history/init111_team_cup_titles.sql` (`season_history.international_cup_id`,
  clean-up and backfill), `history/init15_season_history.sql`, `init101_league_season_history.sql`.
- API: `TeamHistoryHandler` (`handlers/seasonHistoryHandlers.go`).
- Frontend: `features/team/TeamDetailsPage.tsx` (`TrophyCabinet`, `HonoursSummary`); test
  `features/team/TrophyCabinet.test.tsx`.
- Tests: `common/cuptitles_integration_test.go`, `engine/cup_titles_integration_test.go`.
