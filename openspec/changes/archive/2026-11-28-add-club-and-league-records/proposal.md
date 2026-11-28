# Proposal

## Why

Clubs and leagues have a trophy cabinet and past tables, but nothing that tells a story across
seasons: a club's biggest win, its longest unbeaten run, its record signing or its all-time top
scorer; a league's champions season by season. Everything needed is already stored - fixtures,
final tables, transfers and, with add-player-stats, the player match stats.

## What Changes

- **Club records** on the team's history tab: biggest win and defeat, most goals in a match,
  record home attendance (league matches and cup ties), longest winning, unbeaten and losing league
  streaks, best finish and most points in a season, record signing and sale, all-time top 5 scorers
  and appearances in official club competitions.
- **League history page** `/division/:id/records` (linked from the division page): champions season
  by season, most titles, Golden Boots, most points in a season, biggest win, highest-scoring match
  and all-time top scorers.
- Computed on demand, never counting a fixture still on air, cached 10 minutes.
- New endpoints `GET /v1/teams/team/:id/records` and `GET /v1/teams/division/:id/records`.

Out of scope:
- Records of national teams and of friendlies.
- A team's name as it was when it set the record (current names are shown).

No open question for the owner.
