# Tasks

## 1. Storage, validation and API

- [ ] 1.1 Add `resources/sql/init97_lineup_roles.sql` (role columns on the three per-fixture lineup tables, `default_lineup_roles`), re-runnable; verify by applying it twice on the test DB.
- [ ] 1.2 Add the role fields to `common.Lineup`, read/write them in every lineup getter/setter and the kickoff snapshot, dropping roles of players no longer in the squad; verify "Roles saved" and "Taker sold" with integration tests for each lineup kind.
- [ ] 1.3 Validate roles (XI or bench, no goalkeeper takers) in every lineup handler; verify "Goalkeeper as taker" with a handler test.

## 2. Simulation (engine)

- [ ] 2.1 Use the named taker when on the pitch in `processSetPiece`; verify "Named taker on the pitch", "Named taker on the bench" and "Named free-kick taker substituted off" with unit tests in `scorer_simulation_test.go`, plus the unchanged automatic scenarios.
- [ ] 2.2 Add the captain's 10% card cancellation in `DisciplineSimulation`; verify "Captain on the pitch", "Captain booked" and "Captain sent off" with seeded unit tests, and update the expected card rate in `discipline_simulation_test.go`.

## 3. CPU captain

- [ ] 3.1 Name the oldest outfield starter as captain in `PlanAiMatchLineup`; verify "Veteran captain" and "Takers" in `ailineup_test.go`.

## 4. Frontend

- [ ] 4.1 Add the three role toggles to `SelectedPlayerPanel` and badges on the owner's pitch, sent with the lineup; add en/es strings; verify with vitest tests that a role moves from one player to another and is saved.
- [ ] 4.2 Show the "C" armband in `Lineups.tsx` / `MatchPitchChip.tsx` and the ratings list from kickoff on; verify "Finished match" and "No captain" with vitest tests.

## 5. Specs and checks

- [ ] 5.1 Update the Code maps of `match-simulation`, `cpu-team-management` and `match-viewer`; verify `openspec validate add-captain-and-set-piece-takers --strict` passes.
- [ ] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
