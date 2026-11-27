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
Saving a structure, global or per country, SHALL be rejected when its places (the sum of groups × teams per group) are fewer than the user-owned teams of any active country it applies to, naming the country and both numbers. Fewer places than a country's teams is allowed: the surplus CPU teams are dissolved at that country's next rollover (see "Surplus CPU teams are dissolved").

#### Scenario: Shrinking below the current team count
- **WHEN** an active country has 310 teams, 12 of them user-owned, and the admin saves 1/2/4/8 groups of 10 (150 places) and confirms
- **THEN** the structure is saved, and the country's next rollover dissolves 160 CPU teams

#### Scenario: Not enough places for the users
- **WHEN** an active country has 40 user-owned teams and the admin saves 1/2 groups of 10 (30 places)
- **THEN** the save is rejected, naming the country, 30 places and 40 user teams

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
At rollover, a country SHALL be restructured when its active leagues differ from its effective structure (its own structure if it has one, otherwise the global one) in the number of tiers, the groups of any tier, or the teams per group. Saving a structure, or setting or removing a country's own, SHALL change nothing until each affected country's next rollover. A country already in its effective structure is left alone.

#### Scenario: Old world
- **WHEN** a country has 1/4/16 groups and the saved structure is 1/2/4/8/16
- **THEN** its next rollover restructures it

#### Scenario: Country with its own structure
- **WHEN** the global structure is 1/2/4/8/16 and Andorra has its own 1/2 structure
- **THEN** Andorra's next rollover reshapes it into 2 tiers and leaves every other country on 5

#### Scenario: Removing a country's own structure
- **WHEN** the admin removes Andorra's own 1/2 structure
- **THEN** Andorra's next rollover reshapes it into the global structure, topping up with new CPU teams

### Requirement: Restructure ordering
A restructure SHALL first apply the season's direct and playoff moves. Then every team is ordered by tier after those moves, then by final position, points, goal difference and goals scored. Tier 1 takes the first G1 × N teams, tier 2 the next G2 × N, and so on. Teams that do not fit in a tier drop to the next one.

#### Scenario: From 1/4/16 to 1/2/4/8/16
- **WHEN** a country with 210 teams in 1/4/16 groups of 10 is restructured
- **THEN** tier 1 keeps its 10 teams, the 20 worst of tier 2 drop to tier 3, 140 of tier 3 drop to tier 4, and 60 drop to tier 5

### Requirement: Restructure fills groups and tops up
Within a tier, teams SHALL be dealt to groups in a snake order by their position in the ordering (1..G, then G..1, and so on). When a structure has more places than teams, the bottom tier's empty places SHALL be filled with newly generated CPU teams, created like those of a new world. When it has fewer places than teams, the teams beyond the last place are dissolved, never placed.

#### Scenario: Top-up
- **WHEN** 210 teams are restructured into 310 places
- **THEN** 100 new CPU teams are created in tier 5, each with a full squad, lineup and AI level

#### Scenario: Snake order in tier 2
- **WHEN** tier 2 gets ordered teams t1..t20 into 2 groups
- **THEN** group 1 gets t1, t4, t5, t8, … and group 2 gets t2, t3, t6, t7, …

#### Scenario: Removing the bottom tier
- **WHEN** a 1/2/4/8/16 country with 310 CPU teams is restructured into 1/2/4/8
- **THEN** the first 150 teams in the ordering are placed in tiers 1-4 and the other 160 are dissolved

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

### Requirement: Restructure leaves no team behind
After a restructure, every non-dissolved team of the country SHALL play in a league of its structure. A team planned for dissolution that is not dissolved is protected, and the lowest-ranked unprotected CPU team takes its place on the dissolution list. A team still in a league outside the structure is placed with the same ordering. A league outside the structure is then empty and archived.

#### Scenario: Club claimed during the restructure
- **WHEN** a country shrinks from 310 to 150 places and the CPU club ranked 200th gets a user before it is dissolved
- **THEN** that club plays next season in the bottom tier, the CPU club ranked 150th is dissolved instead, and every bottom-tier group has 10 teams

#### Scenario: Dissolution error
- **WHEN** dissolving one planned CPU club fails with a database error
- **THEN** that club is kept like a protected one, another CPU club is dissolved instead, and the old league it was in is archived

#### Scenario: Leftover groups from an earlier restructure
- **WHEN** a country whose structure is 1/2/4 x 10 has active leagues I.1, II.1, II.2, III.1, III.2 and III.3 with 10 teams each (no III.4)
- **THEN** it plays its season in legacy mode, and its next rollover creates III.4, deals the 30 tier-3 teams into the four groups, adds 10 new CPU teams, and leaves 1/2/4 groups of 10

### Requirement: Restructure result is checked
After a restructure, the country's active leagues SHALL be compared with its structure. If they differ, the rollover SHALL stop before the cup step, and the next rollover tick runs the restructure again. After the 3rd failed attempt in the same rollover, the rollover continues, an error naming every differing league is logged, and the country stays pending for its next rollover.

#### Scenario: Transient failure
- **WHEN** creating group III.4 fails on the first attempt and succeeds 5 minutes later
- **THEN** the second tick finishes the restructure, and the season advances once with 1/2/4 groups

#### Scenario: Persistent failure
- **WHEN** the country still differs from its structure after 3 attempts
- **THEN** the season advances anyway, every league keeps its fixtures, the log names the differing leagues, and the admin still lists the country as pending

### Requirement: Admin sees why a country is pending
The admin overview SHALL list, for each country pending a restructure, every difference between its active leagues and its structure. Each difference is one of: an extra group (tier, group, name, teams), a missing group (tier, group), a group of the wrong size (name, teams, expected), or a shape that breaks the doubling rule. The World tab SHALL show these under the pending notice.

#### Scenario: Leftover groups
- **WHEN** the admin opens the World tab while ES has 1/2/3 groups against a 1/2/4 x 10 structure
- **THEN** under the pending notice it reads, for ES, that group III.4 is missing and that the shape breaks the doubling rule

#### Scenario: Nothing pending
- **WHEN** every active country matches its structure
- **THEN** the overview's details list is empty and the World tab shows no pending notice

### Requirement: Per-country structure
An admin SHALL be able to give an active country its own full structure (tiers, names, groups, teams per group), validated by the same pyramid rules and limits as the global one, and to remove it again. The admin overview MUST show which countries have their own structure and which are pending a restructure.

#### Scenario: Invalid country structure
- **WHEN** the admin saves 1/4 groups as Andorra's own structure
- **THEN** the save is rejected with "tier 2 must have exactly 2 groups (twice tier 1)"

#### Scenario: Global change and an overridden country
- **WHEN** Andorra has its own structure and the admin saves a new global structure
- **THEN** Andorra keeps its own structure and is not marked pending by the global change

### Requirement: Confirming a structure that dissolves teams
A save, global or per country, that would leave any active country with fewer places than teams SHALL first be refused with the number of CPU teams each country would lose, and SHALL go through only when the same save is sent again with explicit confirmation. A save that dissolves nothing needs no confirmation.

#### Scenario: Admin is warned
- **WHEN** the admin saves 1/2/4/8 groups of 10 for a world whose countries have 310 teams each
- **THEN** the save is refused, listing 160 teams to dissolve in each country, and nothing changes

#### Scenario: Admin confirms
- **WHEN** the admin sends the same save with confirmation
- **THEN** the structure is saved and each country shrinks at its own next rollover

### Requirement: Surplus CPU teams are dissolved
When a restructure has fewer places than teams, the teams to dissolve SHALL be the last in the restructure ordering that are not protected. A protected team (user-owned, or due to play a cup tie) that would fall beyond the last place takes the last place of the bottom tier instead, and the lowest-ranked unprotected team placed there is dissolved in its place. Every dissolved team records the last season it played.

#### Scenario: User team at the bottom
- **WHEN** a country shrinks from 310 to 150 places and a user's team is 300th in the ordering
- **THEN** that team plays next season in the bottom tier and the 150th team in the ordering, a CPU team, is dissolved instead

#### Scenario: CPU team still in a cup
- **WHEN** a CPU team that would be dissolved still has an unplayed regional cup tie
- **THEN** it keeps a place in the bottom tier and the lowest-ranked other CPU team is dissolved instead

### Requirement: Dissolved team
A dissolved team SHALL never play, be assigned to a user, act as a CPU team, or take part in the market again. Its players become free agents, its trainer returns to the market, and its pending offers, listings, loans and unplayed friendlies end. Its past matches, standings and history MUST stay viewable, and its page MUST read "Dissolved in season N".

#### Scenario: Players freed
- **WHEN** a CPU team with 18 senior players and 3 academy prospects is dissolved
- **THEN** its 18 senior players are free agents and its 3 academy prospects retire

#### Scenario: Old result still opens
- **WHEN** a viewer opens a season-3 match a dissolved team played
- **THEN** the match shows as before, with the team's name linking to its page marked "Dissolved in season 4"

#### Scenario: Loan in progress
- **WHEN** a team is dissolved while it has a player out on loan and one in on loan
- **THEN** the player out on loan becomes a free agent at the end of his loan, and the player in on loan returns to his club at once

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
  `applyRestructure` with its replanning loop (`maxDissolveRounds`, `restructureTeams`),
  `notifyRestructureMoved`, the `restructureTarget` hook that `engine/main_test.go` switches off,
  and the `dissolveTeam`/`createLeague` hooks tests use to inject failures),
  `backend/common/teamcreation.go` (`CreateLeague`, `CreateTeam`, `CreateClassification`),
  `backend/common/dissolve.go` (`GetDissolveSkipReason`, `DissolveSkip*`).
- Result check: `backend/common/pyramid.go` (`ShapeDifference`, the `ShapeDiff*` kinds,
  `PyramidShape.Differences`, `DescribeShapeDifferences`; `MatchesConfig` is
  `len(Differences) == 0`); `restructureIfNeeded` returns the differences left, and
  `runSeasonRollover` (`engine/season.go`) retries through `rolloverState.RestructureAttempts`
  up to `maxRestructureAttempts`.
- Pending countries: `common.GetPendingRestructureCountries` (`[]PendingRestructure`), the admin
  overview's `pendingRestructure` and `pendingRestructureDetails` (`handlers.AdminOverviewHandler`),
  rendered by `describeDifference` in `frontend/src/features/admin/WorldTab.tsx` with the
  `admin.worldStructure.pendingDetail.*` strings.
- Per-country structures: `backend/common/worldconfig.go` (`EffectiveStructure`,
  `GetCountryStructure`, `GetCountryStructures`, `SetCountryStructure`, `CountTeamsPerActiveCountry`
  with user teams, `PlanStructureChange`), checked by `handlers.AdminSetWorldConfigHandler`,
  `AdminSet/Get/ClearCountryStructureHandler` and `checkStructureChange`
  (`handlers/worldAdminHandlers.go`); `AdminOverview.countryStructures`; frontend
  `features/admin/WorldTab.tsx` (`useStructureSave`, `CountryStructureDialog`, missing divisions
  against the effective structure), `features/admin/components/TierEditor.tsx`, `api/hooks.ts`
  (`useAdminSetCountryStructure`, `useAdminClearCountryStructure`, `dissolveCountsOf`),
  `ApiError.body` (`api/client.ts`), strings `admin.countryStructure.*`.
- Dissolution: `backend/engine/restructure.go` (`RankedTeam.Protected`, `cutToPlaces`,
  `RestructurePlan.Dissolve`, `applyRestructure` dissolving first), `backend/common/dissolve.go`
  (`DissolveTeam`, `GetProtectedTeams`, `IsTeamDissolved`, `getDissolvedTeamDetails`),
  `clearClubStateTx` shared with `ResetReleasedTeam` (`common/accounts.go`); club listings skipping
  `dissolved_season` (`GetAllTeamIdsStmtSql`, `GetTeamIdsForCountryStmtSql`, `GetFreeTeamStmtSql`,
  the all-teams budget adjustment, AI style backfill, AI level stats, team count, budget gauge);
  `ErrTeamDissolved` in `ChallengeToFriendly`; the team page's dissolved chip
  (`features/team/TeamDetailsPage.tsx`, string `team.dissolved`). The economy check of a shrink is
  `ECON_SIM=1 ECON_SIM_TIERS=4 go test ./engine -run TestEconomySimulation -v`.
- History: `season_history.via_playoff` (`common.RecordSeasonHistoryLeagueResult`,
  `SeasonHistoryEntry.ViaPlayoff`, `LeagueSeasonPosition.ViaPlayoff`).
- Data: `backend/resources/sql/history/init110_league_pyramid_playoffs.sql` (`cups.kind`,
  `cups_playoff_once`, `leagues.archived_season`, `season_history.via_playoff`, the world_config
  move to the new default, `countries.league_kickoff_minute`);
  `backend/resources/sql/history/init119_country_structures.sql` (`countries.structure`,
  `teams.dissolved_season`, `teams_active_clubs`).
- Notifications: `notifications.messages.playoffQualified`, `promotedViaPlayoff`,
  `relegatedViaPlayoff`, `restructureMoved` (`common/notificationtranslations.go`, frontend locales).
- Tests: `backend/common/pyramid_test.go`, `common/worldconfig_test.go`,
  `common/archivedleagues_integration_test.go`, `engine/promotion_test.go`, `engine/playoff_test.go`,
  `engine/restructure_test.go`, `engine/pyramid_integration_test.go`,
  `engine/restructure_hardening_integration_test.go`, `handlers/adminOverviewPending_test.go`,
  `frontend/src/features/admin/WorldTab.test.tsx`,
  `handlers/worldConfigHandler_test.go` (`TestAdminCountryStructure`),
  `common/countrystructure_integration_test.go`, `common/dissolve_integration_test.go`,
  `service/security_test.go`, `frontend/src/features/team/DissolvedTeamPage.test.tsx`.
