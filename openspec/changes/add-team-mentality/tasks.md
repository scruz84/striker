# Tasks

## 1. Storage and API

- [x] 1.1 Add `resources/sql/init95_lineup_mentality.sql` (columns on `lineup`, `cup_tie_lineup`, `friendly_lineup`, `teams.mentality`, `teams.friendly_mentality`), re-runnable; verify by applying it twice on the test DB.
- [x] 1.2 Add `MENTALITY_*` constants and `Lineup.Mentality`; read/write it in every lineup getter/setter and kickoff snapshot; validate −1..1; verify with integration tests that each of the five lineup kinds round-trips its mentality and "Lineup saved before this change" reads Balanced.
- [x] 1.3 Accept and return `mentality` in all lineup handlers (default, friendly default, match, cup tie, friendly); verify with handler tests, including a PUT without the field saving Balanced.

## 2. Simulation (engine)

- [x] 2.1 Add `HomeMentality` / `AwayMentality` to `MatchSimulation`, `applyMentality` in `InitialSimulation.Apply` (after the home advantage, if present), the ±0.01 shift in `chanceProbability` and the fatigue penalty `0.30 + 0.03 × mentality`; verify "Attacking side", "Two Balanced sides", "Two Attacking sides", "Attacking against Defensive" and "Attacking side at full time" with unit tests.
- [x] 2.2 Verify that two Balanced sides produce exactly the same events as before for a fixed seed (golden test in `simulation_test.go`).
  - The chance draw in `ScorerSimulation.Apply` uses the process-wide random source, which can't be seeded, so the golden check (`TestBalancedSidesPlayExactlyAsBefore` in `engine/mentality_test.go`) compares everything the draws depend on: for all 90 minutes of a match between two Balanced sides, the fatigued players (old `FatigueMultiplier`), both sides' zone ratings (old pipeline plus home advantage) and the chance probability are bit-identical to the pre-mentality formulas.
- [x] 2.3 Add a calibration test (skipped unless `MENTALITY_SIM=1`) for "Calibration sample" in both directions; retune the constants and this change's specs if it misses, and record the measured rates here.
  - Measured (20,000 matches per mentality, identical 4-4-2 squads on neutral ground, seed 11): Balanced vs Balanced 1.408 - 1.398 goals. Attacking vs Balanced 1.569 - 1.569 (goals for and against **x1.118**); Defensive vs Balanced 1.245 - 1.255 (for **x0.887**, against **x0.894**). Inside the 8-20% ranges with the original constants (±0.08, ±0.01, ±0.03), so nothing was retuned.

## 3. CPU mentality

- [x] 3.1 Implement `common.PickAiMentality` and call it from `planAiKickoffs`; verify "Manager favourite", "Director away underdog" and "Caretaker" in `ailineup_test.go`, and that friendlies are Balanced.

## 4. Frontend

- [x] 4.1 Add the mentality selector to `MatchLineupForm` (saved with the lineup in every editor that uses it) and the `mentality` field to the lineup types; add en/es strings; verify with a vitest test that changing it marks the lineup dirty and sends it on save.
- [x] 4.2 Show each side's mentality in `features/match/Lineups.tsx` for live and finished views only; verify "Finished match" and "Before kickoff" with vitest tests.

## 5. Specs and checks

- [x] 5.1 Update the Code maps of `match-simulation`, `cpu-team-management` and `match-viewer`; verify `openspec validate add-team-mentality --strict` passes.
- [x] 5.2 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` and verify CPU budgets do not drift (CPU mentality changes results).
  - The economy simulation draws results from its own Poisson model (`playMatch` in `engine/economysim_test.go`), not the match engine, so mentality can't move it: median team budget after season 36 is 637,981 / 970,554 / 716,295 in its three scenarios, against 629,038 / 985,333 / 727,874 on `main` before this change (run-to-run noise).
- [x] 5.3 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
