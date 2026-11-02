# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- A lineup is `common.Lineup{MatchId, TeamId, Formation, Positions, Substitutions}` stored in five
  places: the default (`teams.formation` + `default_lineup_positions` +
  `default_lineup_substitutions`), the friendly default (`teams.friendly_formation` +
  `default_friendly_lineup_positions`), and the per-fixture overrides `lineup`, `cup_tie_lineup`
  and `friendly_lineup` (each with a `formation` column). The per-fixture tables are also where the
  kickoff snapshot is written (`persistResolvedLineupSnapshot` and friends), which the viewer reads.
- Lineup handlers: `handlers/lineupHandlers.go` (`SetLineupRequest`), `matchLineupHandlers.go`,
  `cupHandlers.go`, `friendlyHandlers.go`; validation in `common/lineupvalidation.go`.
- Ratings are recomputed every minute in `InitialSimulation.Apply`; chance probability in
  `chanceProbability` (`engine/scorer_simulation.go`); fatigue in `StaminaSimulation`
  (`common.FatigueMaxPenalty = 0.30`, `common/fatigue.go`).
- CPU kickoff picks go through `planAiKickoffs` → `common.PlanAiMatchLineup` and must pass the
  same validation as a human's lineup (cpu-team-management).

## Goals / Non-Goals

**Goals:**
- Mentality lives on the lineup, so every place a lineup goes (default, overrides, snapshot, CPU
  pick, national team) carries it with no special case.
- Mentality is per side and mutable during the simulation, so conditional orders can change it.

**Non-Goals:**
- A UI to see the opponent's mentality before kickoff.

## Decisions

### D1. Storage
Migration `resources/sql/init95_lineup_mentality.sql` (renumber if needed): `mentality smallint
NOT NULL DEFAULT 0 CHECK (mentality BETWEEN -1 AND 1)` on `lineup`, `cup_tie_lineup` and
`friendly_lineup`, plus `mentality` and `friendly_mentality` on `teams`. Constants
`MENTALITY_DEFENSIVE = -1`, `MENTALITY_BALANCED = 0`, `MENTALITY_ATTACKING = 1` in
`common/fmconstants.go`; `Lineup.Mentality int \`json:"mentality"\``. Every getter/setter listed in
Context reads/writes the column; `SetDefaultLineup` / `SetDefaultFriendlyLineup` gain a mentality
argument; validation rejects values outside −1..1. A missing field in a PUT means Balanced.
*Alternative:* a separate `team_tactics` table. Rejected: the per-fixture override and the
kickoff snapshot would need their own copies anyway.

### D2. Simulation
`MatchSimulation` gains `HomeMentality` / `AwayMentality` (initialised from the kickoff lineups).
- `InitialSimulation.Apply`: after the blend and the home advantage, `applyMentality(qua,
  mentality)` multiplies `Attack`, `LeftAttack`, `RightAttach` and the three defense fields by
  `MentalityAttackFactor = 0.08` / `MentalityDefenseFactor = 0.08`.
- `chanceProbability`: `+ MentalityChanceShift (0.01)` per Attacking side, `−0.01` per Defensive.
- `StaminaSimulation`: penalty `common.FatigueMaxPenalty + 0.03 × mentality` per side.
Set pieces read the same ratings, so they follow automatically.

### D3. CPU pick
`common.PickAiMentality(aiLevel, ownStrength, oppStrength int, isHome, isFriendly bool) int`, pure,
called from `planAiKickoffs` with both kickoff XIs' `lineupStrength`. A human opponent's XI is
read from the stored lineup the engine already loads for the match (never exposed to anyone).

### D4. Display
The kickoff snapshot stores the mentality; the match viewer payload (`Lineups`) exposes
`homeMentality` / `awayMentality`; `features/match/Lineups.tsx` shows a label. `MatchLineupForm`
gets a `ToggleButtonGroup` (Defensive / Balanced / Attacking) above the pitch, saved with the
formation. Keys `lineup.mentality.{label,defensive,balanced,attacking}` and
`match.mentality.{defensive,balanced,attacking}` in en/es.

## Risks / Trade-offs

- [Attacking is strictly better for strong teams] → it costs goals against and fatigue; the
  calibration test checks both directions.
- [CPU reads a human's lineup strength] → it only uses it to choose a mentality at kickoff,
  inside the engine, the same information the simulation itself uses; nothing is returned to a
  client.

## Migration Plan

Additive migration with default 0, so every existing lineup is Balanced and plays as today.
Rollback: redeploy the previous engine; the columns are ignored.
