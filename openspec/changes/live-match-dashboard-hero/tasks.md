# Tasks

## 1. Shared helpers and theme

- [x] 1.1 Add `liveMatchPath(match)` to `frontend/src/utils/matchSummary.ts` (league `/v1/teams/match/:id/live`, cup `/v1/cups/ties/:id/live`, friendly `/v1/friendlies/:id/live`) and use it for `liveUrl` in `fromWeekMatch` / `fromCupTie` / `fromFriendly`; verify with a new unit test per kind and the existing `matchViewModel.test.ts` still passing
- [x] 1.2 Expose `errorContainer` / `onErrorContainer` in `frontend/src/theme.ts` (palette augmentation, options and `buildPalette`, from the M3 scheme) and verify `npx tsc --noEmit` passes and both tokens resolve in light and dark themes
- [x] 1.3 Extract the pulsing LIVE chip `sx` from `MatchScoreboard.tsx` into a shared constant reused by the scoreboard; verify the viewer's live chip renders unchanged (`MatchViewer.test.tsx` passes)

## 2. Match viewer: division in the league subtitle

- [x] 2.1 In `fromWeekMatch`, prefix the label with `division.name` when the division is loaded ("<division> · Jornada N de T"), keeping the `division.weekChip` fallback while it loads; verify with `matchViewModel.test.ts` cases for both loaded and loading division
- [x] 2.2 Update `openspec/specs/match-viewer/spec.md` Code map (subtitle bullet mentions the division prefix in `fromWeekMatch`, and `liveMatchPath`); verify by reading the section

## 3. Dashboard live hero

- [x] 3.1 Add and export `selectLiveMatch(teamMatches)` in `DashboardPage.tsx` (kickoff reached and within `mightStillBeLive`, latest kickoff wins, `played` not required); verify with `DashboardPage.test.ts` cases: in-progress match, simulated-early match before kickoff (not live), match older than 2 h (not live), two live matches
- [x] 3.2 Add a `useOwnLiveMatch` hook that opens `useLiveMatch(liveMatchPath(match))` only when a candidate exists, remembers completed `kind:matchId` keys, and invalidates `teamMatches` on completion; verify with a unit test that a `complete` message turns the candidate off
- [x] 3.3 Build `LiveMatchHero` (error-container surface, pulsing "LIVE" label + match kind chip, both team badges in home/away order, broadcast score, minute or "Half-time", no score/minute while `giveUp` or before the first minute, primary "Watch live" button to `matchSummaryLink`); verify with a component test covering minute, half-time and give-up states and that the stored final score is never rendered
- [x] 3.4 Wire `DashboardPage`: render `LiveMatchHero` instead of `NextMatchHero` while a live match is active; compute `lastMatch` excluding only the active live match; remove `LiveMatchBanner` and `LastResultCard`'s LIVE-chip branch; verify with a test that "Last result" shows the previous finished match while one is live
- [x] 3.5 Add `dashboard.liveNow` to `frontend/src/i18n/locales/en.json` ("Live now") and `es.json` ("En directo"), remove `dashboard.liveMatchBanner` from both; verify no remaining references with `grep -rn liveMatchBanner frontend/src`
- [ ] 3.6 Check the live hero manually in light and dark mode at phone and desktop widths (run the app with a seeded live match, or the MSW/dev mock if available) and confirm text contrast on the red surface

## 4. Integration checks

- [x] 4.1 Backend unchanged but run `cd backend && go build ./... && go vet ./... && go test ./...` and verify all pass
- [x] 4.2 Run `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run` and verify all pass

## Workflow follow-up

- Archive the change once reviewed (creates `openspec/specs/team-dashboard/spec.md` and updates `match-viewer`).
- Right after archiving, append a `## Code map` section to `openspec/specs/team-dashboard/spec.md` listing `features/dashboard/DashboardPage.tsx` (`selectLiveMatch`, `useOwnLiveMatch`, `LiveMatchHero`, `NextMatchHero`, `LastResultCard`), `api/useLiveMatch.ts`, `utils/matchSummary.ts` (`liveMatchPath`) and `theme.ts` (container tokens), and replace any `TBD` Purpose placeholder.
