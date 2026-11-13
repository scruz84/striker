# Proposal

## Why

The commentary can name the goalkeeper only on penalties and free kicks ("¡{{keeper}} detiene
el lanzamiento!"). Open-play chances, which are most of a match, don't record who was in goal,
so a save can only say "el portero". Cup ties and friendlies don't store the other side's
players at all. Their set-piece lines read "the goalkeeper" in the finished report even though
the live broadcast named the keeper. Recording the defending keeper on every chance, in every
competition, lets `improve-match-commentary` name the keeper on saves and goals and keeps the
live and finished chronicles the same.

## What Changes

- Every open-play GOAL and NO_GOAL event names the defending side's goalkeeper on the pitch as
  the other team's player, the same way set-piece events already do.
- Cup ties and friendlies store and return the other team's players and names on their events,
  like league matches. A new migration adds the two columns to `cup_tie_events` and
  `friendly_events`.
- Events stored before this change keep an empty keeper. The commentary falls back to "the
  goalkeeper" for them, as it already does.

### Out of scope

- Any change to how chances are resolved. The keeper's rating already counts; this only
  records his name.
- Backfilling old events.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `match-simulation`: chance events record the defending goalkeeper for open play too, and
  every competition stores the other team's players on its events.

## Impact

- Backend: `engine/scorer_simulation.go` (`processChance`), `common/cups.go` and
  `common/friendlies.go` (event insert and read SQL), `common/matchresult.go`
  (`replaceMatchEventsTx` always writes the other team's players).
- Schema: `resources/sql/init108_event_other_team_players.sql`.
- API: the cup tie and friendly event payloads now fill `otherTeamPlayers` /
  `otherTeamPlayerNames`. The fields already exist, so nothing breaks.
- Frontend: nothing in this change. `improve-match-commentary` uses the keeper.
