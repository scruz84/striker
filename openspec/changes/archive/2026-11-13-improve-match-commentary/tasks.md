# Tasks

## 1. Narration engine

- [x] 1.0 Implement `add-open-play-keeper` first: open-play events carry the keeper in every competition.
- [x] 1.1 Add `narrateMatch(events, ctx)` to `frontend/src/utils/matchCommentary.tsx` (design §1): sort with `compareMatchMinute`, return one entry per event with its main pool key, variant index, optional tail and placeholder values. Keep `commentaryFor` working as a wrapper. Verify with the existing `utils/matchCommentary.test.ts` passing unchanged.
- [x] 1.2 Add situation detection (design §3): per-player goal count, a per-side "has trailed" flag, late = minute ≥ 85, the twelve situations in spec order, `secondYellow` for a red after a yellow, and `lateMiss`. Verify with table tests in `matchCommentary.test.ts`, one per scenario of "Goal situation sentence", "Second yellow line" and "Late miss sentence" (late equaliser 2-2 at 88, hat-trick 3-3 at 90+2 with `{{goals}}` = 3, comeback 0-2 → 3-2, opener, rout, extended lead, pulled one back, consolation, booked twice, straight red, missed equaliser at 89, a 3-0 lead missing at 88).
- [x] 1.3 Add the seeded non-repeating picker and the colour rolls (design §4). Verify with tests: 12 goals over a 12-line pool all differ; the same match narrated twice gives identical output; across 10,000 synthetic misses the colour share is between 27% and 33% (yellow cards between 22% and 28%).
- [x] 1.4 Verify spoiler safety (design §5): narrate one match with and without its red-card and injury events, and assert every other entry is identical (spec "Commentary stays the same from live to finished").

## 2. Chronicle rendering

- [x] 2.1 Make `MatchEventsReport` call `narrateMatch` once (memoised), render the entries through `interpolateNodes` with the tail appended to the main line, and apply the "key events only" filter after narration. Verify with a `MatchEventsReport.test.tsx` case: a line reads the same with the toggle on and off.
- [x] 2.2 Replace the fixed kickoff string with the seeded kickoff pool, and the full-time string with the result-based pool. Keep `match.report.fullTime` for abandoned and default-result matches (design §6). Verify with tests for a 0-4 match (big-win line naming the away side) and a default result (neutral line), and that the kickoff line is the same with and without `final`.
- [x] 2.3 Add the half-time line and the `firstHalfOver` prop. Pass `live.halftime || live.minute > 45` from `MatchViewer` and `true` from finished views, except for a match abandoned in the first half. Verify with `MatchEventsReport.test.tsx` cases for the four match-viewer scenarios (1-0 finished match, live at 45+1, live during the break, abandoned at 30) and a `MatchViewer.test.tsx` check that the prop follows `halftime`.

## 3. Writing the lines

- [x] 3.1 Write the Spanish lines in `frontend/src/i18n/locales/es.json` under the keys from design §2, meeting every minimum in "Phrasing pool sizes" and the writing rules in design §7. Rework the existing lines so they don't clash with the new tails.
- [x] 3.2 Write the English lines in `frontend/src/i18n/locales/en.json` the same way: written natively, not translated line by line.
- [x] 3.3 Add a locale test (`utils/matchCommentaryLocales.test.ts`) that checks both languages: every pool exists, meets its minimum, uses only its allowed placeholders, and no substitution line mentions an injury ("lesi", "injur"). Verify it fails when a line is removed from a pool or given an unknown placeholder.
- [x] 3.4 Read through about 5 simulated matches in both languages (one goalless, one rout, one late comeback, one with a second yellow, one with a late equaliser). Done with event lists simulated by the engine's own pipeline, rendered in the real `MatchEventsReport` and in a phone-width page with the app theme; a full world was not brought up. The live half-time behaviour is covered by the `MatchEventsReport` / `MatchViewer` tests. Fix lines that read badly next to each other, and take screenshots of a phone-width chronicle for the PR.

## 4. Specs and checks

- [x] 4.1 Update the Code map of `openspec/specs/match-viewer/spec.md` (`MatchEventsReport` half-time line, `firstHalfOver` from `MatchViewer`), and write the `match-commentary` Code map (`utils/matchCommentary.tsx` `narrateMatch`, locale keys, the tests) for when the new spec is created.
- [x] 4.2 Run backend `go build ./... && go vet ./... && go test ./...` (no backend change expected, as a guard) and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`.

## Workflow follow-up

- Archive the change with `/opsx:archive` once it is merged, and check that the new `openspec/specs/match-commentary/spec.md` has its Purpose and Code map.
