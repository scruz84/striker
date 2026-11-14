# league-pyramid Specification

## Purpose
Defines the shape of a country's league pyramid and how teams move between its tiers at the end
of a season: direct promotion and relegation, the promotion playoff, and restructuring a country
whose leagues no longer match the configured pyramid.

## Requirements

### Requirement: Pyramid structure rule
A world structure SHALL be a list of tiers numbered 1..T with no gaps. Tier 1 has exactly 1 group, and every tier t > 1 has exactly 2 × the groups of tier t−1. Every tier SHALL have the same teams per group, an even number from 8 to 24. The existing limits still apply: at most 6 tiers and at most 1500 teams per country. Saving a structure that breaks any of these rules SHALL be rejected with a message naming the rule.

#### Scenario: Valid default
- **WHEN** the admin saves tiers I/II/III/IV/V with 1/2/4/8/16 groups of 10 teams
- **THEN** the structure is saved (310 teams per country)

#### Scenario: Group count breaks the doubling rule
- **WHEN** the admin saves 1/4/16 groups
- **THEN** the save is rejected with "tier 2 must have exactly 2 groups (twice tier 1)"

#### Scenario: Too few teams for the zones
- **WHEN** the admin saves a tier with 6 teams per group
- **THEN** the save is rejected, because a group needs room for 2 promotion places, 4 relegation places and 2 safe teams

### Requirement: Structure fits existing teams
Saving a structure SHALL be rejected when its places per country (the sum of groups × teams per group) are fewer than the teams of any active country, because a restructure never deletes a team.

#### Scenario: Shrinking below the current team count
- **WHEN** an active country has 310 teams and the admin saves 1/2/4/8 groups of 10 (150 places)
- **THEN** the save is rejected, naming the country and both numbers

### Requirement: Default world structure
When no structure has been saved, the default SHALL be 5 tiers named I, II, III, IV and V, with 1, 2, 4, 8 and 16 groups of 10 teams each.

#### Scenario: Fresh world
- **WHEN** a world is initialized with no saved structure
- **THEN** each country gets 31 leagues and 310 teams

### Requirement: Zones of a group
In a country with T tiers, every group SHALL have these zones by final position (n = teams in the group):
- In tiers 2..T, 1st is direct promotion and 2nd is promotion playoff.
- In tiers 1..T−1, positions n−3 and n−2 are relegation playoff, and n−1 and n are direct relegation.

Tier 1 has no promotion zone and tier T has no relegation zone. Final order is points, goal difference, then goals scored.

#### Scenario: Top division of 10
- **WHEN** the tier-1 season ends
- **THEN** 7th and 8th play the relegation playoff, and 9th and 10th are relegated

#### Scenario: Middle tier group of 10
- **WHEN** a tier-2 group ends
- **THEN** 1st is promoted, 2nd plays the promotion playoff, 7th and 8th play the relegation playoff, and 9th and 10th are relegated

#### Scenario: Bottom tier
- **WHEN** a tier-5 group ends in a 5-tier country
- **THEN** 1st is promoted, 2nd plays the promotion playoff, and nobody is relegated

### Requirement: Direct moves
At rollover, every direct-promotion team SHALL move to tier t−1, and every direct-relegation team SHALL move to tier t+1. Each promoted team takes the place of one relegated team: relegated teams are taken worst first and promoted teams best first, and pair k swaps leagues. Every group therefore keeps its size.

#### Scenario: Tier 1 and tier 2
- **WHEN** the tier-2 winners are A (50 pts) and B (47 pts), and tier 1's 10th and 9th are X and Y
- **THEN** A takes X's place, B takes Y's place, and X and Y go to A's and B's groups

### Requirement: Playoff pairings
For each pair of adjacent tiers, the upper tier's relegation-playoff teams SHALL be ordered worst first, and the lower tier's runners-up best first. Worst means highest position, then fewest points, then goal difference and goals scored. Best means most points, then goal difference, then goals scored. Pair k is one tie: the upper-tier team is at home and the lower-tier team away.

#### Scenario: Tier 1 against tier 2
- **WHEN** tier 1's 8th is P and 7th is Q, and tier 2's runners-up are R (45 pts) and S (41 pts)
- **THEN** the ties are P vs R and Q vs S, with P and Q at home

#### Scenario: Two upper groups
- **WHEN** tier 2's two groups have relegation-playoff teams 8th (20 pts), 8th (18 pts), 7th (22 pts) and 7th (25 pts)
- **THEN** they meet tier 3's four runners-up, the 8th on 18 pts facing the best runner-up

### Requirement: Playoff match
A playoff tie SHALL be a single match with no extra time. A draw goes to penalties, settled 50/50 as in a cup tie. It counts like a cup tie for lineups, live broadcast, gate, sponsorship and match effects. It pays no prize money of its own and no cup round prize, does not count as a cup final, and is not part of any cup.

#### Scenario: Draw
- **WHEN** a playoff tie ends 1-1
- **THEN** a penalty shootout decides the winner, and neither side receives a cup round prize

### Requirement: Playoff outcome
At rollover, the winner of each playoff tie SHALL play next season in the upper tier and the loser in the lower tier. The two teams swap leagues when the lower-tier team wins and stay put when the upper-tier team wins.

#### Scenario: Lower team wins
- **WHEN** R from tier 2 beats P from tier 1
- **THEN** R plays next season in P's tier-1 league and P in R's tier-2 group

#### Scenario: Upper team wins
- **WHEN** Q beats S
- **THEN** both stay in their leagues

### Requirement: Playoff phase
Once every league match of a country's season has been played and its broadcast has finished, the engine SHALL create that season's playoff ties for the country. It does this exactly once. A tie between tiers t and t+1 kicks off on the day after the last league matchday's local date, at tier t's league kickoff time (the country's kickoff + 15 min × (t − 1)). If that is less than 1 hour after the ties are created, the kickoff moves one day. A country with a single tier has no playoff.

#### Scenario: Last matchday broadcast ends
- **WHEN** a 5-tier country with a 14:00 kickoff plays its last league matchday on Monday and the tier-5 broadcasts end at 16:50
- **THEN** 16 playoff ties are created (1+2+4+8 for the four tier pairs), kicking off on Tuesday at 14:00 (I–II), 14:15 (II–III), 14:30 (III–IV) and 14:45 (IV–V)

#### Scenario: Late matchday ending after midnight
- **WHEN** a league with an override kicks off its last matchday on Monday at 23:00 and its broadcast ends on Tuesday at 01:00
- **THEN** the playoff kicks off on Tuesday, the day after the matchday's date, at each tier pair's kickoff time

### Requirement: Rollover waits for the playoff
A country's season SHALL NOT roll over until every one of its playoff ties has been played and its broadcast has finished. Prize money, season income, awards and the final tables SHALL still use the league standings, which the playoff does not change.

#### Scenario: Playoff still live
- **WHEN** all league matches are over but one playoff tie is still broadcasting
- **THEN** the country stays in the same season

### Requirement: Playoff notifications
When playoff ties are created, the owner of each team in one SHALL receive a notification naming the opponent and linking to the tie's page, which shows the kickoff. At rollover, a team that went up or down through the playoff SHALL receive a promotion or relegation notification that says so.

#### Scenario: Qualified for the playoff
- **WHEN** a user's team finishes 2nd in its tier-3 group
- **THEN** once the league broadcasts end, the user is told who they play, with a link to the tie

### Requirement: Season history marks playoff moves
A team's season history line SHALL record whether its promotion or relegation came through the playoff. A team that won its playoff and stayed put records neither promotion nor relegation.

#### Scenario: Promoted through the playoff
- **WHEN** R wins its playoff and goes up
- **THEN** R's history line for that season is promoted, through the playoff

### Requirement: Legacy pyramid season
A country whose active leagues break the structure rule SHALL play no playoff. At its next rollover, the bottom 2 of each upper group SHALL be relegated and swapped one for one with the best lower-tier group winners, ranked by points, goal difference and goals scored. The restructure then follows. Its tables SHALL mark lower-tier winners as promotion candidates in the playoff colour, never as promoted.

#### Scenario: Season in progress at deploy
- **WHEN** a 1/4/16 country finishes its season
- **THEN** no playoff is created, tier 1's 9th and 10th swap with the best 2 of the 4 tier-2 winners, and the country is then restructured

#### Scenario: Legacy table
- **WHEN** the user opens a tier-2 group of a 1/4/16 country
- **THEN** its 1st is in the playoff colour with the legend "Promotion candidate", and 9th and 10th are in the relegation colour

### Requirement: Restructure detection
At rollover, a country SHALL be restructured when its active leagues differ from the saved structure in the number of tiers, the groups of any tier, or the teams per group. Saving a new structure SHALL change nothing until each country's next rollover. A country that is already restructured is left alone.

#### Scenario: Old world
- **WHEN** a country has 1/4/16 groups and the saved structure is 1/2/4/8/16
- **THEN** its next rollover restructures it

### Requirement: Restructure ordering
A restructure SHALL first apply the season's direct and playoff moves. Then every team is ordered by tier after those moves, then by final position, points, goal difference and goals scored. Tier 1 takes the first G1 × N teams, tier 2 the next G2 × N, and so on. Teams that do not fit in a tier drop to the next one.

#### Scenario: From 1/4/16 to 1/2/4/8/16
- **WHEN** a country with 210 teams in 1/4/16 groups of 10 is restructured
- **THEN** tier 1 keeps its 10 teams, the 20 worst of tier 2 drop to tier 3, 140 of tier 3 drop to tier 4, and 60 drop to tier 5

### Requirement: Restructure fills groups and tops up
Within a tier, teams SHALL be dealt to groups in a snake order by their position in the ordering (1..G, then G..1, and so on). When a structure has more places than teams, the bottom tier's empty places SHALL be filled with newly generated CPU teams, created like those of a new world. Teams are never deleted.

#### Scenario: Top-up
- **WHEN** 210 teams are restructured into 310 places
- **THEN** 100 new CPU teams are created in tier 5, each with a full squad, lineup and AI level

#### Scenario: Snake order in tier 2
- **WHEN** tier 2 gets ordered teams t1..t20 into 2 groups
- **THEN** group 1 gets t1, t4, t5, t8, … and group 2 gets t2, t3, t6, t7, …

### Requirement: Restructure keeps league history
A restructure SHALL keep every existing league whose tier and group number still exist, and archive every other league. An archived league keeps its past seasons, matches and history and stays reachable by id. It is no longer listed and takes no teams, fixtures or rollovers. New leagues are created for new groups, named "<tier name>.<group>".

#### Scenario: Old tier 2 groups 3 and 4
- **WHEN** a 1/4/16 country becomes 1/2/4/8/16
- **THEN** leagues II.3, II.4 and III.5..III.16 are archived, and IV.1..IV.8 and V.1..V.16 are created

### Requirement: Restructure notifications
After a restructure, the owner of every team that moved tier because of it, rather than through direct or playoff moves, SHALL receive a notification naming its new league.

#### Scenario: Pushed down by the restructure
- **WHEN** a user's team finished 6th in group II.3 and ends up in tier 3
- **THEN** the user is told the team plays next season in III.x because of the league restructure

## Code map

- Structure: `backend/common/worldconfig.go` (`DefaultWorldTiers`, `ValidateWorldConfig`,
  `ValidateWorldConfigFitsTeams`, `CountTeamsPerActiveCountry`), checked by
  `handlers.AdminSetWorldConfigHandler`; mirrored for the editor in
  `frontend/src/features/admin/pyramidRules.ts`.
- Zones: `backend/common/pyramid.go` (`PyramidZones`, `ZoneOf`, `PyramidShape` with
  `FollowsRatioRule`/`MatchesConfig`, `GetCountryPyramidShape`, `GetLeagueZones`,
  `ArchiveLeagueIfEmpty`); used by `common.GetDivisionDetails` and `common.GetTeamSeasonSituation`.
- Moves: `backend/engine/promotion.go` (`ComputeDirectMoves`, `ComputePlayoffPairings`,
  `ApplyPlayoffOutcomes`, `computeLegacyMoves`, `ComputeSeasonMoves`), applied by
  `applyPromotionRelegation` (`engine/season.go`).
- Playoff: `backend/engine/playoff.go` (`startPlayoff`, `playoffKickoff`, `playoffResults`,
  `notifyPlayoffQualified`), the gate in `advanceCountrySeasonIfComplete` (`engine/season.go`),
  `backend/common/playoffs.go` (`CreatePlayoff`, `GetCountryPlayoff`, `IsPlayoffCup`,
  `GetLastLeagueMatchDate`, `GetLeaguePlayoffTies`), `common/cups.go` (`Cup.Kind`, `CupKind*`,
  `SetCupCompletedWithoutChampion`), `engine.awardCupRoundPrize` (no prize for a playoff),
  `common.IsCupFinal` (never a playoff).
- Restructure: `backend/engine/restructure.go` (`PlanRestructure`, `restructureIfNeeded`,
  `applyRestructure`, `notifyRestructureMoved`, the `restructureTarget` hook that
  `engine/main_test.go` switches off), `backend/common/teamcreation.go` (`CreateLeague`,
  `CreateTeam`, `CreateClassification`); pending countries in `common.GetPendingRestructureCountries`
  (admin overview `pendingRestructure`).
- History: `season_history.via_playoff` (`common.RecordSeasonHistoryLeagueResult`,
  `SeasonHistoryEntry.ViaPlayoff`, `LeagueSeasonPosition.ViaPlayoff`).
- Data: `backend/resources/sql/init110_league_pyramid_playoffs.sql` (`cups.kind`,
  `cups_playoff_once`, `leagues.archived_season`, `season_history.via_playoff`, the world_config
  move to the new default, `countries.league_kickoff_minute`).
- Notifications: `notifications.messages.playoffQualified`, `promotedViaPlayoff`,
  `relegatedViaPlayoff`, `restructureMoved` (`common/notificationtranslations.go`, frontend locales).
- Tests: `backend/common/pyramid_test.go`, `common/worldconfig_test.go`,
  `common/archivedleagues_integration_test.go`, `engine/promotion_test.go`, `engine/playoff_test.go`,
  `engine/restructure_test.go`, `engine/pyramid_integration_test.go`,
  `handlers/worldConfigHandler_test.go`.
