# Design

## Context

- `GET /v1/teams/division/:id/` (`handlers.DivisionDetailsHandler` → `engine.GetDivisionDetails`,
  cached 1 minute under `classDet_key<id>`) returns the standings plus `previousWeekMatches`
  (week W) and `nextWeekMatches` (week W+1). W is the top-ranked team's `matches`. The matches are
  loaded with `common.WeekMatches(dbCtx, leagueId, season, week)`, which already serves any week.
- `features/division/DivisionDetailsPage.tsx` renders two `FixturesCard`s titled
  `division.lastWeek` / `division.nextWeek`. On phones it renders three tabs, and the third one
  ("scorers") stacks `TopScorers` and `RecentLeagueTransfers`.
- `FixturesCard` already hides still-live scores (`mightStillBeLive`, `isPast`).
  `hideStillLiveResults` reverts the table using `previousWeekMatches` only. That stays correct,
  because only week W can still be inside the broadcast window.
- The client already derives T as `weeksInASeason(teamCount)` (`features/match/matchViewModel.ts`),
  and the page subtitle uses the same formula.
- The theme sets `MuiTab` `minHeight: 48` and sentence-case labels. A `fullWidth` MUI tab keeps
  its default `minWidth: 90` and `padding: 12px 16px`.

## Goals / Non-Goals

**Goals:**
- Browse any matchday of the current season with one small, cacheable request per matchday.
- No extra request on page load, and the same spoiler behavior on every matchday.

**Non-Goals:**
- A "whole season fixtures" payload, or a new server-side `totalWeeks` field.
- Refactoring the standings table or `hideStillLiveResults`.

## Decisions

1. **One endpoint per matchday: `GET /v1/teams/division/:id/weeks/:week`.**
   The handler is `LeagueWeekMatchesHandler` in `backend/handlers/statsHandlers.go`, next to the
   other public league reads. It parses `:id` and `:week` (400 if `:id` is not an integer or
   `:week` < 1), resolves `common.GetCurrentSeasonForLeague`, and calls
   `engine.GetLeagueWeekMatches(dbCtx, leagueId, season, week)`. That function wraps
   `common.WeekMatches` with an `ObjectCache.SetValue` (1 minute) entry under a new key
   `leagueWeek_key<league>_<season>_<week>` in `backend/engine/catalog.go`. The route is
   registered in `backend/service/fmservice.go` beside `/top-scorers`. A week past the season
   naturally returns `[]`.
   - *Alternative:* return every matchday in one call (about 240 rows for a 16-team league).
     This was rejected because it costs a heavier payload on every load for data most visits never
     look at, and it would still need the per-week spoiler handling on the client.
   - *Alternative:* add `?week=` to the division details endpoint. This was rejected because it
     would split the 1-minute standings cache per week and recompute the table on each browse.
   - The cache stays at 1 minute even for weeks already played, so a week that becomes played
     never serves a stale "unplayed" copy for longer than division details does.

2. **Seed React Query from division details.** The page adds a `useLeagueWeekMatches(leagueId,
   week, seed?)` hook in `api/hooks.ts`, with query key `['leagueWeek', leagueId, week]`. The page
   passes `initialData` from `previousWeekMatches` when `week === W` and from `nextWeekMatches`
   when `week === W+1`, with `initialDataUpdatedAt` set to the division query's `dataUpdatedAt`.
   When the division query refetches, the seed follows. The default pair therefore never hits
   `/weeks/`. Other weeks use the normal `staleTime`.
   - W used for seeding is the backend's W, which is `positions[0].matches` before
     `hideStillLiveResults`. The "current week" used for the default pair and the chips is the
     reverted W that the subtitle already shows, so a still-live matchday is not labelled
     "Anterior" too early. When the two differ (a match within its live window), the default
     pair is (W_reverted, W_reverted+1) = (W−1, W). Week W is then seeded from
     `previousWeekMatches`, and week W−1 is a single fetch.

3. **Navigator state lives in the URL** (`useSearchParams`, `week` = L). The default pair removes
   the parameter, and changes use `replace: true` so the back button does not step through
   every matchday. The pair index is clamped as the spec states.

4. **Navigator UI:** a `MatchdayNavigator` component in the same file. It holds a ‹
   `IconButton`, a `TextField select` (`size="small"`) listing "Jornada 1…T", a › `IconButton`,
   and a text `Button` "back to current" shown only off the default. All controls get a 44 px
   tap area (the smoke test checks this). On desktop it sits above the two-column grid. On
   phones it sits at the top of the "Jornadas" tab. `FixturesCard` takes `week`, `chip?` and
   `matches | undefined` (a skeleton while loading). Its title becomes `division.matchday`
   ("Jornada {{week}}"), and `division.lastWeek` / `division.nextWeek` are reused as chip labels.

5. **Four mobile tabs that fit at 360 px.** The usable width is 328 px after the 16 px gutters,
   which leaves 82 px per tab. The tabs get `sx={{ minWidth: 0, px: 1 }}`, and the standings tab
   label becomes `division.tabs.table` = "Tabla" / "Table". The longest Spanish label is
   "Goleadores", about 75 px at 14 px plus 16 px padding, so it fits. `division.tabs.transfers`
   = "Traspasos" / "Transfers" is added, and `division.tabs.scorers` becomes "Scorers" in English
   (Spanish stays "Goleadores"). The `aria-label` of the standings table keeps using
   `division.tabs.table`. `DivisionTab` gains `'transfers'`.
   - *Alternative:* `variant="scrollable"` with the long labels. This was rejected because the
     fourth tab would be off-screen on phones, where it is hard to discover.

Specs touched: a new `league-page` capability. There is no migration, no `TRANSACTION_*` reason
and no notification key. The engine is not involved, so no replica-safety claim is needed.

## Risks / Trade-offs

- [The client-side T formula drifts from the scheduler] → It is the same `weeksInASeason` already
  used by the subtitle and the match viewer. A week beyond the real season just shows "Sin
  partidos", because the endpoint returns `[]`.
- [Rapid ‹/› taps fire many requests] → Each week is one small cached query. React Query
  deduplicates and keeps the previous pair visible (`placeholderData: keepPreviousData`) while
  the next one loads.
- [The spoiler window on a browsed week] → Only W can be in the window, and `FixturesCard`
  applies `mightStillBeLive` per match whatever the week. A unit test covers a browsed week that
  contains a still-live match.
- [A shortened "Tabla" label reads less formal] → The page title and the desktop layout are
  unchanged. Only the phone tab strip uses it.

## Migration Plan

Frontend and backend deploy together. The new route is additive. An old client never calls it, and
a new client on an old backend only fails when it browses (the cards show an error state), never
on the default pair. Rollback is to revert the commit.
