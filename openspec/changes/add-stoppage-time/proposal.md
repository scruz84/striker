# Proposal

## Why

Every match runs exactly 90 minutes, however much time was lost to goals, substitutions, cards
and injuries. Real football adds that time back at the end of each half. Adding it makes the
broadcast feel real, and it gives a late goal one more chance to happen. Longer matches touch
every place that hides a result until its replay ends, so the timing rules need to be stated
and checked again, not assumed.

## What Changes

- At the end of each half, the engine works out the time lost in that half from its own events
  (goals, substitutions, injuries, cards, penalties, plus a fixed base). It rounds that to whole
  minutes and clamps it to 1–5 minutes per half.
- The engine simulates those added minutes with the normal per-minute pipeline: chances, cards,
  injuries, conditional orders and the forfeit check. Scheduled substitutions stay limited to
  minutes 1–90. Events in added time do not add more time.
- A minute is now a clock minute plus an added minute (`45+2`, `90+4`). Events and per-minute
  instants store both. Old matches read as added minute 0.
- A new `ADDED_TIME` event (type 9) records the fourth official's board at minute 45 and minute
  90. Each fixture also stores the added time of both halves.
- The live replay plays the added minutes at one per real minute. The half-time break gets
  shorter by the first-half added time, so the second half still starts 59 minutes after
  kickoff. The final whistle moves from kickoff + 103 minutes to at most kickoff + 108.
- **Visibility**: the result must still stay hidden until the replay ends. The 2-hour
  live-broadcast window is kept as the only time-based gate. A new rule says the longest
  possible replay must end at least 5 minutes inside that window, and a test checks it. Every
  gate that reads the window (standings, head-to-head, top scorers, career stats, cup and World
  Cup progression, national-team feeds, post-match effects, season rollover) stays as it is.
  Gates driven by the broadcast (notifications, dashboard hero, live scores' `finished`) follow
  the real final whistle.
- Division live scores, the dashboard hero and the match viewer show added-time minutes as
  `45+2'` / `90+3'`. The viewer's chronicle shows the board ("+3 minutes"). The minute
  scrubbers cover the added minutes too.
- Match ratings count added minutes as minutes played. The result component stays capped at
  90 minutes.

## Capabilities

### New Capabilities
- `stoppage-time`: how the added time of each half is computed, announced and bounded,
  including the rule that the longest replay must fit inside the live-broadcast window.

### Modified Capabilities
- `match-simulation`: the per-minute pipeline runs through each half's added minutes; orders
  stay checkable during added time; the stored record has one instant per simulated minute,
  added minutes included; fatigue is capped at the full-time value.
- `match-scheduling`: the live replay paces the added minutes and the shorter half-time break;
  the spoiler-free rule names the real final whistle.
- `division-live-scores`: entries carry an added minute; the time-based estimate follows the
  new schedule.
- `match-viewer`: minute labels with added time, the added-time board in the chronicle, and
  scrubbers over the added minutes.
- `match-ratings`: added minutes count toward minutes played.

## Impact

- Backend: `engine/simulation.go` (minute loop), the new `engine/stoppage_time.go`, the
  substitution, stamina, injury and discipline stages (they read the minute),
  `engine/livematch.go`, `common/livebroadcast.go` (`CurrentMatchMinute`),
  `common/livematches.go` (score as of a minute), `common/matchresult.go` and the event and
  instant persistence, `common/matchratings.go`, and the `common.Event` / `common.MatchInstant`
  structs.
- Schema: a new migration `init107_stoppage_time.sql`. It adds `added_minute` to the six
  event/instant tables (and makes it part of the instants' primary key), `added_time` to the
  three event tables, and `first_half_added_time` / `second_half_added_time` to `matches`,
  `cup_ties` and `friendly_matches`.
- Frontend: a shared minute formatter. Users of minutes: scoreboard, commentary, events
  report, live scoreboard clock, rating and zone scrubbers, `useMinuteScrub`, live score
  labels, `liveMatchRatings.ts`, `MatchRatingsTab`. New strings in `en.json` / `es.json`.
- Calibration: goals per match rise by about 7% (~6–8 more simulated minutes). The
  home-advantage, mentality and set-piece calibration bounds must be checked again.
- **Out of scope**: extra time in cup ties (a level tie still goes straight to penalties),
  time-wasting by a leading side, VAR, letting players schedule substitutions in added time,
  and adding stoppage time to the pre-match expected-goals estimate.
- **Owner decisions**:
  1. Decided: the half-time break absorbs the first-half added time (break = 15 − first-half
     added minutes, 10–14 minutes), so the second half keeps its current start time.
  2. Decided: at most 5 added minutes in each half.
  3. Still open, defaulted here: added minutes use the normal chance rate, so the ~7% rise in
     goals is accepted rather than offset by a lower per-minute chance rate.
  4. Still open, defaulted here: the cost of each stoppage (seconds lost per goal,
     substitution, etc.) in the stoppage-time spec.
