# Tasks

## 1. Schema

- [ ] 1.1 Add `backend/resources/sql/init86_simulation_failures.sql` (design Decision 7). It adds `simulation_attempts`, `last_simulation_error` and `last_simulation_attempt_at` to `matches`, `friendly_matches` and `cup_ties`, creates the `pending_post_match_effects` table with its `due_at` index, and seeds already-stuck rows with 2 attempts inside the column-existence guard. Verify by running it twice against a test DB (`STRIKER_TEST_DB_DSN`): the second run changes nothing, and a row stuck for 2 hours ends with `simulation_attempts = 2`. If `init85_observability_indexes.sql` has not landed yet, check the numbering.

## 2. Atomic, only-once result save

- [ ] 2.1 Add `SaveLeagueMatchResultTx`, `SaveFriendlyResultTx` and `SaveCupTieResultTx` in `common/`, with `ErrAlreadyPlayed` and the `LockNamespaceStandings` lock (Decision 1). Verify with integration tests: a second save of the same fixture returns `ErrAlreadyPlayed` and leaves the standings unchanged; a failing event insert stores nothing; leftover events and instants are replaced.
- [ ] 2.2 Switch `simulateMatch`, `simulateFriendly` and `simulateCupTie` to these functions, and book money, flavor and the broadcast only after the commit. Remove the swallowed error in `UpdatePlayedMatch` and its separate `UpdateClassification` call. Verify by running `go test ./engine` and an integration test where two concurrent simulations of one match book wages and standings once.
- [ ] 2.3 Update the match-simulation spec Code map (Post-match processing). Verify that the spec names the new save functions.

## 3. Attempt counting, backoff and release on failure

- [ ] 3.1 Increment attempts in `GetSimulableMatches`, `GetSimulableFriendlies`, `GetStuckMatches`, `GetStuckFriendlies` and the cup-tie lock path. Add the `< 3` and 10-minute backoff filters to the three selectors, and check the claim `Exec` and `Commit` errors in `GetSimulableMatches`. Verify with an integration test: a fixture failed 5 minutes ago is not selected; one failed 11 minutes ago is selected, with its attempts incremented.
- [ ] 3.2 Add `common.IsConnectionError` and `common.RecordSimulationFailure` (release the claim, store an error truncated to 500 chars, decrement attempts on a connection error). Call it on the error and panic paths of `SimulateMatchesNow`, `simulateFriendliesNow` and `SimulateCupTies`. Verify with unit tests for `IsConnectionError` (SQLSTATE 08006, 57P01, 53300, `driver.ErrBadConn` are true; 23505 is false) and an integration test where a panicking simulation leaves `in_simulation = false` with the error stored.
- [ ] 3.3 Add a concurrent integration test in `engine/engine_replicas_integration_test.go`: two replicas claiming the same failing fixture count one attempt per claim and never simulate it in parallel.

## 4. Default 0-0 result

- [ ] 4.1 Add `common.MATCH_DEFAULT_RESULT = 7` and `engine.SettleDefaultResults` (Decision 3), and call it from `FinishStuckSimulationsJob` before the re-simulation. Verify with integration tests for each kind: a league match at 3 attempts becomes 0-0 with +1 point each and wages charged, with no gate or sponsor income; a cup tie gets a winner by shoot-out and the prize; a World Cup group friendly gets a group draw.
- [ ] 4.2 Add a concurrent test: two replicas running `SettleDefaultResults` at once settle each fixture once, with standings, wages and prize booked once.
- [ ] 4.3 Add an integration test: after the migration seeding, a stuck league match gets exactly one attempt, then is defaulted, and its country's season rollover (`AdvanceSeasonsIfComplete`) proceeds.
- [ ] 4.4 Frontend: add `EVENT_TYPE_MATCH_DEFAULT_RESULT = 7` in `frontend/src/api/types.ts`, render it in `MatchEventsReport.tsx` and `utils/matchCommentary.tsx`, and add `match.commentary.defaultResult` plus the events-report label to `en.json` and `es.json`. Verify with a vitest case that renders a default-result event in both languages.

## 5. Durable post-match effects

- [ ] 5.1 Add `common.PostMatchPayload`, insert the pending row from each save function (Decision 4), and make `onComplete` claim it with `DELETE … RETURNING` before applying the effects from the payload. Verify with an integration test: a completed replay applies the effects once and leaves no pending row.
- [ ] 5.2 Drain due rows from `LiveBroadcastRecoveryJob` with `FOR UPDATE SKIP LOCKED`. Verify with an integration test: a pending row due 1 minute ago is applied by one of two concurrent drains, and a row due in the future is left alone.
- [ ] 5.3 Update the match-scheduling spec Code map (Recovery from interrupted simulations, Durable post-match effects) and the backend README section on live replay recovery. Verify that both name `pending_post_match_effects`.

## 6. Standings repair

- [ ] 6.1 Add `engine.RepairStandingsOnce` and `common.CountStandingsMismatches` (Decision 5), and call the repair at engine start. Verify with an integration test: a team with `matches = 12` and 11 played matches is rebuilt to the 11-match totals, a consistent league is untouched, and of two concurrent starts only one runs the repair.
- [ ] 6.2 Update the engine-replicas spec Code map (Simulation claims, Attempt counting, One-off standings repair). Verify that it names the slot `standings_repair_v1` and `LockNamespaceStandings`.

## 7. Admin overview and action

- [ ] 7.1 Replace `CountStuckSimulations` with `GetSimulationHealth`, covering cup ties, the failing list capped at 50, and standings mismatches. Change `AdminFinishStuckMatchesHandler` to run the default settlement and then the re-simulation, returning `{resimulated, defaulted, stillFailing}` per kind. Verify with a handler test of the response shape.
- [ ] 7.2 Frontend `features/admin/OverviewTab.tsx`: show the failing-matches table (kind, teams, kickoff, attempts, last error), the standings-mismatch alert and the new success message. Update `api/types.ts` and `api/hooks.ts`, and the `admin.overview.*` / `admin.actions.*` keys in `en.json` and `es.json`. Verify by updating `OverviewTab.test.tsx` for the new alert, the table and the success counts.

## 8. Final checks

- [ ] 8.1 Backend: `cd backend && go build ./... && go vet ./... && go test ./...` passes, including the integration tests with `STRIKER_TEST_DB_DSN` set.
- [ ] 8.2 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run` passes.
- [ ] 8.3 `openspec validate harden-stuck-match-recovery --strict` passes.

## Workflow follow-up

- After deploy, check the admin overview: no stuck or failing rows older than about an hour, and 0 standings mismatches.
- Archive the change once it has been reviewed and deployed.
