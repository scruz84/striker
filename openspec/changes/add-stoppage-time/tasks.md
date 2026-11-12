# Tasks

## 1. Schema and data model

- [x] 1.1 Add `resources/sql/init107_stoppage_time.sql` (design §3): `added_minute` on the three event and three instant tables, `added_time` on the event tables, the guarded primary-key swap on the instant tables, and the two added-time columns on `matches`, `cup_ties` and `friendly_matches`. Verify by applying it twice to a dev database with no error, and check that `\d match_instants` shows the 3-column primary key.
- [x] 1.2 Add `common.ADDED_TIME = 9`, `Event.AddedMinute`, `Event.AddedTime` (`*int`), `MatchInstant.AddedMinute` and `MatchSimulation.FirstHalfAddedTime` / `SecondHalfAddedTime` / `PlayedMinutes`. Verify with `go build ./...`.
- [x] 1.3 Write and read the new columns in `common.Save*ResultTx`, the event and instant getters (league, cup, friendly) and the fixture getters that feed the viewer, all ordered by `minute, added_minute, event_id`. Old rows read as 0, and the default-result path writes 0. Verify with an extended `common/matchresult` integration test that round-trips a match with an added-time goal and a board event.

## 2. Engine: computing and simulating added time

- [x] 2.1 Add `engine/stoppage_time.go` with the constants and `addedMinutesFor` (design §2). Verify with table tests in `engine/stoppage_time_test.go`, one per scenario of the stoppage-time requirements "Time lost per half" and "Added minutes per half" (quiet half, busy half = 395 s, 250 s → 4, penalty, second yellow, both halves clamped at 5).
- [x] 2.2 Restructure `runMatchSimulation` into two halves with a `simulateMinute(clock, added)` step that sets `Minute`, `AddedMinute` and `PlayedMinutes`, appends the `ADDED_TIME` event after minute 45 and minute 90, and stops on forfeit. Verify with `engine/stoppage_time_test.go` cases: 2 + 4 added minutes give 96 instants in order; added-time events don't change A2; a forfeit at 61 has a first-half board only and A2 = 0.
- [x] 2.3 Stamp `AddedMinute` on every event (scorer, discipline, injury, substitution, mentality change) and on each instant. Make scheduled substitutions fire only when `AddedMinute == 0`, and check conditional orders in every tick against the clock minute. Verify with new cases in `engine/conditional_orders_test.go` ("going behind in added time", "from-minute 45 fires at 45+2") and in the substitution tests (a minute-45 sub fires once).
- [x] 2.4 Confirm that fatigue stays capped at 90 with added minutes. Verify with a scenario test in `engine/stamina_simulation` tests (Stamina 0, 95 minutes → −30%).
- [x] 2.5 Pass the added times into the result save for league, cup and friendly (`simulateMatch`, `simulateCupTie`, `simulateFriendly`), and make a cup tie level after added time go to penalties. Verify with `engine` integration tests for each competition asserting the stored columns, the instant count, and `went_to_penalties` after a 90+3 equaliser.
- [x] 2.6 Count added minutes as played in `common/matchratings.go`: elapsed = number of instants, at least 90; the result component stays capped at 90. Verify with `common/matchratings` tests for the three match-ratings scenarios (97 minutes → +0.3; 90+1 injury sub has 6 minutes, s.c.; ratings as of 45+2).
- [x] 2.7 Run the calibration suites (`HOME_ADV_SIM=1 go test ./engine -run TestHomeAdvantageCalibration -v`, `MENTALITY_SIM=1 ... TestMentalityCalibration`, the set-piece goals test) and `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v`. Record the goals-per-match change in the PR, and adjust a bound only with the owner's approval (design: Risks).

## 3. Live replay and time estimates

- [x] 3.1 Rewrite `minutePublishOffset(minute, added)` and the `BroadcastMatchLive` loop: key events by `(minute, added)`, send `halftime` after the last first-half tick and `complete` after the last tick, and add `Minute`/`AddedMinute` to `LiveMatchMessage`. Verify with `engine/livematch` tests covering the 14:00 schedule scenario (45 at 14:44, 45+3 at 14:47, 46 at 14:59, 90+5 at 15:48), the 4-minute first half (46 still at 59), and a late simulation catching up.
- [x] 3.2 Change `common.CurrentMatchMinute` to `(kickoff, a1, a2) → (minute, added)` and `minuteFromLiveMessage` to read the message's own minute. Thread the fixture's added times through `liveStateAsOfNow`, `GetLiveMatchesOverview` and the national-match feeds. Make the score-as-of queries compare `(minute, added_minute)`. Verify with `common/livematches_test.go` cases for the three estimate scenarios and the halftime-with-2-added scenario.
- [x] 3.3 Add `addedMinute` to the division live scores entry and the dashboard live payloads. Verify with `common/league_live_scores_integration_test.go` (entry at 90+2; winner at 90+3 not revealed at 90+1; finished at 90+4) and `handlers/statsHandlers_test.go`.
- [x] 3.4 Add `MaxBroadcastDuration()` and `TestLongestBroadcastFitsLiveWindow` (design §6), and update every "~105 minutes" comment in the backend. Verify the test passes, and that it fails if `MaxAddedSecondHalf` is temporarily set to 13.
- [x] 3.5 Re-check the time-gated readers against the longest match (head-to-head, top scorers, career stats, cup round resolution, World Cup group completion, national feeds, season rollover, pending post-match effects). Verify with one integration test: a league match with its winner at 90+5 is excluded from `GetHeadToHead` and `GetTopScorers` at kickoff + 108 min, its notification is created only after `complete`, and it is included once kickoff + 2 h has passed.

## 4. Frontend

- [x] 4.1 Add `utils/matchMinute.ts` (`formatMatchMinute`, `compareMatchMinute`) and the new type fields in `api/types.ts` and `useLiveMatch.ts` (live state carries `addedMinute`; `halftime`/`complete` read the message minute). Verify with `utils/matchMinute.test.ts` and an `api/useLiveMatch.test.ts` case for added-time messages.
- [x] 4.2 Use the formatter in `MatchScoreboard` (goal list and live clock), `matchCommentary.tsx`, `MatchEventsReport`/`matchEvents.ts`, `matchGoals.ts` and `liveScoreLabel.ts`, and render `ADDED_TIME` as the board. Verify with `MatchViewer.test.tsx` / `MatchScoreboard` tests (`45+2' (pen.)`, live clock `90+3'`, the board shown at 90 and hidden at 88, an old match unchanged) and `liveScoreLabel.test.ts`.
- [x] 4.3 Make `useMinuteScrub`, `MatchMinuteRatings`, `MatchMinuteZoneComparison`, `liveMatchRatings.ts` and `MatchRatingsTab` step over the ordered instants with added-minute labels. Verify with tests for scrubbing 45 → 45+1 → 45+2 → 46, and for the last step `90+4'` showing the stored ratings.
- [x] 4.4 Add the i18n keys `match.addedTime.board` (pluralised) and `match.addedTime.label` to `frontend/src/i18n/locales/en.json` and `es.json`, update the "~105 minutes" comments, and add a test that pins `maxLiveBroadcastWindowMs` to 2 h. Verify with the locale parity test and `npx vitest run`.

## 5. Specs and docs

- [x] 5.1 Update the Code map sections: new `stoppage-time` (`engine/stoppage_time.go`, `runMatchSimulation`, `common.ADDED_TIME`, `init107_stoppage_time.sql`, `TestLongestBroadcastFitsLiveWindow`), `match-simulation`, `match-scheduling` (`minutePublishOffset`, `CurrentMatchMinute`), `division-live-scores`, `match-viewer` (`utils/matchMinute.ts`) and `match-ratings`. Also update the live replay section of `backend/README.md` and the migration list. Verify with `openspec validate add-stoppage-time --strict`.

## 6. Integration checks

- [x] 6.1 Backend: `go build ./... && go vet ./... && go test ./...` passes.
- [x] 6.2 Frontend: `npx tsc --noEmit && npx oxlint && npx vitest run` passes.
- [x] 6.3 Manual run: start the app locally, create a friendly that kicks off in 5 minutes with the replay pacing shortened for testing, and watch it show the board at 45, the `45+N'` clock, the shorter break and `90+N'`. Confirm that the league table and the dashboard show no result before `complete`.

## Workflow follow-up

- Archive the change once it is merged and the specs are synced.
