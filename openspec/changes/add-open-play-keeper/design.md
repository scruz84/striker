# Design

## Context

- `processChance` already reads the defending keeper's rating via
  `(*match.<Side>PlayersByPosition)[common.GOALKEEPER][0]`. It only stores the attacker.
  `processSetPiece` already stores `goalKeeperId` and the keeper's name.
- `replaceMatchEventsTx` takes `withOtherTeam`, true only for `match_events`. The cup and
  friendly insert statements and their getters skip the two columns, and the tables don't have
  them.

## Decisions

1. **Same keeper lookup as the rating.** Use the id at `GOALKEEPER[0]` of the defending side,
   the player whose `GoalKeeper` rating defends the chance, so the name always matches the maths.
   Get the name with `GetPlayerFullName` like the attacker. A side whose keeper slot is empty
   cannot reach here: the forfeit check stops the match first. If the slot is ever empty, store
   no keeper, not a wrong id.
2. **Migration `init108_event_other_team_players.sql`.** Re-runnable:
   `ALTER TABLE cup_tie_events ADD COLUMN IF NOT EXISTS other_team_players varchar(200)` and
   `other_team_player_names varchar(1000)`, and the same for `friendly_events`. Both are
   nullable, matching `match_events`. No backfill.
3. **Drop `withOtherTeam`.** All three insert statements take the same 15 parameters in the same
   order, so `replaceMatchEventsTx` has one code path. `GetCupTieEvents` and `GetFriendlyEvents`
   read the two columns with `sql.NullString` (old rows are NULL) and unmarshal them like
   `GetMatchEvents` does.
4. **No new TRANSACTION_* reason or notification key.** Nothing about money or notifications
   changes.

## Risks / Trade-offs

- [One extra name lookup per open-play chance, about 25 per match] → `GetPlayerFullName` is the
  same cheap lookup already made for the attacker, so cost goes from 1 to 2 per chance.
- [Replicas running old code during a rolling deploy write rows without the columns] → the
  columns are nullable, so old writers keep working and those rows read as "no keeper".

## Migration Plan

`EnsureSchema` applies `init108` on the next `--svc` / `--engine` start. Rollback: revert the
code. The extra columns are harmless if left in place.
