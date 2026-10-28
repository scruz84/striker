# Tasks

## 1. Release re-dated fixtures

- [x] 1.1 Add `common.ReleaseRedatedFixtures` and `RedatedFixtureMargin` (`common/defaultresults.go`) with one `UPDATE` per table (design Decision 1). Add `engine.ReleaseRedatedFixtures` and run it before `SettleDefaultResults` in `FinishStuckSimulationsJob` and `AdminFinishStuckMatchesHandler`. Verify with an integration test: exhausted league match, friendly and cup tie dated in 2 days are released; one dated in 5 minutes and one claimed seconds ago are left alone; the released match is simulated normally once its kickoff arrives.
- [x] 1.2 Add a concurrent integration test: replicas releasing at once leave the fixture released, with 0 attempts.

## 2. Admin action

- [x] 2.1 Add `released` to `finishStuckMatchesResponse` and update `TestBuildFinishStuckMatchesResponse_Shape`.
- [x] 2.2 Frontend: add `released` to `FinishStuckMatchesResult` (`api/types.ts`), pass it in `OverviewTab.tsx`, and add `{{released}}` to `admin.actions.finishStuckMatchesSuccess` in `en.json` and `es.json`. Verify by updating `OverviewTab.test.tsx`.

## 3. Specs

- [x] 3.1 Update the match-scheduling Code map (Recovery from interrupted simulations) to name `ReleaseRedatedFixtures`.

## 4. Final checks

- [x] 4.1 Backend: `cd backend && go build ./... && go vet ./... && go test ./...` passes, including the integration tests with `STRIKER_TEST_DB_DSN` set.
- [x] 4.2 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run` passes.
- [x] 4.3 `openspec validate release-redated-stuck-fixtures --strict` passes.

## Workflow follow-up

- After deploy, press "finish stuck matches" once: the re-dated fixtures should appear as released, and they should no longer count as stuck or failing.
- Archive the change once it has been reviewed and deployed.
