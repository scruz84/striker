# Design

## Context

See proposal.md (Why) for the gap table.

**Frontend.** The three pages share leaf components (`Scoreboard`, `MatchEventsReport`,
`MatchReportTab`, `MatchRatingsTab`, `Lineups`, `MatchMinuteZoneComparison`,
`MatchMinuteRatings` and `HeadToHeadCard`). The page shells around them are duplicated: the
phase switch, the live-completion effect, the live layout and the finished tabs. Cup and friendly
pages already squeeze their payload into `WeekMatch` through `cupTieToWeekMatch` and
`friendlyToWeekMatch`, filling the fields they don't have with blanks.

**Backend.** All three competitions run through the same `engine.runMatchSimulation`, so
instants, weather and attendance already exist in memory for every match. What differs is what
gets persisted afterwards:

- `simulateMatch` stores instants (`match_instants`) and flavor (`common.UpdateMatchFlavor` on
  `matches`).
- `simulateFriendly` stores instants in `friendly_instants` but no flavor.
  `creditFriendlyTicketRevenue` draws its own weather and drops it.
- `simulateCupTie` stores neither. It passes a fresh `common.RandomWeather()` into
  `UpdateTeamFinancesAfterMatch` and drops the attendance that call returns.

Cup ties and friendlies have their own id sequences. That is why `friendly_instants` exists
instead of writing to `match_instants`, and cup ties need their own table for the same reason.

## Goals / Non-Goals

**Goals:**
- One component tree renders every match. A route-specific adapter is the only place where
  competitions differ.
- Cup ties and friendlies persist exactly what league matches persist, so the viewer never needs
  "this competition doesn't have X" branches. The only gaps left are for matches played before
  the migration.

**Non-Goals:**
- No change to the `WeekMatch`, `CupTieDetails` or `FriendlyMatch` field names that already
  exist. New fields are additive and `omitempty`.
- No new REST routes and no WebSocket payload changes. The live message already carries lineups
  and full player records for both sides, which is enough to compute goalkeeper ratings live.

## Decisions

### 1. A view model plus a single `MatchViewer`, with routes kept

New file `features/match/matchViewModel.ts` defines `MatchView`. It holds the ids and localized
names; the home stadium name; the kickoff date; `played`; the flavor fields; events; instants;
lineups and players; ratings; the scores; a `subtitleLabel` string; a `liveUrl`; the `queryKey`
to invalidate on completion; and optional `penaltyWinnerName` / `wentToPenalties`. There are three
pure adapters:

- `fromWeekMatch(match, division, t)`
- `fromCupTie(tie, cup, t)`
- `fromFriendly(match, t)`

These replace `cupTieToWeekMatch` and `friendlyToWeekMatch`.

New file `features/match/MatchViewer.tsx` owns the logic and layout that are duplicated today:

- the phase switch (`isPast` / `mightStillBeLive` / `liveComplete`);
- the `LiveView`, including the completion effect;
- the `ScheduledView`;
- the `FinishedView` with its tabs.

It takes the `MatchView` plus a `lineupSection?: ReactNode`. Each route page builds that section
with its own hooks (`useMatchLineup`, `useCupTieLineup`, `useFriendlyLineup`), and only when
`userTeam` is one of the two sides. The viewer shows the section only in the scheduled view.

Each route page becomes about 40 lines: fetch, build the view model, then render `MatchViewer`.
For cup ties, a bye renders `ByeCard` instead.

Alternatives considered:
- *One route, `/match/:kind/:id`.* Rejected. It breaks every stored notification link and every
  link already in brackets, team pages and the World Cup page, for no gain to the viewer.
- *Keep adapting everything into `WeekMatch`.* Rejected. `WeekMatch` is the league DTO, and
  stretching it hides competition-specific data such as penalties and labels behind dummy zeros.
  That is how the pages drifted in the first place.
- *A backend "unified match" endpoint.* Rejected for now. It would need a three-way union query
  or a new table, which the proposal puts out of scope. The adapters keep the change
  frontend-only on the read side, apart from a few added fields.

### 2. Competition label

- **League.** `match.matchday` ("Jornada N de T"), as today. `weeksInASeason` moves into the
  adapter.
- **Cup tie.** The adapter calls the existing `useCupDetails(tie.cupId)`. It then reuses
  `translateCupName`, `roundName`, `expectedTotalRounds` and `isFirstRoundPlayIn` from
  `CupPage.tsx`, so the label matches the bracket's own round names. No backend change is
  needed. While cup details are loading, the subtitle falls back to the date alone.
- **Friendly.** `FriendlyMatch` gains `worldCupGroupName` and `worldCupRound` (both `omitempty`),
  filled by a `LEFT JOIN world_cup_group_matches` / `world_cup_groups` in
  `GetFriendlyMatchByIdStmtSql`. The label is `match.competition.worldCupGroup` ("Mundial ·
  Grupo {{group}} · Jornada {{round}}") when they're set, and `match.competition.friendly`
  ("Amistoso") otherwise.

`matchSubtitle` changes to take `(label, matchDate, weather)`, so it no longer depends on
`WeekMatch`.

### 3. Persisting instants and flavor for cup ties and friendlies

- **Migration** `resources/sql/init83_match_viewer_parity.sql`, which can be re-run:
  - `CREATE TABLE IF NOT EXISTS cup_tie_instants`, with the same columns as `friendly_instants`,
    `tie_id bigint NOT NULL REFERENCES cup_ties(tie_id)` and `PRIMARY KEY (tie_id, minute)`.
  - `ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS` for `weather varchar(20)`,
    `attendance int`, `referee_name varchar(120)`, `assistant1_name varchar(120)` and
    `assistant2_name varchar(120)`, matching the flavor columns of `matches` in `init.sql`.
  - The same `ALTER TABLE` on `friendly_matches`.
- **SQL and helpers.**
  - `common.SetCupTiePlayed` gains an `instants []MatchInstant` parameter and inserts them
    through a new `InsertCupTieInstantStmtSql`, mirroring `SetFriendlyPlayed`.
  - New `GetCupTieInstants`.
  - `common/matchflavor.go` gains `UpdateCupTieFlavorStmtSql` / `UpdateCupTieFlavor` and
    `UpdateFriendlyFlavorStmtSql` / `UpdateFriendlyFlavor`.
- **Engine.**
  - `simulateCupTie` draws `weather := common.RandomWeather()` once, keeps the attendance that
    `UpdateTeamFinancesAfterMatch(..., weather)` returns, and passes both to a generalized
    `updateMatchReportFlavor`.
  - `creditFriendlyTicketRevenue` takes `weather` as a parameter instead of drawing it.
    `simulateFriendly` draws it once and stores flavor the same way.
  - `updateMatchReportFlavor` takes a store function, `func(weather, attendance, ref, a1, a2)
    error`, so the three competitions share the name-drawing and logging, and failures stay
    best-effort as they are today.
- **Read side.**
  - `CupTieDetails` gains `MatchInstants`, `HomeStadiumName` and the five flavor fields.
  - `FriendlyMatch` gains `HomeStadiumName` and the five flavor fields.
  - The stadium name comes from `teams.stadium_name` of the home side, read at fetch time, the
    same way `WeekMatch.HomeStadiumName` does it. A national team has no stadium, so the column
    is NULL and the caption is hidden.
  - Instants and flavor are filled only when the match is played, matching `WeekMatch` and
    `EnrichPlayedFriendly`. This exposes nothing new before the replay, because these appear only
    on the match page itself, which the spoiler rule allows.

There are no new `TRANSACTION_*` reasons and no new notification keys. Ticket revenue keeps its
existing reason, and only the weather that feeds it is now kept instead of thrown away.

### 4. Zone duel: percentage plus strength in the same cell

`DuelCell` renders two lines in each half:

- the share, `body2`, bold when stronger (as today);
- the raw strength, `caption` (12 px), `Math.round`, at 0.85 opacity.

The tooltip changes from `formatValue` (an integer or one decimal) to always one decimal. The
help text in `match.positionBattle.help` is updated to explain both numbers.

Why under the percentage and not beside it: the earlier version printed the raw values at about
9.5 px and they were removed as unreadable on a phone (see the `DuelCell` comment). On a 360 px
screen each half-cell is about 50 px wide. `60% · 43` at `body2` needs about 58 px, while two
stacked lines need about 30 px each. The vertical layout also keeps the percentage as the primary
reading. The cell height grows by about 14 px, which keeps the duel at roughly the height of the
lineup pitch next to it on desktop.

Alternative considered: a separate per-team ratings table (Hattrick style) under the pitch. It
would hold the same nine numbers twice and would lose the duel pairing that makes them
meaningful. A central zone sums two or three players, so its raw value only makes sense next to
its direct opponent's value.

### 5. Goalkeeper live

`LiveView` passes `goalkeeperRating(live.homeLineup, live.homePlayers)` (and the away
equivalent) instead of `0` once the lineups have arrived, and `0` before that. Both views read the
starting goalkeeper, so minute 90 matches across the switch, as the spec requires.

## Risks / Trade-offs

- [Cup ties and friendlies played before the migration have no instants or flavor] → The viewer
  already hides the zone duel and the flavor lines when the data is missing. The spec states this
  explicitly.
- [The `SetCupTiePlayed` signature change touches its only caller and its tests] → It's a small,
  compile-checked change. The engine tests in `engine/cups_test.go` exercise it.
- [The cup label needs a second request (`/v1/cups/:id`)] → It's already cached by react-query
  from the bracket page in the usual navigation path, and the subtitle degrades to the date
  alone while it loads.
- [Refactoring three pages at once could regress the live/finished switch, which is
  issue-prone (#31, #37)] → The phase logic moves verbatim into `MatchViewer`. Its existing
  comments move with it, and `MatchDetailsPage.test.ts` gets cases for cup and friendly view
  models.
- [Every visible cell now also shows the raw strength, so there is more on screen] → The
  percentage keeps visual priority (bold `body2` against a muted `caption`).

## Migration Plan

1. Deploy the backend together with `init83_match_viewer_parity.sql`. The migration is additive
   only, and older frontends ignore the new JSON fields.
2. Deploy the frontend.

Rollback: revert the frontend. The extra columns and the table can stay, because nothing older
reads them.

## Open Questions

- Placement of the strength figure (under the percentage, which is the recommended default, or
  beside it) and the World Cup label wording. See the proposal's open questions. Either answer
  only changes `DuelCell` styling or an i18n string, not the plan.
