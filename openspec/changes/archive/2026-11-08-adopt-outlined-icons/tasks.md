# Tasks

## 1. Icons

- [x] 1.1 Replace every `*TwoTone` icon import and its JSX uses with the `*Outlined` glyph in the 14 files (design decision 1); verify with `npx tsc -b` and `grep -rn "TwoTone" frontend/src` returning nothing outside the new test.
- [x] 1.2 Add `frontend/src/iconStyle.test.ts` (design decision 2); verify it passes, and fails when a `@mui/icons-material/FlagTwoTone` import is temporarily added to a file ("No two-tone icon left").

## 2. Checks

- [x] 2.1 Against a local seeded backend, screenshot the dashboard, cup, leagues, national teams, friendlies and help pages at 360 px before and after ("Dashboard card headers"), and run the mobile smoke test; no new finding over `main`.
- [x] 2.2 Extend the `ui-theme` spec's Code map with `iconStyle.test.ts` when archiving, and run `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run && npm run build` and the backend `go build ./... && go vet ./... && go test ./...`; all pass.
