# Tasks

## 1. Shape differences

- [ ] 1.1 Add `ShapeDifference` and the pure `PyramidShape.Differences(target)` to `backend/common/pyramid.go` (kinds `extra`, `missing`, `size`, `ratio`, `invalid_structure`). Re-express `MatchesConfig` through it. Verify with table tests in `common/pyramid_test.go`: in shape (no differences), 1/2/3 against 1/2/4 (`missing` III.4 plus `ratio`), a leftover IV.2 (`extra`), an 11-team group (`size`), an invalid 1/2/3 target (`invalid_structure` only). Check that the existing `MatchesConfig` tests still pass.

## 2. Restructure leaves no team behind

- [ ] 2.1 Route `common.DissolveTeam` and `common.CreateLeague` through `engine` package variables (`dissolveTeam`, `createLeague`), as with `restructureTarget`. Add `common.GetDissolveSkipReason` (a named `StmtSql` plus a wrapper) for logging an `ok=false` dissolution. Verify with `go build ./...` and the existing `engine/pyramid_integration_test.go` passing unchanged.
- [ ] 2.2 In `engine/restructure.go`, turn the dissolution phase into the replanning loop from design D1 (at most 5 rounds, failed teams pinned as protected, a team that is already dissolved not pinned). Verify with a `PlanRestructure` unit test in `engine/restructure_test.go`: a pinned team beyond the cut makes the lowest unprotected team the one dissolved, and the groups keep their size.
- [ ] 2.3 Add integration tests to `engine/pyramid_integration_test.go`, against `STRIKER_TEST_DB_DSN`, for the spec scenarios "Club claimed during the restructure", "Dissolution error" and "Leftover groups from an earlier restructure". Use the `dissolveTeam` hook to set an owner just before the first planned dissolution, or to return an error. Verify that every active league matches the structure with an even team count, that no old league stays active, and that the right number of teams is dissolved.

## 3. Result check and retry

- [ ] 3.1 Make `restructureIfNeeded` return the shape differences after applying. In `engine/season.go`, add `RestructureAttempts` to `rolloverState` and keep the step at `rolloverStepPromotionApplied` while differences remain and fewer than 3 attempts have run. After the 3rd attempt, log every difference at error level and move on. Verify with integration tests for "Transient failure" (the `createLeague` hook fails once: season 1 after the first tick, season 2 with a 1/2/4 shape after the second, players aged once) and "Persistent failure" (it always fails: the season advances on the 3rd tick and the country is still pending).
- [ ] 3.2 Add a concurrent test: three replicas run `advanceCountrySeasonIfComplete` at once on the transient-failure path, then on the next tick. Verify that the season advances by exactly 1, one cup is created and the shape matches. This is the engine-replicas rule for a changed rollover step.

## 4. Admin sees why

- [ ] 4.1 Change `common.GetPendingRestructureCountries` to return `[]PendingRestructure` with the differences. Add `pendingRestructureDetails` to `common.AdminOverview` and fill it in `handlers.AdminOverviewHandler`, keeping `pendingRestructure []string`. Verify with a handler test that seeds a 1/2/3 country under a 1/2/4 structure and checks the JSON: `missing` III.4 and `ratio`.
- [ ] 4.2 Show the details under the pending `Alert` in `frontend/src/features/admin/WorldTab.tsx`, extend `api/types.ts`, and add `admin.worldStructure.pendingDetail.{extra,missing,size,ratio,invalidStructure}` to `en.json` and `es.json`. Verify with a `WorldTab.test.tsx` case that renders one line per difference, and one case with no notice when nothing is pending.

## 5. Specs and checks

- [ ] 5.1 Update the `league-pyramid` Code map (in this change's design.md until archive): `PyramidShape.Differences`, `GetDissolveSkipReason`, the `dissolveTeam`/`createLeague` hooks, `rolloverState.RestructureAttempts`, and `pendingRestructureDetails`. Update the restructure notes in the backend README. Verify that `openspec validate harden-league-restructure --strict` passes.
- [ ] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` with `STRIKER_TEST_DB_DSN` set, and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`. Verify that all pass, apart from failures that also fail on `main`; list any such failures in the PR.

## Workflow follow-up

- Archive after `allow-removing-divisions` is archived, because both deltas touch `league-pyramid` (`/opsx:archive`).
- After the deploy, check that ES restructures to 1/2/4 at the end of season 4 and that the admin no longer lists it as pending.
