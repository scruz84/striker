# Design

## Context

`ScorerSimulation.Apply` (`backend/engine/scorer_simulation.go`) runs once per minute. It rolls
`chanceProbability` (14%, plus up to 10 points for midfield dominance), picks the side with
`getTeamChance`, and resolves the chance in `processChance`: zone, shooter, clear-cut roll, then
the logistic `chanceSuccessProbability`. The outcome is a `common.Event` of type `GOAL` (0) or
`NO_GOAL` (1).

Events are stored in three tables with the same shape: `match_events` (league,
`engine/database.go` and `common/database.go`), `cup_tie_events` (cups, including the World Cup,
`common/cups.go`) and `friendly_events` (`common/friendlies.go`). The live replay broadcasts the
in-memory `common.Event` structs as JSON (`engine/livematch.go`). Many readers filter on
`event_type = GOAL`: top scorers, season awards, player details stats, match ratings, and the
frontend's `goalScorers` / `MatchEventsReport`.

By hand, an even match currently gives about 12.6 chances × ~0.236 average conversion (20%
clear-cut at ~0.42, 80% half-chances at ~0.19), or about 3.0 goals per match.

## Goals / Non-Goals

**Goals:**
- Add penalties and direct free kicks without changing any reader that counts goals.
- Keep goals per match where they are today: set pieces are carved out of the existing chance
  volume.
- Keep the new constants together in one place, so the owner can tune them later (for example,
  the penalty baseline, fixed at 78% by the owner).

**Non-Goals:**
- A card for the penalty foul, manager-designated takers, or shootouts (see the proposal's out of
  scope).
- Changing the open-play model or its constants.

## Decisions

### 1. Set piece is an attribute of GOAL / NO_GOAL, not a new event type

Add `SetPiece int` (`json:"setPiece"`) to `common.Event` and a `set_piece smallint NOT NULL
DEFAULT 0` column to the three event tables. Add constants `SET_PIECE_NONE = 0`,
`SET_PIECE_PENALTY = 1` and `SET_PIECE_DIRECT_FREE_KICK = 2` to `common/fmconstants.go`.

*Alternative considered:* new event types (`PENALTY_GOAL`, `PENALTY_MISS`, `FREE_KICK_GOAL`,
`FREE_KICK_MISS`). We rejected it because every `event_type = GOAL` reader would need to change:
`topscorers.go`, `seasonawards.go`, `playerdetails.go`, `matchratings.go`, `matchGoals.ts`,
`MatchEventsReport.tsx` and `liveMatchRatings.ts`. Missing one would quietly undercount goals.
With an attribute, no reader can undercount, and old rows read as open play through the column
default.

### 2. Set pieces replace chances instead of adding to them

In `processChance`, before the zone roll, a new `classifyChance(r)` returns the set-piece kind
(r < `ProbPenaltyPerChance` = 0.022 → penalty; r < 0.022 + `ProbDirectFreeKickPerChance` = 0.087
→ free kick). Penalties and free kicks are then handled by `resolvePenalty` and
`resolveDirectFreeKick`; open play keeps today's path unchanged.

*Alternative considered:* a separate per-minute set-piece roll, independent of chances. We
rejected it because it adds about 0.27 goals per match on top of today's rate, and it would ignore
which side is on top. Carving the set pieces out of existing chances adds about +0.01 goals per
match (see the spec scenario "Total goals unchanged"), and the dominant side wins more penalties
for free.

### 3. Automatic taker selection from effective skills

- Penalty: the highest `Shot` among the on-pitch outfield players (`EffectiveHomePlayers` /
  `EffectiveAwayPlayers` over `*PlayersByPosition`, excluding `GOALKEEPER`). Ties go to the
  lowest id.
- Free kick: the highest `0.6 × Shot + 0.4 × Pass`, with the same tie rule.

Picking the taker deterministically means real squad changes matter: losing your best finisher to
a red card or a substitution changes who takes the next penalty. Using fatigue-adjusted copies
keeps this consistent with every other stage. A helper `bestOnPitchOutfielder(players,
byPosition, score func(Player) float32) int64` serves both cases and returns -1 when nobody is
left. In that case the chance is discarded, like open play.

### 4. Conversion: linear and clamped, not logistic

Penalty: `clamp(0.78 + 0.30 × (Shot − GK) / 99, 0.60, 0.92)`. Free kick:
`clamp(0.06 + 0.10 × (value − GK) / 99, 0.02, 0.15)`. The `GoalKeeper` value comes from the
defending side's effective goalkeeper, the same one `processChance` already reads.

Real penalty and free-kick conversion rates fall in narrow bands, so a clamped line is easier to
reason about and tune than another logistic curve. Neither path applies the clear-cut roll; it
has no meaning for a dead ball.

Constants (`scorer_simulation.go`): `ProbPenaltyPerChance = 0.022`,
`ProbDirectFreeKickPerChance = 0.065`, `PenaltyConversionBaseline = 0.78`,
`PenaltySkillWeight = 0.30`, `PenaltyConversionMin = 0.60`, `PenaltyConversionMax = 0.92`,
`FreeKickConversionBaseline = 0.06`, `FreeKickSkillWeight = 0.10`, `FreeKickConversionMin =
0.02`, `FreeKickConversionMax = 0.15`, `FreeKickShotWeight = 0.6`, `FreeKickPassWeight = 0.4`.

### 5. Event payload

A set-piece event carries `TeamOwnerPlayers = [taker]`, `OtherTeamPlayers = [goalkeeper]` (and
the names), and `SetPiece` set to the kind. Open-play events keep `OtherTeamPlayers = nil`, as
today. `ComputeMatchRatings` only reads `TeamOwnerPlayers`, so the keeper is not affected. A
penalty save does not give the keeper a rating bonus in this change.

### 6. Persistence and reads

- New migration `backend/resources/sql/init82_match_event_set_piece.sql`:
  `ALTER TABLE {match_events,cup_tie_events,friendly_events} ADD COLUMN IF NOT EXISTS set_piece
  smallint NOT NULL DEFAULT 0;`. It can be re-run and needs no backfill.
- `InsertMatchEventStmtSql`, `InsertCupTieEventStmtSql` and `InsertFriendlyEventStmtSql` gain the
  column. Their `Exec` calls pass `e.SetPiece`.
- `GetWeekMatchesEventsByIdStmtSql`, `GetCupTieEventsStmtSql` and `GetFriendlyEventsStmtSql`
  select and scan it.
- No new `TRANSACTION_*` reason and no new notification key: set pieces don't move money or send
  notifications.

### 7. Frontend

- `api/types.ts`: `MatchEvent.setPiece?: number` (optional, so cached or old payloads still type
  check), plus `SET_PIECE_PENALTY = 1` and `SET_PIECE_DIRECT_FREE_KICK = 2`.
- `utils/matchCommentary.tsx`: pick the commentary key by `(eventType, setPiece)`. New keys are
  `match.commentary.penaltyGoal`, `penaltyMiss`, `freeKickGoal` and `freeKickMiss`, each a variant
  array with a `{{keeper}}` placeholder that links to `otherTeamPlayers[0]`.
- `utils/matchGoals.ts`: `ScorerLine.minutes: number[]` becomes `goals: { minute: number;
  setPiece: number }[]`. The scoreboard renders the suffix from
  `match.setPiece.penaltyShort` / `freeKickShort` ("pen." / "f.k."; "p." / "f.d." in Spanish).
- `MatchEventsReport.tsx`: set-piece goals are emphasized like goals today. A missed penalty is
  always notable, so it shows even when full commentary is off, because a missed penalty is a key
  moment. Missed free kicks stay hidden like other `NO_GOAL` events.

## Risks / Trade-offs

- [Real conversion is higher in today's top leagues (~87–90%)] → The baseline is one constant;
  the owner chose 78%. Moving it to 0.88 later would add about 0.03 goals per match.
- [Best-Shot taker plus a typical keeper converts slightly above 78%] → This is intended (good
  teams have good takers). The expected effect stays within the ±0.05 goals-per-match band in the
  spec, and the calibration test below enforces it.
- [Penalties are a coin flip on match outcomes, which managers may find frustrating] → At about
  0.28 per match, roughly one league match in four has one, the same as in real football.
- [The `ScorerLine` shape change breaks existing tests and callers] → It is a small surface
  (`matchGoals.test.ts` and the scoreboard component); the tasks update both.
- [Old cached live-match payloads have no `setPiece`] → The frontend treats a missing value as 0.

## Migration Plan

1. Deploy the migration (additive, default 0). The old engine ignores the column.
2. Deploy the backend. New events carry `set_piece`, and old rows read as 0.
3. Deploy the frontend. It tolerates a missing `setPiece`, so the deploy order is flexible.

Rollback: revert the backend and frontend. The column can stay, because it is harmless with
default 0.

## Open Questions

- Should a penalty save earn the goalkeeper a match-rating bonus (for example +0.5)? This can be
  added later without changing these specs.
