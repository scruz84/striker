# Design

## One line per player and fixture, written with the result

A running per-season counter would be updated when the post-match effects run, but those are
claimed (`DELETE ... RETURNING`) and applied outside any transaction: an engine stopping in between
would lose the increment. A line per player and fixture written by `replaceMatchEventsTx`, in the
same transaction as the result, is exactly-once like the result itself; a total is an indexed
`SUM` over at most a few thousand rows. The kickoff is stored on the line so the spoiler cutoff
needs no join (a played fixture is never rescheduled).

## Migration `init123_player_match_stats.sql`

`player_match_stats (kind, match_id, player_id PK, team_id, kickoff, season, competition,
competition_id, appeared, started, minutes, rated, rating, goals, penalty_goals, yellow_cards,
red_cards, position)`, indexes on `(player_id, season)`, `(competition, competition_id, season)` and
`(team_id, season)`. `schema_backfills (name)` marks the one-off backfill, which runs in the same
implicit transaction as the file. No TRANSACTION_* reason and no notification key.

Competitions (`common.StatsCompetition*`): 1 league, 2 domestic cup, 3 playoff, 4 regional cup,
5 World Cup (knockout tie or group friendly), 6 club friendly, 7 national-team friendly. The
season is the league's or domestic cup's own; for the rest the team's country's current season
(the backfill uses the season whose first league kickoff in that country precedes the fixture).

## Writing

`BuildPlayerMatchStatLines(ratings, events)` (pure): a line per rated player, plus goals
(`TeamOwnerPlayers[0]`, penalties by `SetPiece`) and cards; a player named by an event but not
rated gets a line without an appearance. `replacePlayerMatchStatsTx` resolves the fixture context
and inserts. A default result stores nothing.

## Reading

`playerstatsqueries.go`: `GetPlayerCareerStats`, `GetPlayerSeasonStats`, `GetLeagueLeaders`
(`LeadersGoals|Rating|Minutes|Cards`, cutoff optional, limit 0 = all), `GetTeamSeasonStats`.
`GetTopScorers*` and `GetPlayerOfTheSeason` become thin wrappers over the league totals; names come
from `players` or `retired_players`, the team from the player's latest league appearance that season.

## Code map

- `backend/resources/sql/init123_player_match_stats.sql`
- `backend/common/playerstats.go` (`BuildPlayerMatchStatLines`, `replacePlayerMatchStatsTx`, `StatsCompetition*`)
- `backend/common/playerstatsqueries.go` (career, season lines, leaders, squad stats)
- `backend/common/matchresult.go` `replaceMatchEventsTx`
- `backend/common/topscorers.go`, `backend/common/seasonawards.go`
- `backend/handlers/statsHandlers.go` `LeagueLeadersHandler`, `TeamSeasonStatsHandler`; `playerHandlers.go`
- `frontend/src/features/stats/` (`PlayerSeasonStatsSection`, `LeagueLeadersCard`, `TeamStatsSection`, `competition.ts`, `teamStatsSort.ts`)
- `frontend/src/features/division/DivisionDetailsPage.tsx`, `features/team/TeamDetailsPage.tsx`
