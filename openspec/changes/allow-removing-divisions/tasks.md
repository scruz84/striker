# Tasks

## 1. Schema and structure model

- [x] 1.1 Add `backend/resources/sql/init119_country_structures.sql` (`countries.structure`, `teams.dissolved_season`, partial index), re-runnable. Verify by applying it twice through `EnsureSchema` against a fresh Postgres.
- [x] 1.2 `common/worldconfig.go`: `EffectiveStructure`, get/set/clear of a country's structure, and `PlanStructureChange` (user teams must fit, returns teams to dissolve per country). Verify with `common/worldconfig_test.go` table tests and a `common` integration test for a country with an override and one without.

## 2. Restructure with dissolution

- [x] 2.1 `engine.PlanRestructure`: `userOwned` on `RankedTeam`, cut at places, swap user teams into the bottom places, `Dissolve` list. Verify with `engine/restructure_test.go` cases for removing the bottom tier, a user team beyond the cut, and an unchanged top-up.
- [x] 2.2 `common.DissolveTeam` with the clearing steps extracted from `ResetReleasedTeam` into a shared helper (no behaviour change for the reset). Loans in go home, loans out release at loan end, academy prospects retire. Verify with an integration test for each "Dissolved team" scenario, and check the existing `ResetReleasedTeam` tests still pass.
- [x] 2.3 `restructureTarget(dbCtx, country)` returns the effective structure. `applyRestructure` dissolves before moving teams. Verify with `engine/pyramid_integration_test.go`: 5→4 tiers dissolves 160 CPU teams and keeps the user team, a per-country override leaves other countries alone, and a rollover interrupted after the dissolutions resumes to the same result.

## 3. Dissolved teams stay out

- [x] 3.1 Audit every `FROM teams` statement in `common`, `engine` and `handlers` and add `dissolved_season IS NULL` where clubs are listed for play, assignment, AI, market, wages or sponsors. Verify with an integration test that seeds a dissolved team and checks it is absent from `AssignTeam`, AI daily activity candidates, friendly opponents, market buyers and the weekly batches.

## 4. Admin API

- [x] 4.1 `PUT /v1/admin/world-config` gains `confirmDissolve` and the 409 with per-country counts. Add `GET/PUT/DELETE /v1/admin/countries/:code/structure` and list the countries with their own structure in the admin overview. Verify with `handlers/worldConfigHandler_test.go` cases: 409 without confirmation, 200 with it, user teams that do not fit rejected, non-admin refused.

## 5. Frontend

- [x] 5.1 World tab: own-structure editor per country, "Use global structure", the dissolve confirmation dialog, and "missing division" checked against the effective structure. Team page: "Dissolved in season N" chip and no actions. Add the strings to `en.json`/`es.json`. Verify with `WorldTab.test.tsx` cases (409 → dialog → confirmed resend) and a team page test.

## 6. Specs, economy and checks

- [x] 6.1 Update the `league-pyramid` Code map (kept in design.md until archive, since a delta cannot carry one) (migration, `EffectiveStructure`, `PlanStructureChange`, `DissolveTeam`, the new handlers) and the backend README's world-management section. Verify `openspec validate allow-removing-divisions --strict` passes.
- [ ] 6.2 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` with a 5→4 tier shrink and record the free-agent and wage figures in this change's design. Verify the economy stays within the simulation's existing bounds.
- [x] 6.3 Run backend `go build ./... && go vet ./... && go test ./...` with `STRIKER_TEST_DB_DSN` set, and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`, all passing apart from the existing order-dependent engine failures, which also fail on `main`.

## Workflow follow-up

- Archive the change once implemented and reviewed (`/opsx:archive`), syncing the `league-pyramid` delta into `openspec/specs/`.
