# Tasks

## 1. Crowd before kickoff (engine)

- [ ] 1.1 Split `stadiumAttendanceAndRevenue` into `drawMatchdayCrowd` and crediting from a drawn `matchdayCrowd`; adapt `creditTicketRevenue`, `creditFriendlyTicketRevenue` and `UpdateTeamFinancesAfterMatch`; verify the existing `stadium_test.go` and `recommended_ticket_price_integration_test.go` still pass.
- [ ] 1.2 Draw weather and crowd before `runMatchSimulation` in `simulateMatch`, `simulateCupTie` and `simulateFriendly`, credit the same crowd after the save commits and store it in the report flavor; verify the "Crowd drawn before kickoff" and "Retried after a failed save" scenarios with an integration test (failed save → no ticket revenue row).

## 2. Home advantage (engine)

- [ ] 2.1 Add `HomeAdvantage` to `common.MatchSimulation`, `HomeAdvantageBase = 0.04`, `HomeAdvantageCrowdWeight = 0.06` and `homeAdvantageMultiplier`, applied to the nine home zone ratings after `blendLine` in `InitialSimulation.Apply`; World Cup fixtures pass 1; verify "Full stadium", "Empty stadium", "Half-full stadium" and "World Cup group match" with unit tests in `initial_simulation_test.go` / `simulation_test.go`.
- [ ] 2.2 Add a calibration test (skipped unless `HOME_ADV_SIM=1`) simulating 20,000 league matches between identical squads at fill 0.5 and asserting the "Expected effect between even sides" ranges; retune the constants and this change's specs if it misses, and record the measured rates here.
- [ ] 2.3 Verify the match viewer parity test (`match_viewer_parity_integration_test.go`) still matches stored instants, since they now include the multiplier.

## 3. Economy and specs

- [ ] 3.1 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` before and after; record prize-money and budget medians here and verify no division drifts towards insolvency.
- [ ] 3.2 Update the Code maps of `match-simulation` (home advantage, crowd draw), `socios` and `fanbase`; no en/es locale or notification template is needed (verify with `git diff --stat` that none changed); verify `openspec validate add-home-advantage --strict` passes.
- [ ] 3.3 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
