# Proposal

## Why

The match chronicle is the part of a match a manager actually reads, and it repeats itself.
Each event type has 1-4 one-sentence phrasings (`match.commentary.*`), picked by a hash that can
land on the same line twice in a row. Nothing reacts to the story of the match: an equaliser in
the 90th minute reads exactly like the third goal of a 5-0, a second yellow reads like any red
card, and the report closes with the same "the referee blows the final whistle" whatever
happened. More phrasings, lines that read the match situation, and a few longer sentences
would make every match feel different and give the chronicle some humour.

## What Changes

- **The keeper by name**: open-play goal and miss lines may name the defending goalkeeper
  (`{{keeper}}`, stored by `add-open-play-keeper`). When the event has no keeper, they fall back
  to "the goalkeeper" / "el portero", as set pieces already do.
- **Much bigger phrasing pools** in Spanish and English, written natively in each language
  rather than translated: at least 12 lines for an open-play goal and an open-play miss, and
  6-8 for every other event type (see the `match-commentary` spec for the exact minimums).
  The tone stays a radio commentator's: energetic, sometimes funny, never mocking a real
  group of people.
- **Situation-aware goal lines**: a goal also gets a second sentence when it is the opener,
  an equaliser, a go-ahead goal, a late winner or equaliser (minute 85 or later, or
  second-half added time), part of a comeback, a brace, a hat-trick, a rout or a
  consolation. The situation is worked out only from events up to and including that goal.
- **New special lines** for a second yellow (a red card for a player already booked in the
  match) and for a late miss by a side that is level or one goal behind.
- **Colour sentences**: 30% of missed chances (when no late-miss sentence applies) and 25% of
  yellow cards get a short extra sentence about the crowd, the bench or the ball, so the text is
  longer and less mechanical. Goals always get their situation sentence instead.
- **No repeats**: within one match, a phrasing pool cycles through all its lines before it
  reuses one. The choice depends only on the match and the earlier events, so a line already
  shown live never changes when later events arrive or after full time.
- **Bookend lines**: several kickoff lines, a new half-time line with the score at the break,
  and full-time lines chosen by the result (goalless draw, scoring draw, home win, away win,
  win by 3 or more). An abandoned or default-result match keeps the neutral full-time line.
- **Spoiler safety is unchanged**: red cards and injuries stay hidden while a match is live, and
  no line shown live may reveal one (for example, a substitution line never says the player
  coming off is injured).

### Out of scope

- New data from the engine beyond the defending keeper: assists, shot types, the zone a chance
  came from. The keeper on open-play chances comes from the separate change
  `add-open-play-keeper`, which this change builds on. Old matches get the richer commentary
  too, and their lines fall back to "the goalkeeper" where no keeper was stored.
- Notification texts, the season awards and any text outside the match chronicle.
- Commentary in languages other than Spanish and English.

### Open questions for the owner

- **Tone**: is light humour fine everywhere, or should a match marked as a Clásico or a cup
  final stay serious? The plan assumes humour is fine everywhere.

## Capabilities

### New Capabilities
- `match-commentary`: how the chronicle turns each event into text. Covers phrasing pools and
  their minimum sizes, situation detection, colour sentences, the no-repeat rule, the bookend
  lines, and spoiler safety for commentary text.

### Modified Capabilities
- `match-viewer`: the chronicle gains a half-time line, live and finished. The live view shows
  it only once the first half is over.

## Impact

- Frontend only: `utils/matchCommentary.tsx` (situation detection, picker, colour sentences),
  `features/match/MatchEventsReport.tsx` (bookend lines, half-time line), `MatchViewer.tsx`
  (passes whether the first half is over), locale files `i18n/locales/{en,es}.json`
  (`match.commentary.*`, `match.report.*`).
- No backend, API, schema or migration change in this change (the keeper comes from
  `add-open-play-keeper`). No change to what is visible while a match is
  live.
- Tests: `utils/matchCommentary.test.ts`, `features/match/MatchEventsReport.test.tsx`, plus a
  locale test that checks the pool minimums and placeholders in both languages.
