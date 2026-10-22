# Proposal

## Why

Every goal in Striker today comes from the same open-play chance, so a match report never shows
the two set pieces that decide many real matches: a penalty given away in the box and a goal
scored directly from a free kick. Adding them makes reports read more like real football and gives
a squad's best finisher and its dead-ball specialist a role of their own. The real-world rates
below are low enough that we can add both without changing how many goals a match produces.

## What Changes

- A share of the attacking chances the simulator already generates becomes a **penalty** or a
  **direct free kick** instead of an open-play chance. The number of chances per match stays the
  same, so the team that dominates midfield still wins more penalties, as in real football.
- Each set piece has its own taker and its own conversion curve, based on the taker against the
  opposing goalkeeper:
  - Penalty: the on-pitch outfield player with the highest Shot takes it. It is scored about 78%
    of the time in an even matchup, bounded to 60%–92%.
  - Direct free kick: the best combination of Shot and Pass takes it. It is scored about 6% of the
    time in an even matchup, bounded to 2%–15%.
- Set-piece outcomes reuse the existing `GOAL` / `NO_GOAL` events, plus a new `setPiece` field
  (0 open play, 1 penalty, 2 direct free kick). The defending goalkeeper is recorded as the
  event's other-team player. Top scorers, season awards, player goal counts, match ratings and the
  live scoreboard therefore keep counting penalty and free-kick goals without any change.
- The match report and live commentary show set-piece goals and misses with their own phrasing,
  and the goal list marks them `(pen.)` / `(f.k.)` (`(p.)` / `(f.d.)` in Spanish).
- A migration adds `set_piece` to `match_events`, `cup_tie_events` and `friendly_events`. It
  defaults to 0, so existing rows read as open play.

### Real-world reference (used to pick the constants)

| Metric | Real world | Target in Striker |
|---|---|---|
| Penalties awarded per match | 0.24 (Premier League before VAR) – 0.41 (after VAR, La Liga) | ~0.28 |
| Penalty conversion | ~78% long-run, 85–90% in recent Premier League seasons; ~12–19% saved | 78% even matchup |
| Share of goals from penalties | ~8–12% | ~7–8% |
| Direct free-kick shots per match | ~0.8–0.9 (Premier League: 301–337 a season over 380 matches) | ~0.82 |
| Direct free-kick conversion | ~4.7–5.7% | 6% even matchup |
| Direct free-kick goals per match | ~0.04–0.05 (14.7 a season since 2020/21) | ~0.05 |

### Out of scope

- A card for the foul that gave away the penalty (for example, a red card for denying an obvious
  goal-scoring opportunity). Cards stay on the independent 3%-per-minute discipline roll.
- A designated penalty or free-kick taker chosen by the manager in the lineup screen.
- Indirect free kicks, corners and headed set-piece goals.
- Penalty shootouts. Cup ties still settle drawn ties as they do today (`engine/cups.go`).
- Any change to the existing chance-share, conversion or clear-cut constants.

### Owner decisions (2026-10-22)

1. A penalty foul does not book the defender. Cards stay on the independent discipline roll, so
   card rates are unchanged.
2. The penalty baseline is 78%, the long-run real-world figure.
3. Takers are picked automatically from skills. Manager-chosen takers are left for a later change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-simulation`: "Chance resolution" gains set-piece chances (penalty and direct free kick),
  each with its own taker selection and conversion curve. A new requirement covers how set-piece
  events are recorded.

## Impact

- Backend engine: `engine/scorer_simulation.go` (set-piece roll, taker selection, conversion),
  `engine/database.go` (persisting the new column), `engine/cups.go` and `engine/friendlies.go`
  through their event inserts.
- Backend common: `common/fmconstants.go` (`SET_PIECE_*` constants), `common/fmstructs.go`
  (`Event.SetPiece`), and the event INSERT/SELECT statements in `common/database.go`,
  `common/cups.go` and `common/friendlies.go`.
- Schema: new migration `resources/sql/init82_match_event_set_piece.sql`.
- Frontend: `api/types.ts`, `utils/matchCommentary.tsx`, `utils/matchGoals.ts`,
  `features/match/MatchEventsReport.tsx`, plus new strings in `i18n/locales/{en,es}.json`.
- Not affected: finances, notifications, standings, top scorers and season awards. They count
  `GOAL` events, and set-piece goals are still `GOAL` events.
