# player-stats Specification

## Purpose
The per-player, per-fixture match stats stored with every result, and what is totalled from them:
career totals and season lines on a profile, league leaders, squad stats and the season awards -
never before a match's live replay ends.

## Requirements

### Requirement: Match stats stored with the result
Every played league match, cup tie and friendly SHALL store, in its result transaction, one line per player who featured or scored or was booked: team, kickoff, season, competition, started, minutes, rating, goals, penalty goals, yellow and red cards and position. A default result stores none. A retried save MUST replace the lines of an earlier attempt.

#### Scenario: A league match
- **WHEN** a match is saved where A1 scores a penalty and a goal in 90 minutes and A2 comes on at 70
- **THEN** A1's line reads 2 goals, 1 penalty, started, and A2's line reads not started

### Requirement: Competition and season of a line
A line's competition SHALL be 1 league, 2 domestic cup, 3 promotion playoff, 4 regional cup, 5 World Cup, 6 club friendly or 7 national-team friendly. Its season is the league's or domestic cup's own; otherwise the current season of the player's team's country at kickoff.

#### Scenario: A club friendly
- **WHEN** a friendly between two clubs is saved while their country is in season 3
- **THEN** its lines are competition 6, season 3

### Requirement: Stats never show early
Every public total SHALL leave out lines of a fixture whose kickoff is after the live-broadcast cutoff (kickoff + 2 hours). The season awards count every line of the season.

#### Scenario: Match still on air
- **WHEN** a player scores in a match that kicked off 30 minutes ago
- **THEN** his career goals, season lines and the league leaders do not count it yet, and the season-end Golden Boot would

### Requirement: Career totals
A player's profile SHALL show appearances, minutes, goals and cards over every competition except club friendlies, plus goals for his current team and caps as before.

#### Scenario: Friendly goals left out
- **WHEN** a player scored 2 league goals and 1 goal in a club friendly
- **THEN** his career goals read 2

### Requirement: Season lines on the profile
A player's profile SHALL list one line per season, competition and team, newest season first then by competition: appearances (starts), minutes, goals, cards and average rating of his rated appearances (two decimals; a dash without one).

#### Scenario: League and friendly in one season
- **WHEN** in season 3 a player played one league match rated 8.0 with 2 goals and one club friendly with 1 goal
- **THEN** his profile lists season 3 league (1 app, 2 goals, 8.0) then season 3 friendlies (1 goal)

### Requirement: League leaders
A league season SHALL rank its league matches' players by goals, average rating, minutes or cards (yellow + 3 x red, then red), ties to the lower player id, top 10. The rating list only holds players with at least 36 minutes per match of the team that played the most. Each row shows the team of his latest appearance.

#### Scenario: Rating threshold
- **WHEN** after one matchday A1 played 90 minutes rated 8.0, B1 90 minutes rated 5.5 and A2 20 minutes rated 6.5
- **THEN** the rating leaders read A1, B1, and A2 is left out

### Requirement: Squad stats
A team SHALL offer its players' lines for one season (its newest by default) over every competition but club friendlies, including players who have left or retired, marked as such, and the list of seasons it has lines for. The squad tab switches between the squad and these stats, sortable by minutes, appearances, goals, rating or cards.

#### Scenario: A retired scorer
- **WHEN** a team's top scorer of season 3 retired
- **THEN** its season 3 stats still list him, marked retired

## Code map

- `backend/resources/sql/init123_player_match_stats.sql`
- `backend/common/playerstats.go` (`BuildPlayerMatchStatLines`, `replacePlayerMatchStatsTx`, `StatsCompetition*`)
- `backend/common/playerstatsqueries.go` (career, season lines, leaders, squad stats)
- `backend/common/matchresult.go` `replaceMatchEventsTx`
- `backend/common/topscorers.go`, `backend/common/seasonawards.go`
- `backend/handlers/statsHandlers.go` `LeagueLeadersHandler`, `TeamSeasonStatsHandler`; `playerHandlers.go`
- `frontend/src/features/stats/` (`PlayerSeasonStatsSection`, `LeagueLeadersCard`, `TeamStatsSection`, `competition.ts`, `teamStatsSort.ts`)
- `frontend/src/features/division/DivisionDetailsPage.tsx`, `features/team/TeamDetailsPage.tsx`
