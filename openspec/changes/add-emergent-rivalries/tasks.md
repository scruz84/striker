# Tasks

## 1. Storage and levels (common)

- [ ] 1.1 Add `resources/sql/init94_team_rivalries.sql` (table `team_rivalries`, indexes, `rivalry_level` on `matches` and `cup_ties`), re-runnable; verify by applying it twice on the test DB.
- [ ] 1.2 Add `common/rivalries.go` with the `*StmtSql` constants, `AddRivalryPoints` (returns old/new score, cap 40), `GetRivalryLevel`, `GetTopRivalries`, `DecayCountryRivalries` and `RivalryLevel(score)`; verify the "Thresholds", "Cap" and "Old feud" scenarios in `rivalries_test.go` and an integration test where two goroutines add points to one pair concurrently and the sum is exact.

## 2. Earning points (engine)

- [ ] 2.1 Implement the pure `rivalryPointsForMatch` (draw/one-goal +1, margin ≥ 3 +1.5, reds +1 each max 2, cup elimination +3, final +5, ×1.5 both human) and call it from `applyPostMatchEffects` for league and cup kinds; verify every "Rivalry events" and "Human rivalry bonus" scenario in `engine/rivalries_test.go`.
- [ ] 2.2 Notify both human owners with `NOTIFICATION_RIVALRY` when the level rises (keys `rivalryRival` / `rivalryClasico`, Spanish templates in `notificationtranslations.go`); verify with an integration test that the notification exists only after `completePostMatchEffects` ran.
- [ ] 2.3 Add the title-race +4 and the ×0.6 decay to `advanceCountrySeasonIfComplete`; verify "Title race" and "Decay" (no notification on a fall) with an integration test, plus a concurrent test where two replicas advance the same country and the decay is applied once.
- [ ] 2.4 Write a calibration test that plays 5 simulated seasons of a 20-team division with realistic score lines and asserts at most 5% of pairs are Rival or above and at most 1% Clásico without cup/title events; retune the constants and this change's specs if it fails.
- [ ] 2.5 Add the key-player sale +3 to `ExecuteTransfer` and `AcceptOffer` (evaluated inside the lock, applied after commit, same country only, never for loans); verify "Key player sold to a domestic club" with an integration test.

## 3. Effects of a rivalry (engine)

- [ ] 3.1 Read the level at kickoff in `simulateMatch` / `simulateCupTie` (friendlies 0), store it on `MatchSimulation.RivalryLevel` and in `rivalry_level` via `SaveLeagueMatchResultTx` / `SaveCupTieResultTx`; verify "Level fixed at kickoff" with an integration test.
- [ ] 3.2 Apply the level to `ProbCardIncident` (3.6% / 4.2%), `rivalFactor` (×1.10 / ×1.20) and the fanbase result drift (30% / 45%); verify the "Clásico", "Clásico league match", "Friendly between rivals" and "Clásico defeat" scenarios with unit tests in `discipline_simulation_test.go`, `stadium_test.go` and `fanbase_test.go`.
- [ ] 3.3 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` before and after and record the gate medians in this change; verify no division's budget median moves by more than 2%.

## 4. API and frontend

- [ ] 4.1 Add `GET /v1/teams/team/:id/rivalries` and the `rivalry` field on `TeamMatchSummary` and the match viewer payload; verify with handler tests (a played match returns its stored level, an upcoming one the current level).
- [ ] 4.2 Add the "Rivals" section to `TeamDetailsPage` (top 3, level chip, 0–40 bar, head-to-head link) and a `RivalryChip` used in `TeamMatchesSection`, the dashboard `NextMatchHero` and `MatchScoreboard`; add en/es strings; verify with vitest tests for "Team page" and "Upcoming Clásico".

## 5. Specs and checks

- [ ] 5.1 Add the Code map to the new `rivalries` spec and update the Code maps of `socios`, `fanbase` and `match-simulation`; verify `openspec validate add-emergent-rivalries --strict` passes.
- [ ] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
