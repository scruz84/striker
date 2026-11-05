# Tasks

## 1. Measurement

- [x] 1.1 Add `common/ailevelstats.go` (`GetAiLevelStatsStmtSql`, `GetAiLevelStats`, `AiLevelStats`) and register the four `striker.ai.*` gauges in `RegisterMarketGauges`; verify "Two Directors" and "Owned club" with an integration test.
- [x] 1.2 Add `AiLevels` to `AdminOverview` and a per-level table to the admin overview tab with en/es strings; verify with a vitest test.
- [x] 1.3 Add the CPU level panels to the Simulated world row of `striker-dashboard.json`; verify the metric-coverage test passes.
- [x] 1.4 Add `engine/ai_level_calibration_test.go` (`AI_LEVEL_SIM=1`); run it and record the numbers in this change.
  Result (4,000 matches each, seed 11): Manager 1.399 points per match vs its Caretaker's 1.360
  (won 38.6%, drew 24.1%, lost 37.3%); Director 1.501 vs 1.277 (won 42.6%, drew 22.2%, lost 35.2%).
  The Manager's edge is close to noise: its fixed second-best formation costs it most of what its
  mentality and substitutions gain (to be addressed by the CPU tactics change).

## 2. CPU kickoff fixes

- [x] 2.1 Make `AiCaptain` skip scheduled outgoing players; verify "Veteran captain" and "Veteran due off" in `ailineup_test.go`.
- [x] 2.2 Add `common.AiCupDrawOrder` and append it in `planAiKickoff` for cup ties; verify "Favourite level late", "Underdog level late" and "League match" with unit tests.

## 3. CPU management

- [x] 3.1 Add `bestSponsorDeal` / `chooseSponsorDealIfManager`; verify "Payout formulas", "Strugglers take the guarantee", "Runaway leader" and "Too early" with unit tests plus one integration test of the lock.
- [x] 3.2 Add `pickManagerTrainer` / `hireTrainerIfManager`; verify "Elite coaches left alone", "Only elite coaches" and "Better coach appears" with unit tests.

## 4. National teams

- [x] 4.1 Make `selectNationalSquadLineup` drop injured players and pick the best formation, captain and Director orders; verify "Injured star" and "Pool full of strikers" with an integration test and "Orders" with a unit test.

## 5. Specs and checks

- [x] 5.1 Update the Code maps of `cpu-team-management`, `national-teams` and `observability`; verify `openspec validate improve-cpu-ai-foundations --strict` passes.
- [x] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
