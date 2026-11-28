# Proposal

## Why

The game stores every result, event and rating, but a player's profile only shows goals and cards,
counted from league `match_events` alone: cup and international goals never count, and there are
no appearances, minutes or ratings. The top scorers and the season awards parse each event's
`team_players` JSON and look every scorer up one by one, and skip one who has retired. There is no
squad stats view, and a league shows its top scorers and nothing else.

## What Changes

- **Player match stats.** `player_match_stats`: one line per player and fixture (league match, cup
  tie or friendly) written in the result's own transaction - minutes, started, rating, goals,
  penalty goals, cards, position - with the fixture's kickoff, season and competition. A migration
  backfills every played fixture from its ratings and events.
- **Never early.** Every public total leaves out lines whose kickoff is after
  `LiveBroadcastCutoff()`, like every other aggregate.
- **Career totals** (appearances, minutes, goals, cards) count every competition but club
  friendlies. **Season lines** on the profile: one per season, competition and team.
- **League leaders**: goals, average rating (at least 36 minutes per match of the team that played
  the most), minutes and cards (yellow + 3 x red), for the current season or a past one.
- **Squad stats**: a team's players of one season, gone and retired included, sortable.
- **Awards from the stats**: the Golden Boot and Player of the Season read the same lines, so a
  winner who retired in that very rollover still wins.
- **Division page**: the "Scorers" phone tab becomes "Stats" (leaders); a past season shows its own
  leaders.
- New endpoints `GET /v1/teams/division/:id/leaders` and `GET /v1/teams/team/:id/stats`; the player
  profile gains `appearances`, `minutes` and `seasonStats`.

Out of scope:
- Assists: the simulation does not pick an assister yet.
- A club friendly counting in any total but its own season line.

No open question for the owner.
