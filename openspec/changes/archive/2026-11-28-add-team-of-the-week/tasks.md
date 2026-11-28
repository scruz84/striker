# Tasks

## 1. Schema

- [x] 1.1 Add `backend/resources/sql/init124_best_xi.sql`.

## 2. Backend

- [x] 2.1 Add `common/bestxi.go`; verify "Ties within a line" with `TestSelectBestXi`, and "Matchday not over" and the exactly-once claim with `TestBestXi_MatchdayCompletesOnceAcrossReplicas`.
- [x] 2.2 Close the matchday from `applyPostMatchEffects` (`maybeCompleteBestXi`) and pick the team of the season in `awardSeasonHonors`, notifying the owners; verify "Four replicas" with `TestBestXi_ClosedOnceWhenTheMatchdayEnds`. Add the Spanish templates.
- [x] 2.3 Add `GET /v1/teams/division/:id/best-xi` and the profile's `bestXi` counts.

## 3. Frontend

- [x] 3.1 Add `BestXiCard` (pitch, matchday arrows, `?xi=`), the profile counts and en/es strings, including the two notification keys; verify with vitest tests and a 360 px check that no two chips overlap.

## 4. Specs and checks

- [x] 4.1 Copy the design's Code map into `openspec/specs/best-xi/spec.md` after archiving.
- [x] 4.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.
