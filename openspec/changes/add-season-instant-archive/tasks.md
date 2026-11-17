# Tasks

## 1. Schema and settings

- [x] 1.1 Add `backend/resources/sql/init118_instant_archive.sql`: `season.archive_instants_enabled` (default false), `season.archive_keep_seasons` (default 1, CHECK 0-10), `instants_archived` on `matches` / `cup_ties` / `friendly_matches`, and the three partial indexes, all `IF NOT EXISTS`. Verify by applying it twice against a fresh Postgres through `EnsureSchema` (any integration test) with no error.
- [x] 1.2 `common`: `GetInstantArchiveSettings` / `SetInstantArchiveSettings` (validation 0-10) as named `...StmtSql` constants. Verify with an integration test that reads the defaults (off, 1), saves (true, 2), and rejects 11 without a change.

## 2. Archive statements

- [x] 2.1 `common/instantarchive.go`: the batched archive statements for league matches, domestic cups and playoffs, regional/international cups and friendlies (design §2), each with a wrapper returning rows archived, plus `ArchiveOldInstants(dbCtx, keep)` looping per kind until a batch archives 0 rows. Verify with `common/instantarchive_integration_test.go` covering the match-data-retention scenarios: keep 1 and keep 0 in season 5, two countries in different seasons, the regional edition, the friendly before and after the cutoff, an unplayed match and an incomplete cup left alone. Each test also checks that events, ratings and lineups remain and the flag is set.
- [x] 2.2 Concurrency: an integration test running `ArchiveOldInstants` from two goroutines over the same data, checking every eligible fixture is archived exactly once and nothing else is touched.

## 3. Engine job

- [x] 3.1 `engine.InstantArchiveJob` (claims `instant_archive` for 24 h, returns while the archive is off, logs counts) and its daily 04:41 UTC schedule wired in `engine/fmengine.go` (`StartEngine`) next to the other daily jobs. Verify with an engine integration test where two job executions in the same slot run the pass once, and an off setting deletes nothing.

## 4. Admin API and UI

- [x] 4.1 `PUT /v1/admin/instant-archive` handler in `handlers/`, registered in `service/fmservice.go` on the admin group, and the two fields on `common.AdminOverview` / `AdminOverviewHandler`. Verify with a handler test: 200 and persisted for valid input, 400 for 11 or a missing field.
- [x] 4.2 Frontend: `useAdminSetInstantArchive` in `api/hooks.ts`, the overview fields in `api/types.ts`, `InstantArchiveForm` in `features/admin/RulesTab.tsx` (switch, number 0-10, a note that the deletion is permanent), and the `admin.instantArchive.*` strings in `i18n/locales/en.json` and `es.json`. Verify with `RulesTab.test.tsx` cases for render and submit.

## 5. Specs, docs and checks

- [x] 5.1 Write the `match-data-retention` Code map (kept in design.md until archive, since a delta cannot carry one) (the migration, `common/instantarchive.go`, the job, the handler, `RulesTab`). Mention the archive in `backend/README.md` next to the other admin rules, including that disk only shrinks with `VACUUM FULL`. Verify `openspec validate add-season-instant-archive --strict` passes.
- [x] 5.2 Run backend `go build ./... && go vet ./... && go test ./...` with `STRIKER_TEST_DB_DSN` set, and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`, all passing apart from the existing order-dependent engine failures, which also fail on `main`.

## Workflow follow-up

- Archive the change once implemented and reviewed (`/opsx:archive`), syncing `match-data-retention` and the `engine-replicas` delta into `openspec/specs/`.
