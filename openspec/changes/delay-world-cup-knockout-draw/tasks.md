# Tasks

## 1. Broadcast-aware group-stage completion

- [ ] 1.1 Make `getWorldCupGroupMatchCountsStmtSql` also return `MAX(fm.match_date)` and have `IsWorldCupGroupStageComplete` require it to be at or before `LiveBroadcastCutoff()`; verify with a common integration test where all group matches are played with the last kickoff 40 minutes ago (not complete) and then moved back past the cutoff (complete)
- [ ] 1.2 Add an engine integration test calling `advanceWorldCup` for a World Cup whose last group match is played but kicked off 40 minutes ago, asserting no knockout cup and no knockout call-up notification exist, then that one knockout cup exists after moving the kickoff back 2 h; verify `go test ./engine -run WorldCup` against Postgres
- [ ] 1.3 Check that `TestAdvanceWorldCup_ConcurrentReplicasStartOneKnockout` (`engine/engine_replicas_integration_test.go`) still passes unchanged (its group match kicked off 3 hours ago)

## 2. Specs and verification

- [ ] 2.1 Update the `world-cup` spec Code map (`IsWorldCupGroupStageComplete` now uses `LiveBroadcastCutoff`) and verify `openspec validate delay-world-cup-knockout-draw --strict`
- [ ] 2.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass

## Workflow follow-up

- Archive the change and sync the `world-cup` spec once merged.
