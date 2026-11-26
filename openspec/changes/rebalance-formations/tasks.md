# Tasks

## 0. Prerequisite

- [ ] 0.1 Archive `improve-cpu-tactics` (`openspec archive improve-cpu-tactics`) so `cpu-team-management` "Lineup and formation" carries its text before this change's delta applies; verify `openspec validate rebalance-formations` passes afterwards

## 1. Zone model

- [ ] 1.1 Add the named constants `LineCoverWeight = 0.1`, `LateralStrikerCentralWeight = 0.5`, `LateralMidfieldCentralWeight = 0.65`, `LateralMidfieldFlankShare = 0.5`, `BackThreeWingBackCentralWeight = 0.15`, `MidfieldPassWeight = 0.3` and `AttackDribblingWeight = 0.25` in `engine/initial_simulation.go`, each with a doc comment pointing to the "Formation balance" and "Squad fit" requirements; verify `go build ./...`
- [ ] 1.2 Rewrite `calculateQua` to build weighted central totals (own line weight 1, line cover, lateral strikers, lateral midfielders, back-three wing-backs) and the shared flank midfield, and change `blendLine` to take a float weight sum; verify with new `engine/initial_simulation_test.go` cases for the "Identical 4-4-2 and 4-3-3" (110/185/110 and 115/165/115), "Back-three wing-backs" and "Flank midfield" scenarios
- [ ] 1.3 Add the crowded-back-line flank term (0.15 × central average × (W ÷ 2 − 1)) to the defense line after blending; verify with a unit test for the "Crowded back line" scenario (54.1 / 50.8 / 48.1)
- [ ] 1.4 Use the midfield value (0.7 × Midfield + 0.3 × Pass) and the attacking value (0.75 × Shot + 0.25 × Dribbling) in the central lines and in `lateralAttackValue`; verify with a unit test that +10 Pass raises a central midfielder's line and +10 Dribbling a striker's
- [ ] 1.5 Update `newSideDuelists`/`sideContributions` so each player's contribution is his weighted value in every duel he feeds and the rival side uses weighted per-player averages; verify `go test ./engine -run 'PlayerContribution|ZoneDuel'` and fix the expected figures in `engine/player_contributions_test.go` and `engine/zoneduel_test.go`
- [ ] 1.6 Verify that the back-three flank weights (35/65 for defense, 65/35 for midfield) and the red-card cases (empty central line, missing full-back) still behave as before with the existing `engine/initial_simulation_test.go` and `engine/discipline_simulation_test.go` tests
- [ ] 1.7 Update `openspec/specs/match-simulation/spec.md` "Code map" (stages bullet: the new constants, the weighted `blendLine` and the crowded-back-line term)

## 2. Shot takers

- [ ] 2.1 Make `getPlayerForChance` draw the 70/20/10 share from the striker pool (STRIKER plus lateral strikers), the midfield pool (MIDFIELD plus lateral midfielders) and the defense pool (DEFENSE plus full-backs), keeping the fallbacks; verify with a seeded test that a 4-3-3's three forwards each take about 23.3% of open-play chances
- [ ] 2.2 Change `shooterMix` in `engine/expectedgoals.go` the same way; verify the existing `engine/expectedgoals_test.go` cases ("Projection matches the engine") still pass with updated figures

## 3. Slot weights

- [ ] 3.1 Add `common/slotweights.go` with `SlotWeights` (the table in the player-development "Slot weights" scenario) and rewrite `PositionValue` on top of it, keeping goalkeeper = GoalKeeper; verify with a table test of every slot on a fixed player
- [ ] 3.2 Make `GetMidfieldValue`, `GetStrikerValue` and the `GetLateral*Value` helpers in `common/fmteam.go` wrap `PositionValue`, or remove them where every caller can use `PositionValue`; verify `go build ./... && go vet ./...`
- [ ] 3.3 Generate `positionSkillSQL` (`common/aimarket.go`) from `SlotWeights` for GOALKEEPER, DEFENSE, MIDFIELD and STRIKER; verify with a new `common/aimarket_test.go` that each generated expression names the same weights as `SlotWeights` and that its integer arithmetic gives `PositionValue` for a fixed player
- [ ] 3.4 Make `driftSkillsForPosition` drift every skill whose slot weight is at least `DriftSkillMinWeight = 15`; verify `engine/progression_test.go`, updated for the "Full match as a striker" and "Position skill sets" scenarios
- [ ] 3.5 Check `BestCorePosition` and the `transfer-comparables`/`squad-list` callers with the new values; verify `common/playerdetails_test.go` and any snapshot test, updating figures only where the best position legitimately changes
- [ ] 3.6 Update `frontend/src/utils/playerRating.ts` (`midfieldValue`, `strikerValue`, the defense value) to the DEFENSE, MIDFIELD and STRIKER rows of `SlotWeights` with Go's rounding; verify with `frontend/src/utils/playerRating.test.ts` cases pinned to the same numbers as the Go table test
- [ ] 3.7 Update `openspec/specs/player-development/spec.md` "Code map" (drift: `DriftSkillMinWeight`, `common/slotweights.go`) and the comment in `common/aimarket.go`

## 4. Best XI and CPU formation ranking

- [ ] 4.1 Make `SelectBestLineupForFormation` (`common/lineup.go`) return the optimal assignment of players to the formation's slots by total `PositionValue` (exact Hungarian assignment, ties to the lower player id), leaving `BackfillEmptyPositions` greedy; verify with a common unit test for the cpu-team-management "Best XI by optimal assignment" scenario (117 against the greedy 106) and that existing `common/lineup_test.go` and `common/ailineup_test.go` cases pass or are updated where the old pick was suboptimal
- [ ] 4.2 Add `common.SetFormationScorer` and make `SelectBestFormationAndLineup` rank by the registered score against the same squad's best 4-4-2 XI (ties: `lineupStrength`, then `AllFormations` order), falling back to `lineupStrength` when none is registered; verify with common unit tests for both paths using a stub scorer
- [ ] 4.3 Register the engine scorer (`ProjectExpectedGoals`, neutral, own XI vs reference XI) from an `init()` in `engine/aiformation.go`; verify with an engine test that a Manager-level pick for a fixed squad equals the formation with the best projected difference
- [ ] 4.4 Re-run `AI_LEVEL_SIM=1 go test ./engine -run TestAiLevelCalibration -v` and record the new figures in the comment of `engine/ai_level_calibration_test.go` (prototype before the assignment change: Manager 1.446 vs 1.313, Director 1.494 vs 1.270); verify the test passes (Manager and Director still ahead)
- [ ] 4.5 Update `openspec/specs/cpu-team-management/spec.md` "Code map" (optimal assignment in `SelectBestLineupForFormation`, `SetFormationScorer`, the engine registration)

## 5. Balance and squad-fit calibration

- [ ] 5.1 Turn `engine/tactics_study_test.go` into the balance calibration: assert the "Formation balance" bounds in the `rr` and `xg` blocks (round robin within 0.06 points per match of the mean; generalist random squads within ±0.05 xG of their own 4-4-2) and the goal-volume bounds (3.0–3.4) in a 4-4-2 mirror, still only with `TACTICS_SIM=1`; verify `TACTICS_SIM=rr,xg go test ./engine -run TestTacticsStudy -v` passes
- [ ] 5.2 Add a `fit` block with the pure-specialist profiles (3 centre-backs, 3 strikers, 2 wingers, 2 full-backs at 75 and at 40) and the capped-strikers profile, asserting the "Squad fit" scenarios ("best for" at least 15% of 100 squads, and a larger edge over 4-4-2 at 75 than at 40); verify `TACTICS_SIM=fit go test ./engine -run TestTacticsStudy -v` passes
- [ ] 5.3 Add a `fitmc` block for "Five at the back with good and poor centre-backs" (150 squads × 12 matches: at least +0.08 points per match over 4-4-2 at 75, at most +0.03 at 40); verify `TACTICS_SIM=fitmc go test ./engine -run TestTacticsStudy -v` passes
- [ ] 5.4 Add a `real` block with 1,000 realistic squads (2–4 partial specialists of random types: one role skill from 60 to 80, the rest 15 lower), asserting "Realistic squads in full matches" (10 matches per squad and formation, every formation within 0.05 points per match of the 4-4-2 mirror) and "Variety across realistic squads" (no formation best for more than 30%, every formation for at least 8%); verify `TACTICS_SIM=real go test ./engine -run TestTacticsStudy -v`
- [ ] 5.5 With 4.1 in place, tune `LineCoverWeight`, `LateralStrikerCentralWeight`, `LateralMidfieldCentralWeight`, `BackThreeWingBackCentralWeight` and the crowded-back-line term until 5.1–5.4 all pass together, narrowing the gap between fit gains (prototype: 4-3-3 +0.13 xG with two wingers, 5-3-2 +0.02 with three centre-backs) and raising 4-5-1 (18% best with no strikers); record the chosen values and figures in the constants' doc comments and in this change's proposal, or record which bound cannot be met and ask the owner before relaxing it
- [ ] 5.6 Re-run `HOME_ADV_SIM=1 go test ./engine -run TestHomeAdvantageCalibration -v`; verify it passes, or retune `HomeAdvantageBase`/`HomeAdvantageCrowdWeight` and record why
- [ ] 5.7 Add the calibration command and its blocks (`rr`, `xg`, `fit`, `fitmc`, `real`) to `openspec/specs/match-simulation/spec.md` "Code map", next to the home advantage calibration

## 6. Integration

- [ ] 6.1 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify both are green
- [ ] 6.2 Run `openspec validate rebalance-formations --strict`; verify it passes

## Workflow follow-up

- Archive the change once it is merged and the calibrations are recorded.
