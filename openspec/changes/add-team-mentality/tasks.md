# Tasks

## 1. Storage and API

- [ ] 1.1 Add `resources/sql/init95_lineup_mentality.sql` (columns on `lineup`, `cup_tie_lineup`, `friendly_lineup`, `teams.mentality`, `teams.friendly_mentality`), re-runnable; verify by applying it twice on the test DB.
- [ ] 1.2 Add `MENTALITY_*` constants and `Lineup.Mentality`; read/write it in every lineup getter/setter and kickoff snapshot; validate −1..1; verify with integration tests that each of the five lineup kinds round-trips its mentality and "Lineup saved before this change" reads Balanced.
- [ ] 1.3 Accept and return `mentality` in all lineup handlers (default, friendly default, match, cup tie, friendly); verify with handler tests, including a PUT without the field saving Balanced.

## 2. Simulation (engine)

- [ ] 2.1 Add `HomeMentality` / `AwayMentality` to `MatchSimulation`, `applyMentality` in `InitialSimulation.Apply` (after the home advantage, if present), the ±0.01 shift in `chanceProbability` and the fatigue penalty `0.30 + 0.03 × mentality`; verify "Attacking side", "Two Balanced sides", "Two Attacking sides", "Attacking against Defensive" and "Attacking side at full time" with unit tests.
- [ ] 2.2 Verify that two Balanced sides produce exactly the same events as before for a fixed seed (golden test in `simulation_test.go`).
- [ ] 2.3 Add a calibration test (skipped unless `MENTALITY_SIM=1`) for "Calibration sample" in both directions; retune the constants and this change's specs if it misses, and record the measured rates here.

## 3. CPU mentality

- [ ] 3.1 Implement `common.PickAiMentality` and call it from `planAiKickoffs`; verify "Manager favourite", "Director away underdog" and "Caretaker" in `ailineup_test.go`, and that friendlies are Balanced.

## 4. Frontend

- [ ] 4.1 Add the mentality selector to `MatchLineupForm` (saved with the lineup in every editor that uses it) and the `mentality` field to the lineup types; add en/es strings; verify with a vitest test that changing it marks the lineup dirty and sends it on save.
- [ ] 4.2 Show each side's mentality in `features/match/Lineups.tsx` for live and finished views only; verify "Finished match" and "Before kickoff" with vitest tests.

## 5. Specs and checks

- [ ] 5.1 Update the Code maps of `match-simulation`, `cpu-team-management` and `match-viewer`; verify `openspec validate add-team-mentality --strict` passes.
- [ ] 5.2 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` and verify CPU budgets do not drift (CPU mentality changes results).
- [ ] 5.3 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
