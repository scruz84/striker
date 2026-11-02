# Tasks

## 1. Crowd before kickoff (engine)

- [x] 1.1 Split `stadiumAttendanceAndRevenue` into `drawMatchdayCrowd` and crediting from a drawn `matchdayCrowd`; adapt `creditTicketRevenue`, `creditFriendlyTicketRevenue` and `UpdateTeamFinancesAfterMatch`; verify the existing `stadium_test.go` and `recommended_ticket_price_integration_test.go` still pass.
- [x] 1.2 Draw weather and crowd before `runMatchSimulation` in `simulateMatch`, `simulateCupTie` and `simulateFriendly`, credit the same crowd after the save commits and store it in the report flavor; verify the "Crowd drawn before kickoff" and "Retried after a failed save" scenarios with an integration test (failed save → no ticket revenue row).

## 2. Home advantage (engine)

- [x] 2.1 Add `HomeAdvantage` to `common.MatchSimulation`, `HomeAdvantageBase` (0.04, retuned to 0.08 in 2.2), `HomeAdvantageCrowdWeight` (0.06, retuned to 0.14 in 2.2) and `homeAdvantageMultiplier`, applied to the nine home zone ratings after `blendLine` in `InitialSimulation.Apply`; World Cup fixtures pass 1; verify "Full stadium", "Empty stadium", "Half-full stadium" and "World Cup group match" with unit tests in `initial_simulation_test.go` / `simulation_test.go`.
- [x] 2.2 Add a calibration test (skipped unless `HOME_ADV_SIM=1`) simulating 20,000 league matches between identical squads at fill 0.5 and asserting the "Expected effect between even sides" ranges; retune the constants and this change's specs if it misses, and record the measured rates here.
  - Measured (20,000 matches, identical 4-4-2 squads, seed 7): with no advantage home 37.5% / draw 24.7% / away 37.7%, goals 1.41–1.40. With the first constants (0.04 / 0.06, ×1.07 half full) home 41.6% / draw 24.4% / away 33.9%, goal ratio 1.13: **missed** all three ranges. A 6,000-match scan gave ×1.10 → 42.2/25.3/32.5, ×1.14 → 45.3/24.6/30.2, ×1.16 → 46.7/23.6/29.7. Retuned to 0.08 / 0.14 (×1.08 empty, ×1.15 half full, ×1.22 full): home **45.5%** / draw 24.1% / away **30.4%**, goals 1.65–1.31, ratio **1.26**, inside every range. Proposal, design and the match-simulation and fanbase deltas updated to the new numbers.
- [x] 2.3 Verify the match viewer parity test (`match_viewer_parity_integration_test.go`) still matches stored instants, since they now include the multiplier.

## 3. Economy and specs

- [x] 3.1 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` before and after; record prize-money and budget medians here and verify no division drifts towards insolvency.
  - The economy simulation does not run the match engine: its `playMatch` draws goals from its own Poisson model, which already gives the home side 1.70 vs 1.35 expected goals (ratio 1.26, the same as the calibrated engine now produces). Before and after therefore differ only by run-to-run noise. Shipped scenario median team budget after season 36: 643,843 before / 637,163 after (season 30: 686,656 / 692,526); every class and tier keeps an S2-N income/expense ratio of at least 0.95 in both runs (lowest: AI-caretaker T3 and human-optimal T3, 0.95), so no division drifts towards insolvency. The simulation has no separate prize-money column; cup prizes are in the income totals above.
- [x] 3.2 Update the Code maps of `match-simulation` (home advantage, crowd draw), `socios` and `fanbase`; no en/es locale or notification template is needed (verify with `git diff --stat` that none changed); verify `openspec validate add-home-advantage --strict` passes.
- [ ] 3.3 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
