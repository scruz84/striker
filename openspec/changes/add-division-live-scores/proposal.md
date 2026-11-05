# Proposal

## Why

While a matchday is being broadcast, a manager can only follow their own match. The league page
replaces every other score with a "LIVE" chip (so it doesn't spoil anything), and the table
reverts to how it stood before the matchday. Nowhere in the game can you see how the rest of your
division is going, or what the table would look like if the scores stayed as they are, which is
the most exciting part of a matchday in a football manager.

The backend already knows the broadcast state of every match: `common.liveScoreAsOfNow` reads the
last published message of the match's live channel, which is what the dashboard's suggested match
uses. That state is never the stored final result, so it can be shown without spoiling anything.

## What Changes

- **Division live scores endpoint.** `GET /v1/teams/division/:id/live` returns every league match
  of that division inside its live window (from 15 minutes before kickoff to 2 hours after), with
  the broadcast's minute, score and status (`pending`, `live`, `halftime`, `finished`). Scores come
  only from the broadcast, never from `matches.home_goals/away_goals`. Cached for 5 seconds per
  league.
- **"Rest of the matchday" in the dashboard's live hero.** When the owner's live match is a league
  match, the hero gets a section listing the other matches of the same matchday with their live
  score and minute. It starts collapsed on phones, showing a one-line summary ("7 matches · 12
  goals"), and expanded on desktop. It is not shown for cup ties or friendlies.
- **Live scores on the league page's matchday cards.** A match inside its live window shows its
  broadcast score and minute ("63'", "Half-time", "FT") instead of a bare "LIVE" chip. If the
  endpoint fails, the page falls back to the chip it shows today.
- **Live table.** On the current season's standings, while at least one match of the matchday is
  in its live window, a "Live" switch recomputes the table from the pre-matchday standings plus
  the broadcast scores. It shows ↑/↓ position changes against the pre-matchday table and marks
  the rows of teams that are playing.
- Polling: both views refresh every 15 seconds, and only while there is something live.

## Capabilities

### New Capabilities

- `division-live-scores`: the per-division live scores endpoint (window, statuses, broadcast-only
  scores, cache), and the client polling rules shared by every view that shows it.

### Modified Capabilities

- `league-page`: matchday cards show broadcast scores instead of the bare LIVE chip; new live table
  switch on the standings.
- `team-dashboard`: the live hero gets a "Rest of the matchday" section for league matches.

## Impact

- Backend: `common/livematches.go` (a new `GetLeagueLiveScores`, and `liveScoreAsOfNow` extended
  to report half-time), `common/cache.go` (a short-TTL setter), `handlers/statsHandlers.go`
  (handler), `service/fmservice.go` (route). No migration, no engine job, no money movement, no
  notification.
- Frontend: `api/hooks.ts` and `api/types.ts` (hook and type), a new shared
  `components/DivisionLiveScores.tsx`, `features/dashboard/DashboardPage.tsx` (`LiveMatchHero`),
  `features/division/DivisionDetailsPage.tsx` (matchday cards and standings), and en/es locales.
- Load: one indexed query plus up to one Redis `LRANGE` per live match, per league, at most every
  5 seconds, and only while that league has something live.

## Out of scope

- A live table on the dashboard (the team header's position stays the official one).
- Live scores for cup ties, other divisions, or a world-wide "every live match" page.
- A live-scores panel inside the match viewer itself ("other grounds"). It can reuse the same
  component later.
- Push or in-app notifications for goals in other matches.
- Past seasons: the live table and live scores only apply to the current season.

## Open questions for the owner

1. **Default state of the live table switch.** This proposal turns it **on** by default while
   something is live, since that is the point of the feature. Turning it off by default keeps the
   official table first. The choice is not persisted between visits.
2. **Whether to include the owner's own match in the hero list.** This proposal excludes it,
   because the hero already shows it from the WebSocket, and two sources a few seconds apart
   could briefly disagree.
