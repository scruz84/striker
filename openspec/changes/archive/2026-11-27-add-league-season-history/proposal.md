# Proposal

## Why

When a country rolls its season over, every classification row is reset to zero and the new
table is ordered by points, goal difference, goals scored, wins and draws. All of those are 0 on
matchday 0, so the order is whatever Postgres returns: the table loses how the teams finished the
season before. A promoted team can open at the top and the champion halfway down.

And once a season is over, its table is gone from the league page. `season_history` keeps each
team's final position and promotion/relegation outcome for the team's own history tab. It does
not keep points or goals, and no page shows a whole division's table for an earlier season, or
its matchdays.

## What Changes

- **Starting order.** At rollover each league gets a starting order (`classification.seed`,
  1..N). Teams relegated into the league come first, then the teams that stayed in it, then the
  teams promoted into it. Each group keeps the order it finished in last season. `seed` is the last
  tie-break of every standings ordering (backend and frontend), so on matchday 0 the table reads in
  that order, and later it only settles otherwise exact ties.
- **The season already under way** gets its `seed` from the same migration, rebuilt from the
  previous season's `season_history`, so the current table is fixed as soon as it ships.
- **Full final tables.** `season_history` also stores each team's matches, points, wins, draws,
  losses, goals scored and goals received. The rollover fills them, and a migration backfills
  every past season from its played league matches.
- **Past seasons on the league page.** A season picker on the standings ("Season 3 (current)",
  "Season 2", ...) kept in a `season` query parameter. A past season shows its final table
  (champion, real promotions and relegations) and its matchdays, read-only.
- **Links from the team history.** Each season in a team's history tab links to that division's
  table for that season.
- **New endpoints:** `GET /v1/teams/division/:id/seasons` and
  `GET /v1/teams/division/:id/seasons/:season`. `GET /v1/teams/division/:id/weeks/:week` takes
  an optional `season` query parameter.

Out of scope:
- Top scorers and recent transfers of past seasons: in a past season the page shows only the
  table and the matchdays.
- Team names as they were: a past table shows each club's current name.
- Cup brackets of past seasons.

Decided by the owner:
- Relegated teams start at the top of the lower division, promoted teams at the bottom of the
  upper one.
- Past-season matchdays are included, not just the table.

Open questions for the owner: none.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `league-page`: starting order, past-season table and matchdays, season picker, new endpoints.

## Impact

- Backend: migration `init101_league_season_history.sql`; `engine/season.go` (rollover),
  a new pure `engine/seasonseeds.go`; `common/database.go` (standings orderings),
  `common/seasonhistory.go`; new handlers and routes in `service/fmservice.go`.
- Frontend: `features/division/DivisionDetailsPage.tsx` (season picker, past-season mode, sort),
  `features/team/TeamDetailsPage.tsx` (history links), `api/` hooks and types, en/es strings.
- Docs: `openspec/specs/league-page/spec.md` purpose and code map on archive.
