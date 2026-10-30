# Tasks

## 1. Shared building blocks in common

- [x] 1.1 Move `engine.fatigueMultiplier` and its constants to `common.FatigueMultiplier`, have `engine/stamina_simulation.go` call it, and verify `go test ./engine -run Stamina` still passes unchanged
- [x] 1.2 Move `backfillEmptyPositions` to `common.BackfillEmptyPositions`, iterating positions in `formationSlotOrder` order; add a unit test for the "two positions need the same reserve" scenario (defence wins) and verify the existing simulation tests pass
- [x] 1.3 Move `handlers.validateLineup` / `validateSubstitutions` to `common.ValidateLineup` / `common.ValidateSubstitutions`, have the handlers call them, and verify `go test ./handlers` (incl. `lineupHandlers_test.go`) passes unchanged
- [x] 1.4 Add an `rng *rand.Rand` parameter to `formationRankForAiLevel` / `SelectBestFormationAndLineup`, update `refreshAiTeamLineup` to pass a time-seeded one, and verify `lineup_test.go` covers a fixed seed giving a stable Caretaker rank

## 2. Engine: substitutions, injuries and the kickoff snapshot

- [x] 2.1 Add `HomeSubstitutedOff` / `AwaySubstitutedOff` to `common.MatchSimulation` and an `engine.bestEligibleBenchPlayer` helper that excludes on-pitch, unavailable, sent-off, injured-this-match and substituted-off players; switch `pickInjuryReplacement` to it and add a test for "best option already came off"
- [x] 2.2 Rework `applyScheduledSubstitutions`: skip when 3 substitutions are used, skip when the outgoing player is off, replace an ineligible incoming player with `bestEligibleBenchPlayer` (skip if none), and mark the outgoing player substituted off; cover all four "Scheduled substitutions" scenarios in `substitution_simulation_test.go`
- [x] 2.3 Give `runMatchSimulation` a snapshot callback invoked right after `excludeUnavailablePlayers` with the lineup rebuilt from `*PlayersByPosition`; switch `simulateMatch`, `simulateCupTie` and `simulateFriendly` to it and drop their pre-simulation persist calls; verify with a test that an injured player in a saved lineup is absent from the stored snapshot and his replacement is in his slot
- [x] 2.4 Check that `common.ComputeMatchRatings` reads starters from the kickoff XI, not the pre-refill lineup; fix if needed and add a test that an unavailable player in the saved lineup gets no rating; verify `match_viewer_parity_integration_test.go` passes
- [x] 2.5 Update `openspec/specs/match-simulation/spec.md` Code map (`common.BackfillEmptyPositions`, `engine.bestEligibleBenchPlayer`, snapshot callback) and verify the named functions exist

## 3. CPU planner: competitive matches

- [x] 3.1 Create `common/ailineup.go` with `AiMatchPlan` and `PlanAiMatchLineup`; for competitive matches build regulars from the squad minus `leagueUnavailable` for Manager/Director and the full squad for Caretaker; unit-test the "Injured starter, Manager" and "Injured starter, Caretaker" scenarios
- [x] 3.2 Implement the fatigue gain (via `common.FatigueMultiplier` at the interval midpoint) and `planCompetitiveSubstitutions` for the three levels (Caretaker ≤1 at 70 with Stamina < 50 and ≥80% value; Manager top 2 at 70 with ≥5%; Director top 3 over 55–85 with ≥2%, third moved to ≥75); unit-test each level's scenario, "Nothing worth changing" and "no player in two pairs"
- [x] 3.3 Assert in the planner tests that every plan passes `common.ValidateLineup` / `ValidateSubstitutions` (availability enforced for Manager/Director) across 200 random squads

## 4. CPU planner: friendly rotation

- [x] 4.1 Implement `planFriendlyRotation` for Caretaker (regulars + 3 best bench at 60), Manager (rest the top-5 outfield regulars, best XI from the rest in league formation, subs from non-starters) and Director (reserve XI incl. backup GK, lowest-valued non-key regular fills gaps, subs prefer age ≤ 21); unit-test the three level scenarios
- [x] 4.2 Unit-test the "Thin squad" scenario (13 fit players → 11 + at most 2 subs) and that suspended players are eligible in friendlies

## 5. Engine wiring for CPU kickoff plans

- [x] 5.1 In `simulateMatch`, `simulateCupTie` and `simulateFriendly`, for each `isAiManagedTeam` side without a per-match override, replace its lineup and substitutions with `PlanAiMatchLineup` (rng seeded `matchId*31 + teamId`), validating and falling back to the stored lineup with a log on failure; verify with an integration test that a Director's league match snapshot carries planned substitutions and a Manager's friendly snapshot starts reserves
- [x] 5.2 Add a test that simulating the same match twice for a CPU side yields the same planned lineup (stuck-match recovery determinism)
- [x] 5.3 Add a seeded engine test over 500 matches comparing a Director side's mean goal difference with and without planned substitutions and verify it is not worse
- [x] 5.4 Update `openspec/specs/cpu-team-management/spec.md` Code map (`common/ailineup.go`, `PlanAiMatchLineup`, `planCompetitiveSubstitutions`, `planFriendlyRotation`, the kickoff hook in the three simulate functions) and verify the named functions exist

## 6. Projected replacements in the lineup editor

- [x] 6.1 Add `ProjectedReplacement` and `Lineup.ProjectedReplacements` (`omitempty`) plus `common.ProjectKickoffReplacements`; fill it in the default, friendly-default, match, cup-tie and friendly GET lineup handlers (friendly unavailability for the friendly ones); handler tests check an injured starter yields his projected replacement and the saved lineup is unchanged
- [x] 6.2 Add `projectedReplacements` to the frontend lineup types and render "Replaced by {name}" next to the unavailability badge in `features/lineup` (`PlayerChip`/`UnavailabilityBadge`); add a vitest case for the label
- [x] 6.3 Add `lineup.replacedBy` to `frontend/src/i18n/locales/en.json` ("Replaced by {{name}}") and `es.json` ("Le sustituye {{name}}") and verify both locales load in the vitest run

## 7. Final checks

- [x] 7.1 Run `openspec validate improve-cpu-lineup-intelligence --strict` and verify it reports valid
- [x] 7.2 Run backend `go build ./... && go vet ./... && go test ./...` and verify all pass
- [x] 7.3 Run frontend `npx tsc --noEmit && npx oxlint && npx vitest run` and verify all pass

## Workflow follow-up

- Archive the change and sync the `cpu-team-management` and `match-simulation` specs once it is merged.
