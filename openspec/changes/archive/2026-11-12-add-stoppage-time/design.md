# Design

## Context

- `runMatchSimulation` (`engine/simulation.go`) loops `for min := 1; min <= 90` and sets
  `MatchSimulation.Minute`. Every stage reads that field, and so do events, instants,
  `trackMinutesByPosition` and fatigue (`common.FatigueMultiplierWithPenalty`, already capped
  at `FatigueMatchLength = 90`).
- `BroadcastMatchLive` (`engine/livematch.go`) publishes each instant at
  `matchDate + minutePublishOffset(minute)`: minute *m* ≤ 45 at *m*−1, otherwise at *m*+13,
  so 46 goes out at 59 and 90 at 103. It sends `halftime` after minute 45 and `complete` after
  the last instant. `common.CurrentMatchMinute` (`common/livebroadcast.go`) is the mirror
  estimate, and `minuteFromLiveMessage` hardcodes 45 for `halftime` and 90 for `complete`.
- Spoiler gating today:
  - Broadcast-driven gates: the replay's `onComplete` (notifications, pending post-match
    effects) and the live `complete` message (viewer, division live scores, dashboard hero).
  - Time-driven gates: `common.LiveBroadcastWindow` = 2 h (head-to-head, top scorers, career
    stats, cup round resolution, World Cup group completion, national feeds, season rollover,
    the post-match recovery deadline) and the frontend's `maxLiveBroadcastWindowMs` = 2 h
    (`mightStillBeLive`, `useIsBroadcastOver`, the standings revert in `liveTable.ts`).
  - Nothing assumes "kickoff + 105 minutes" in code. That figure appears only in comments.
- `match_instants`, `cup_tie_instants` and `friendly_instants` have the primary key
  `(id, minute)`. A second instant at minute 45 would break it.
- `getMatchScoreAsOfMinuteStmtSql` and its siblings filter on `minute <= $2`.

## Goals / Non-Goals

**Goals:**
- Added time is deterministic given a match's events, simulated by the existing stages, and
  stored so every reader (replay, estimate, viewer, ratings) agrees on it.
- Old rows read unchanged: added minute 0, no board.
- No new time-based gate. The existing 2-hour window still covers the longest match, and a
  test enforces that.

**Non-Goals:**
- Changing `LiveBroadcastWindow` or switching the time gates to a stored per-match end time.
  That would be more precise, because results would appear right at the final whistle instead
  of 2 h after kickoff. It would also touch about 15 queries and every frontend list. That is a
  separate change if the owner wants it.
- Extra time, time-wasting tactics, scheduling substitutions in added time.

## Decisions

### 1. Minute identity = (clock minute, added minute)
Events and instants get an `AddedMinute int` (0 in regulation). First-half added time is
`(45, 1..A1)` and second-half added time is `(90, 1..A2)`.
- *Why*: every existing reading of `minute` stays correct. `minute <= 45` is still "first
  half", `minute >= 46` is still the second half, and the sub validation (1–90) and the order
  from-minute (1–89) keep their meaning. Old rows need no backfill.
- *Alternative*: a sequential tick (1..90+A1+A2) stored in `minute`. Rejected: tick 46 would
  mean "45+1" in new rows and "46" in old ones, so every consumer would have to branch on
  match age.
- The engine keeps one more counter, `MatchSimulation.PlayedMinutes` (the tick index). Minute
  tracking and fatigue already count per simulated minute, so only the 90-cap matters.

### 2. Engine loop and the stoppage calculator
New `engine/stoppage_time.go`:
- Constants: `StoppageBaseFirstHalf = 30s`, `StoppageBaseSecondHalf = 60s`,
  `StoppageGoal = 40s`, `StoppageSubstitution = 30s`, `StoppageInjury = 60s`,
  `StoppageYellow = 15s`, `StoppageRed = 40s`, `StoppagePenalty = 45s`;
  `MaxAddedFirstHalf = 5`, `MaxAddedSecondHalf = 5`, minimum 1 each (owner decision).
- `addedMinutesFor(events []common.Event, fromMinute, toMinute int, base time.Duration, max int) int`
  is a pure function over events with `AddedMinute == 0` in that range. It rounds half up and
  clamps. Penalties are recognised by `SetPiece == SET_PIECE_PENALTY` on GOAL/NO_GOAL events.
  A second yellow is stored only as a `RED_CARD` event (`DisciplineSimulation.Apply`), so it
  counts once, as a red.
- `runMatchSimulation` becomes two halves. Each half runs `simulateMinute(clock, added)` over
  its regulation minutes, appends the `ADDED_TIME` event (`common.ADDED_TIME = 9`,
  `Event.AddedTime`), then runs the added minutes. The forfeit check still runs after every
  tick, and a forfeit in the first half skips the second half (A2 = 0).
  `MatchSimulation.FirstHalfAddedTime` and `SecondHalfAddedTime` carry the values to the save.
- `SubstitutionSimulation`: scheduled subs fire only when `AddedMinute == 0`, so a minute-45
  sub fires once. Conditional orders are checked every tick by comparing against the clock
  minute.
- The scorer, discipline and injury stages copy `AddedMinute` onto the events they create, and
  `InitialSimulation` copies it onto the instant.

### 3. Schema: `resources/sql/init107_stoppage_time.sql`
Re-runnable:
- `ALTER TABLE match_events|cup_tie_events|friendly_events ADD COLUMN IF NOT EXISTS added_minute smallint NOT NULL DEFAULT 0, ADD COLUMN IF NOT EXISTS added_time smallint`
  (`added_time` is set only on `ADDED_TIME` events, the same way the existing `mentality`
  column is used).
- `ALTER TABLE match_instants|cup_tie_instants|friendly_instants ADD COLUMN IF NOT EXISTS added_minute smallint NOT NULL DEFAULT 0`,
  then a guarded `DO $$ ... $$` block. If the primary key does not yet include
  `added_minute`, it drops the constraint and adds `PRIMARY KEY (id, minute, added_minute)`.
- `ALTER TABLE matches|cup_ties|friendly_matches ADD COLUMN IF NOT EXISTS first_half_added_time smallint NOT NULL DEFAULT 0, ADD COLUMN IF NOT EXISTS second_half_added_time smallint NOT NULL DEFAULT 0`.
- The event and instant inserts in `common.Save*ResultTx` write the new columns. Readers select
  them and order by `minute, added_minute, event_id`. The default-result path writes 0.
- No new `TRANSACTION_*` reason and no new notification key: nothing about money or
  notifications changes.

### 4. Replay pacing (owner decision: the break absorbs A1)
`minutePublishOffset(minute, added)`:
- regulation *m* ≤ 45 → *m*−1
- `(45, k)` → 44+*k*
- *m* ≥ 46 → *m*+13
- `(90, k)` → 103+*k*

`halftime` is sent after the last first-half tick and `complete` after the last tick. Both
messages carry `Minute` and `AddedMinute` (new fields on `LiveMatchMessage`), so
`minuteFromLiveMessage` stops hardcoding 45 and 90.

Because 46 stays at 59, the lineup lock, `CurrentMatchMinute`'s second half and anything that
treats the second-half start as fixed need no change. The longest silence is the 14-minute
break with A1 = 1, under `LiveMatchStaleTimeout` (18 min).
- *Alternative*: a full 15-minute break. Rejected for now. It shifts the whole second half by
  A1 and makes every estimate depend on a stored A1, with no visible benefit.

### 5. Estimate and score-as-of
`common.CurrentMatchMinute(kickoff, a1, a2) (minute, added int)` follows the same schedule.
During the break it returns `(45, a1)`, and after the last tick it clamps to `(90, a2)`. Its
callers (`liveStateAsOfNow`, `GetLiveMatchesOverview`, national feeds) read both added-time
columns from the fixture row they already query. The score-as-of queries become
`WHERE id = $1 AND (minute, added_minute) <= ($2, $3) ORDER BY minute DESC, added_minute DESC, event_id DESC`.

### 6. Visibility bound
Add `common.MaxBroadcastDuration()` = offset(90, `MaxAddedSecondHalf`) and a unit test
`TestLongestBroadcastFitsLiveWindow` asserting
`MaxBroadcastDuration() + 5*time.Minute <= LiveBroadcastWindow`. The engine constants live in
`engine/`, so the test sits in `engine` and imports `common.LiveBroadcastWindow`. The
"~105 minutes" comments in livematch.go, livehub.go, worldcup.go, nationalmatches.go,
liveMatchHandlers.go, useLiveMatch.ts, MatchViewer.tsx and LoadPreviousLineupControl.tsx are
updated to "up to 108 minutes". A frontend test pins `maxLiveBroadcastWindowMs` to the same
2 h.

### 7. Frontend
- `utils/matchMinute.ts`: `formatMatchMinute(minute, addedMinute)` returns `"45+2'"`, and
  `compareMatchMinute` orders minutes.
- Types gain `addedMinute` on `MatchEvent`, `MatchInstant` and `LeagueLiveScore`, and
  `addedTime` on events. Live state in `useLiveMatch` carries `addedMinute`.
- Scrubbers (`useMinuteScrub`, `MatchMinuteRatings`, `MatchMinuteZoneComparison`) work on the
  index into the ordered instants, not on the minute value, and label each step with
  `formatMatchMinute`.
- `liveMatchRatings.ts` and `MatchRatingsTab` compute "minutes so far" as the count of ticks up
  to the selected one. `RATING_FULL_MATCH = 90` stays as the result cap.
- `matchCommentary.tsx` / `MatchEventsReport` render `ADDED_TIME` as the board. New i18n keys:
  `match.addedTime.board` ("+{{count}} minutes" / "+{{count}} minutos", pluralised) and
  `match.addedTime.short`.
- `liveScoreLabel.ts` appends `+N` while `addedMinute > 0`.
- `matchRatings.go` (line ~313) "elapsed" becomes the number of stored instants, at least 90,
  instead of the last minute value.

## Risks / Trade-offs

- [About 7% more goals per match will push the calibration tests (`HOME_ADV_SIM`,
  `MENTALITY_SIM`, set-piece goal totals) and season stats upward] → Run the three calibration
  suites before and after. The proposal's default accepts the rise. If a bound fails, widen it only
  with the owner's approval, or lower the chance rate in added time (one constant).
- [A reader that ignores `added_minute` would merge 45 and 45+1 into one minute, for example a
  `map[int]` keyed by minute in the replay] → Grep every `eventsByMinute`, `minute <=` and
  `ORDER BY minute`. The replay keys by `(minute, added)`, and integration tests cover a match
  with added time end to end.
- [Primary key change on large instant tables] → The `DO` block runs once. Postgres rebuilds the
  index, and the table is append-only and narrow. Run it during a quiet period.
- [The CPU AI's substitution heuristic uses `(90 - m)`] → It slightly undervalues late subs.
  That is acceptable and out of scope.
- [Results still appear 2 h after kickoff in time-gated lists, even though the replay ends at
  103–108 min] → This is unchanged from today (it was 103). The non-goal above is the follow-up
  if the owner wants faster reveals.

## Migration Plan

1. Deploy the migration first: the columns default to 0 and old code ignores them, but the new
   instants primary key also accepts the old inserts.
2. Deploy backend and frontend together. Matches simulated before the deploy replay as before.
   A match whose replay spans the deploy is recovered by the existing stale-broadcast job.
3. Rollback: revert the code. The extra columns and the wider primary key are harmless to the
   old code, because it never writes an added minute other than 0.

## Code map for the new stoppage-time spec

Carried into `openspec/specs/stoppage-time/spec.md` at archive:

- Time lost and added minutes: `backend/engine/stoppage_time.go` (`Stoppage*`, `MinAddedMinutes`,
  `MaxAddedFirstHalf`, `MaxAddedSecondHalf`, `timeLost`, `addedMinutesFor`,
  `firstHalfAddedMinutes`, `secondHalfAddedMinutes`); tests in `engine/stoppage_time_test.go`.
- Simulation and board: `engine.playMatchMinutes` / `announceAddedTime` (`engine/simulation.go`),
  `common.ADDED_TIME`, `Event.AddedMinute` / `AddedTime`, `MatchInstant.AddedMinute`.
- Storage: `resources/sql/init107_stoppage_time.sql`; `MatchResult.FirstHalfAddedTime` /
  `SecondHalfAddedTime` in `common/matchresult.go`; round trip in
  `common/stoppage_time_integration_test.go`.
- Bound: `engine.MaxBroadcastDuration` and `TestLongestBroadcastFitsLiveWindow`
  (`engine/livematch_test.go`) against `common.LiveBroadcastWindow`; the frontend's
  `maxLiveBroadcastWindowMs` pinned in `api/useLiveMatch.test.ts`.
- Results hidden until the final whistle: `TestStoppageTime_LongestMatchStaysHiddenUntilTheWindowCloses`
  (`common/stoppage_time_integration_test.go`).
- Notation: `frontend/src/utils/matchMinute.ts`.

