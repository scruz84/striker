# Proposal

## Why

The AI level calibration added by improve-cpu-ai-foundations showed a Manager barely ahead of a
Caretaker (1.399 vs 1.360 points per match with identical squads): its fixed "second-best
formation" handicap cost it most of what its mentality and substitutions gained. It also showed the
CPU ranks formations by the sum of each starter's value at his slot, which ignores how the match
engine actually turns zones into chances and goals. The engine already exposes that model to
humans through the opponent preview (`ProjectZoneDuel`), so a Director can read it too.

## What Changes

- **Manager formation**: a Manager plays the strongest formation by XI strength (rank 0) instead
  of the second best. A Caretaker keeps the third or fourth best.
- **Expected goals projection**: `ProjectExpectedGoals` rates two XIs with the engine's own chance
  and conversion formulas over `ProjectZoneDuel`'s minute-1 zones (no randomness, no set pieces).
- **Director formation against the opponent**: at the kickoff of a league match or cup tie a
  Director rates the best XI of each of the six formations against the opponent's XI and plays
  the one with the highest expected goal difference (ties to the stronger XI).

Calibration (4,000 matches each, `AI_LEVEL_SIM=1`): Manager 1.500 vs its Caretaker's 1.271 points
per match (was 1.399 vs 1.360); Director 1.734 vs 1.041 (was 1.501 vs 1.277).

Out of scope:
- Using the projection for mentality or substitutions.
- Weighing full-backs and wingers in the squad-depth rules (signings already gain them through
  the XI-strength score).
- Showing a CPU club's formation reasoning to players.

Open questions for the owner:
- Directors become clearly tougher opponents for humans; is the gap to Caretakers (≈0.7 points
  per match with identical squads) wanted, or should Directors read the opponent with some noise?

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cpu-team-management`: "Lineup and formation" changes the Manager's rank and adds the Director's
  re-pick; new "Director formation against the opponent" requirement.

## Impact

- Backend: `common/lineup.go` (`formationRankForAiLevel`), `common/ailineup.go`
  (`AiPlanOptions.Formation`), new `engine/expectedgoals.go` and `engine/aiformation.go`,
  `engine/aikickoff.go` (second pass), `engine/ai_level_calibration_test.go`.
- No schema change. Two to three extra small reads per Director kickoff.
