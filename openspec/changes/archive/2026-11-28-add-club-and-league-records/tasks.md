# Tasks

## 1. Backend

- [x] 1.1 Add `common/records.go` with the club and league records and their cache; verify "Margins tie" and "W W D L L" with `TestClubMatchRecords`, "Two champions with one title each" with `TestLeagueTitleCounts`, and "Title beats runner-up" and "Record set on air" with `TestRecords_FromTheStoredHistory`.
- [x] 1.2 Add `GET /v1/teams/team/:id/records` and `GET /v1/teams/division/:id/records`.

## 2. Frontend

- [x] 2.1 Add the types, hooks and en/es strings; `ClubRecordsSection` on the history tab; `LeagueRecordsPage` at `/division/:id/records` with the "History" button on the division page. Verify with vitest tests and a 360 px check.

## 3. Specs and checks

- [x] 3.1 Copy the design's Code map into `openspec/specs/records/spec.md` after archiving.
- [x] 3.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.
