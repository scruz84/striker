# Tasks

## 1. Backend

- [x] 1.1 Add `resources/sql/init116_resumable_season_rollover.sql` (`rollover_season`, `rollover_step`, `rollover_state` on `countries`), re-runnable.
- [x] 1.2 `common/rollover.go`: read, record and clear the marker, and the season bump that records step 7.
- [x] 1.3 `engine/season.go`: number the steps, record each one, skip the ones done, keep the standings and moves in the marker, generate the fixtures of a season moved by an interrupted rollover.
- [x] 1.4 Integration tests in `engine/season_rollover_resume_integration_test.go`: stop after the prizes and after the season bump, run again, check every effect happened once.

## 2. Specs and checks

- [x] 2.1 `engine-replicas` delta: rollover requirement names the marker; new "Resumable season rollover" requirement.
- [x] 2.2 Run backend `go build ./... && go vet ./... && go test ./...` and the engine integration tests against Postgres.
