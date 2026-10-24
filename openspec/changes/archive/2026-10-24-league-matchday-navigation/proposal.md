# Proposal

## Why

Since the division page redesign (#280) the league page only shows two fixed matchdays: the last
one played and the next one. A manager cannot look back at earlier results or ahead at later
fixtures without opening team pages one by one. On phones, the third tab is labelled "Goleadores"
but also holds the recent-transfers panel, so transfers sit under a label that does not describe
them.

## What Changes

- The league page's fixtures section keeps its default view (the last matchday played and the next
  one, side by side on desktop, stacked on phones). A matchday navigator above the two cards
  (‹ previous, a week picker, › next, and "back to current") moves the pair one matchday at a time
  or jumps to any matchday of the current season.
- Card titles become "Jornada N" with a "Anterior" / "Próxima" chip on the two default matchdays.
- New public endpoint `GET /v1/teams/division/:id/weeks/:week` returns one matchday's matches for
  the league's current season. The page reuses `previousWeekMatches` / `nextWeekMatches` from the
  division details response for the default pair, so a page load does not make an extra request.
- Results on browsed matchdays follow the same spoiler rule as today: a match still inside its
  live-broadcast window shows a LIVE chip instead of its score.
- Mobile tabs become four: "Tabla", "Jornadas", "Goleadores", "Traspasos". The scorers tab holds
  only the top-scorers panel, and the recent-transfers panel moves to the new tab. The table tab
  label is shortened from "Clasificación" to "Tabla" so the four tabs fit at 360px. The desktop
  layout is unchanged.

## Capabilities

### New Capabilities
- `league-page`: the league (division) details page: the standings table, matchday fixtures and
  matchday navigation, top scorers, recent transfers, and how they are laid out on desktop and on
  phones.

### Modified Capabilities
<!-- None: no existing spec covers the league page. The spoiler rule it applies is the existing
     match-scheduling "Spoiler-free side effects" requirement, which does not change. -->

## Impact

- Backend: new handler in `backend/handlers/statsHandlers.go` (or next to `DivisionDetailsHandler`),
  route in `backend/service/fmservice.go`, reusing `common.WeekMatches` and
  `common.GetCurrentSeasonForLeague`, with a short-lived `ObjectCache` entry. No schema change and
  no migration. No engine job is involved, so no replica-safety work is needed.
- Frontend: `features/division/DivisionDetailsPage.tsx` (navigator, card titles, four mobile tabs),
  `api/hooks.ts` (`useLeagueWeekMatches`, query key), `i18n/locales/{en,es}.json`. The mobile smoke
  test `frontend/e2e/mobile-smoke.spec.ts` already visits `/division/:id` and checks tap areas.
- No change to the dashboard, which also renders `RecentLeagueTransfers`.

## Out of scope

- Browsing previous seasons' matchdays. The endpoint serves only the current season.
- Changing the desktop layout of the standings, top scorers or recent transfers.
- Matchday navigation on the team page or the match viewer.

## Open questions

- Should the navigator remember the last matchday viewed per league (for example in the URL as
  `?week=N`)? The plan puts it in the URL query so a browsed matchday can be shared and survives a
  reload. Drop this if the owner prefers the page to always open on the default pair.
