# Tasks

## 1. Schema

- [x] 1.1 Add `backend/resources/sql/init108_event_other_team_players.sql` (design §2). Verify by applying it twice to a local Postgres with no error.

## 2. Engine and storage

- [x] 2.1 In `engine/scorer_simulation.go` `processChance`, record the defending keeper's id and name in `OtherTeamPlayers` / `OtherTeamPlayersNames` (design §1). Verify with a unit test that an open-play event lists the defending side's `GOALKEEPER[0]`, for a home and an away chance.
- [x] 2.2 Add the two columns to `InsertCupTieEventStmtSql`, `InsertFriendlyEventStmtSql`, `GetCupTieEventsStmtSql` and `GetFriendlyEventsStmtSql`, remove `withOtherTeam` from `replaceMatchEventsTx`, and read the columns in `GetCupTieEvents` / `GetFriendlyEvents` (NULL → empty). Verify with an integration test (STRIKER_TEST_DB_DSN) that a cup tie and a friendly round-trip the keeper on an open-play event, and that a row inserted without the columns reads back with no keeper.

## 3. Specs and checks

- [x] 3.1 Update the match-simulation Code map (keeper on every chance, `init108`, the cup and friendly getters) and the `Event` doc comment in `common/fmstructs.go`.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...`, with the integration tests against a local Postgres. Run frontend `npx tsc --noEmit && npx oxlint && npx vitest run` as a guard.
