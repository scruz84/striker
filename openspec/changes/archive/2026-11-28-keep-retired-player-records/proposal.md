# Proposal

## Why

A player who retires is deleted from `players` (`common.RetirePlayer`, and
`common.RetireFreeAgentAsTrainer` for a free agent who moves into coaching). His Golden Boots,
Player of the Season awards, transfers and club history were tied to that row with
`ON DELETE CASCADE`, so they vanished with him. Nothing in the game could look back on a career:
a league's list of Golden Boots loses its winners as they retire, and an all-time top scorer cannot
exist. Retirement also runs before the season awards in the rollover, so a top scorer who retires
that same rollover could not be found to win the Golden Boot.

## What Changes

- **Retirement snapshot.** Retiring writes a `retired_players` row in the same transaction as the
  delete: name, birth country, best position, age, last club and the season he retired in.
- **The record stays.** `player_season_awards`, `player_transfers` and `player_club_history` drop
  their foreign key to `players`, so their rows outlive the player. A world reset clears them (and
  `retired_players`) explicitly.
- **Retired profile.** `GET /v1/teams/player/:id/` answers a retired id with his snapshot and kept
  record (`retired` set) instead of 404; the player page shows it read-only.
- **Panels unchanged.** The transfer market's biggest fee, a league's recent transfers and the
  transfer comparables keep showing only players still in the game, as before.

Out of scope:
- Showing a retired player in squads, markets or search.
- Bringing back players retired before this change: their rows were already deleted.
- Stats and records built on this (see add-player-stats, add-club-and-league-records).

No open question for the owner.
