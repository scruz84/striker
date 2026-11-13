# Design

## Context

- `utils/matchCommentary.tsx` `commentaryFor(event, t, ...)` narrates one event at a time. It
  has no view of the rest of the match. It picks a variant with `pickVariantIndex(minute, type,
  setPiece, playerId)`, a hash, so two events can land on the same line.
- `features/match/MatchEventsReport.tsx` renders the chronicle for both the live view
  (`MatchViewer` passes `withholdSpoilerEvents(live.events)`, which has no red cards or injuries)
  and the finished report (all events, `final`). It hides minor misses behind the "full
  commentary" toggle. Kickoff and full time are fixed strings, `match.report.kickoff` and
  `match.report.fullTime`.
- Each event stores `homeGoals`/`awayGoals` as the score after it, plus `minute`, `addedMinute`,
  `teamOwner`, the player ids and names, and, for set pieces only, the defending keeper. That is
  enough to work out every situation in the spec. With `add-open-play-keeper`, open-play goals
  and misses list the keeper too. Old events and events from before that change fall back to
  `match.setPiece.theKeeper`, as `commentaryFor` already does for set pieces.
- `useLiveMatch` exposes `halftime` (the broadcast is paused for the break) and `minute`, which
  is what the half-time line needs live.

## Goals / Non-Goals

**Goals:**
- One pure function that narrates a whole event list, so situations and the no-repeat rule can
  see earlier events, and that can be unit-tested without rendering.
- Live and finished output identical for every event both show.

**Non-Goals:**
- No change to which events the chronicle shows or hides.
- No commentary cache or stored line choice: everything is recomputed from the events.

## Decisions

### 1. Narrate the match, not the event
Add `narrateMatch(events, ctx)` in `utils/matchCommentary.tsx`. It takes all events in match
order (`compareMatchMinute`), before the "key events only" filter, plus the team ids and names.
It returns one entry per event: the main line's pool key and variant index, plus an optional
tail (`situation`, `lateMiss` or `colour`) with its own key and index. Placeholder values are
filled in too. `MatchEventsReport` calls it once (memoised on `events`) and renders the visible
entries through the existing `interpolateNodes`, so player and team links keep working.
`commentaryFor` stays as a thin wrapper for any single-event caller, and its existing tests
keep passing.

*Alternative*: pass a running match state into `commentaryFor` call by call. Rejected: the
picker state would live in the component, and the toggle could reorder it.

### 2. Pool keys
- Event pools: keep `match.commentary.{goal,noGoal,yellowCard,redCard,injury,substitution,...}`
  and add `match.commentary.secondYellow`.
- Tails: `match.commentary.situation.{hatTrick,lateWinner,lateEqualiser,comeback,brace,
  equaliser,goAhead,opener,rout,extendLead,pullsOneBack,consolation}`,
  `match.commentary.lateMiss`, `match.commentary.colour.{miss,yellowCard}`.
- Bookends: `match.report.kickoff` becomes an array. Add `match.report.halfTime.{goalless,level,
  homeLeads,awayLeads}` and `match.report.fullTimeResult.{goallessDraw,scoringDraw,homeWin,
  awayWin,bigWin}`. Keep `match.report.fullTime` as the neutral line for abandoned and
  default-result matches.
- The penalty and free-kick goal pools get the same situation tails as open-play goals.

### 3. Situation detection
Walk the events once, tracking each player's goals so far and, per side, whether it has
trailed at any point. For a goal by side S with score `f`-`a` after it (from S's point of
view), with `late = minute >= 85` (a 90+N minute is stored as minute 90, so it counts):
the first match in this order wins: `goals >= 3` hatTrick; `late && f == a+1`
lateWinner; `late && f == a` lateEqualiser; `f == a+1 && trailed(S)` comeback; `goals == 2`
brace; `f+a == 1` opener; `f == a` equaliser; `f == a+1` goAhead; `f-a >= 4` rout; `f-a >= 2`
extendLead; `a-f == 1` pullsOneBack; otherwise consolation. `trailed(S)` only reads goals
before this one. A second yellow is a red card whose player has an earlier yellow card in
the list. A late miss is a NO_GOAL at `minute >= 85` where `a - f` is 0 or 1 for the side that
missed.

### 4. Deterministic, non-repeating picker
For each pool key, build a permutation of its indices with a small seeded PRNG (mulberry32)
seeded from `hash(homeId, awayId, kickoff, poolKey)`. The kickoff time (`matchKey`, the view's
`matchDate`) tells a rematch apart: with the teams alone, every meeting of the same two sides read
the same way. It is known before kickoff, so live and finished agree. The k-th use of that pool in the match takes
`perm[k % n]`. Every line of a pool is used once before any repeats. Two different fixtures
usually start in different places, and the choice depends only on earlier events.
Colour-sentence rolls use a separate hash of `(homeId, awayId, kickoff, minute, addedMinute, eventType,
playerId)` mod 100 against 30 (miss) or 25 (yellow). That roll does not consume a pool use
unless it succeeds.

*Alternative*: keep the per-event hash and add more lines. Rejected: with 12 lines and 6 goals
there is still a ~75% chance of a repeat (birthday problem).

### 5. Spoiler safety by construction
`redCard`, `secondYellow` and `injury` pools are used only by those event types, and those
events add no tails and update no state that other events read. Dropping them from the list
(live) therefore changes nothing for other events. A test narrates the same match with and
without them and compares the other entries. The substitution pool has no "injured" wording,
and the locale test rejects lines containing "lesion"/"injur" in that pool.

### 6. Half-time and full-time lines
`MatchEventsReport` gets a `firstHalfOver` prop. `MatchViewer` passes `live.halftime ||
live.minute > 45` (a first-half added minute keeps minute 45). Finished views pass `true`
unless the event list has an ABANDONED event at minute 45 or earlier. The line is inserted
before the first event with `minute > 45`, or at the end if there is none yet. The half-time
score is the score after the last event with `minute <= 45`. The full-time line uses the score
after the last event (0-0 when there are no events). Its pool index comes from
`hash(homeId, awayId, kickoff)` because it is used once per match.

### 7. Writing the lines
Each language is written on its own, not translated, in a radio commentator's voice. Spanish
follows the existing locale's Spain register ("portero", "amarilla", "el colegiado").
Humour comes from the game (the ball in the stands, a frustrated bench, the kit man), never
from nationality, appearance or a real group of people. Lines stay at one or two short
sentences so a phone screen does not wrap them into a wall of text. Grammatical gender stays
neutral where Spanish allows it ("el jugador de {{team}}", "{{player}}"), since player names
carry no gender.

## Risks / Trade-offs

- [Text gets long on a phone] → situation tails are one short sentence, colour tails are
  limited to 30% / 25%, and minor misses stay collapsed by default.
- [A situation phrase reads wrong for an edge case, such as "comeback" after 0-1 then 2-1] →
  that is a comeback by the definition. Tests pin each case with the score sequences from the
  spec scenarios.
- [Locale files grow by about 200 lines per language] → acceptable. The locale test keeps the
  structure honest in both languages.
- [The no-repeat permutation depends on the pool size] → adding lines later reshuffles which
  line an old match shows. That is fine: nothing stores the choice and the live/finished
  consistency holds within one deploy.

## Migration Plan

Frontend-only and stateless: no migration file, no TRANSACTION_* reason, no notification key.
Deploy with the frontend. Roll back by reverting the commit.

## Code map (for the `match-commentary` spec created at archive)

- Narration: `frontend/src/utils/matchCommentary.tsx`: `narrateMatch` (whole match, in match
  order), `goalSituation`, `GOAL_SITUATIONS`, `LATE_MINUTE`, `LinePicker` (seeded, non-repeating),
  `colourRoll` (`MISS_COLOUR_PERCENT`, `YELLOW_COLOUR_PERCENT`), `kickoffLine`, `halfTimeLine`,
  `fullTimeLine` (`halfTimeCase`, `fullTimeCase`); `commentaryFor` stays for a single event.
- Rendering: `features/match/MatchEventsReport.tsx` (narrates once, then applies the "key events
  only" filter; kickoff, half-time and full-time lines; `matchKey` from `MatchViewer` /
  `MatchReportTab`).
- Lines: `i18n/locales/{en,es}.json` under `match.commentary.*` (event pools, `secondYellow`,
  `situation.*`, `lateMiss`, `colour.*`) and `match.report.*` (`kickoff`, `halfTime.*`,
  `fullTimeResult.*`, neutral `fullTime`).
- Keeper: `{{keeper}}` comes from `event.otherTeamPlayerNames` (see match-simulation "Defending
  keeper on every chance"), falling back to `match.setPiece.theKeeper`.
- Tests: `utils/matchCommentary.test.ts`, `utils/matchCommentaryLocales.test.ts` (pool minimums,
  placeholders, keeper position, no injury in substitutions), `features/match/MatchEventsReport.test.tsx`,
  `features/match/MatchViewer.test.tsx`.
