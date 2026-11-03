# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- Set-piece takers are picked by `bestOnPitchOutfielder(players, byPosition, score)` in
  `processSetPiece` (`engine/scorer_simulation.go`): Shot for penalties, `freeKickTakerValue` for
  free kicks.
- Cards: `DisciplineSimulation.Apply` (`engine/discipline_simulation.go`) rolls the incident, then
  `pickOutfieldPlayer` picks the player from either side.
- A lineup lives in five places (default, friendly default, `lineup`, `cup_tie_lineup`,
  `friendly_lineup`); the per-fixture tables also hold the kickoff snapshot the viewer reads.
- The XI may change before kickoff through the refill of unavailable players
  (`common.BackfillEmptyPositions`); a named role can therefore point at someone who is not
  playing.

## Goals / Non-Goals

**Goals:**
- Roles are hints, never failures: an absent or ineligible named player silently falls back to the
  automatic rule, so no match can break because of a role.

**Non-Goals:**
- Showing other clubs' set-piece takers (only the armband is public, after kickoff).

## Decisions

### D1. Storage
Migration `resources/sql/init97_lineup_roles.sql` (renumber if needed): nullable
`captain_id`, `penalty_taker_id`, `free_kick_taker_id` (`bigint REFERENCES players ON DELETE SET
NULL`) on `lineup`, `cup_tie_lineup` and `friendly_lineup`, and a new `default_lineup_roles
(team_id bigint REFERENCES teams ON DELETE CASCADE, friendly boolean, captain_id, penalty_taker_id,
free_kick_taker_id, PRIMARY KEY (team_id, friendly))` for the two default lineups.
`common.Lineup` gains `CaptainId`, `PenaltyTakerId`, `FreeKickTakerId` (`*int64`, JSON
`captainId` / `penaltyTakerId` / `freeKickTakerId`, omitted when nil). Reads drop a role whose
player is no longer in the team's squad ("Taker sold").
*Alternative:* six more columns on `teams`. Rejected: `teams` already carries two formations and
would carry six player references that only the lineup code reads.

### D2. Validation
`common/lineupvalidation.go`: each named role must be one of the lineup's starters or a squad
player not in the XI (the bench); takers must not have `Position == GOALKEEPER` in the lineup nor
be the squad's goalkeepers by best position (`BestCorePosition`).

### D3. Simulation
`MatchSimulation` gains `HomeRoles` / `AwayRoles` from the kickoff lineups. `processSetPiece`
first checks whether the named taker is on the pitch (`playersByPosition` contains him and he is
not a goalkeeper) and uses him, else calls `bestOnPitchOutfielder`. In `DisciplineSimulation`,
after `pickOutfieldPlayer`, if the player's side has its captain on the pitch and the player is
not the captain, a draw `< CaptainCardCancelProbability (0.10)` cancels the card. The extra draw
comes from the same seeded `s.r`, so results stay reproducible.

### D4. CPU and frontend
`PlanAiMatchLineup` sets `CaptainId` to the oldest outfield starter (ties lowest id) in every
match and leaves the takers nil. In the frontend, `SelectedPlayerPanel` gets three toggles
("Captain", "Penalties", "Free kicks"; one holder each) and the pitch shows small badges on the
owner's own editor; `Lineups.tsx` / `MatchPitchChip.tsx` show the "C" armband from the snapshot.
Keys `lineup.roles.{captain,penalties,freeKicks,automatic}` and `match.captain` in en/es.

## Risks / Trade-offs

- [Fewer cards overall shifts suspensions] → about −9% cards for sides with a captain; the
  discipline simulation test (`discipline_simulation_test.go`) is updated with the new expected
  rate.
- [A weak named taker lowers conversion] → that is the manager's choice; the automatic rule stays
  the default.

## Migration Plan

Additive migration; no role is named for existing lineups, so behaviour is unchanged until a
manager names one (CPU teams start naming a captain at their next kickoff). Rollback: redeploy the
previous engine; the columns are ignored.
