# Proposal

## Why

The national-teams page (`/national-teams`) is broken in three ways an owner reported from a
phone screenshot, and a fourth found while investigating:

1. **Not every match is shown.** Both feeds (`/friendlies/upcoming`, `/friendlies/recent`) are
   capped at 10 rows (`nationalTeamFriendliesLimit`), but since `add-all-fifa-countries` about 100
   friendlies are played every Saturday. The "En juego" box is built from that same 10-row
   recent feed, so it also shows at most 10 of the live matches.
2. **A country plays twice on the same evening.** A friendly is simulated (and flips `played`) up
   to 15 minutes before kickoff. The 5-minute pairing pass then sees that country as free and
   re-pairs it; `NextLocalSaturdayKickoff` returns *today* 21:00 in the new home side's timezone
   whenever that is still in the future. Observed: Nigeria vs Ghana and São Tomé vs Nigeria;
   São Tomé vs Djibouti and São Tomé vs Nigeria (same home timezone, same kickoff instant);
   Northern Ireland vs Latvia and Northern Ireland vs Ireland.
3. **The live box spoils the result.** `LiveNationalTeamMatches` renders `homeGoals/awayGoals`
   from the REST row, which is the final simulated score, while the match page shows the
   broadcast's score at the current minute. That breaks the hard rule that nothing may reveal a
   result before the live replay finishes.
4. **World Cup group tables spoil results too.** `world_cup_group_teams` is updated in the
   simulation transaction, and `GetWorldCupGroups` returns it as is, so points and goal
   difference move as soon as a group match is simulated, up to 2 hours before its broadcast ends.

The page also gives little information and reads badly on a phone (team names wrap over three
lines, no kickoff grouping, no way to find one country, no history). Fixing the feeds is the
moment to restructure it.

## What Changes

**Bug fixes**
- Friendly pairing respects a rest rule: a country's next friendly kicks off on the first local
  Saturday that is at least 6 days after both countries' latest national-team kickoff, so a
  country plays at most one national-team match per international week.
- The national-team API never returns a result before its broadcast ends: a match inside the
  live window is reported as `live` with the score at the current minute, never the final score.
- World Cup group standings count only matches whose broadcast has finished; a live group
  match's score is masked the same way.
- The match feeds are no longer capped at 10: the current round lists every fixture, and past
  results are paginated.

**Page improvements**
- `/national-teams` becomes three tabs: **Jornada** (this international round: live, upcoming and
  just-finished matches, grouped by kickoff time, with live minute), **Resultados** (paginated
  history of every national-team match, friendlies and World Cup alike) and **Mundial**.
- Search by country and filter by region on Jornada and Resultados.
- "My players" highlight: matches where the signed-in owner has a player in a saved call-up show
  a badge with the count, plus a toggle to show only those.
- Compact two-line match rows that fit a 360 px screen (competition and kickoff on top; flag,
  name with ellipsis, score below).
- World Cup tab: full group tables (P W D L GF GA GD Pts), fixtures grouped by matchday with
  dates, the list of past World Cups and their champions, and an estimate of when the next World
  Cup starts when none is running.
- National team page (`/teams/:id` of a national team) gains a **Partidos** tab: upcoming
  fixture, last results and the last-5 form strip.
- **BREAKING (internal API)**: `GET /v1/national-teams/friendlies/upcoming` and `/recent` are
  replaced by `GET /v1/national-teams/round` and `GET /v1/national-teams/results`. The only
  consumer is this frontend.

**Out of scope**
- A national-team ranking (Elo/FIFA-style) - needs a new rating model; a candidate follow-up.
- Following a country and notifications for its matches.
- Continental tournaments, managers controlling national teams, changes to squad selection.
- Cleaning up the double-booked fixtures already played: they are history and stay as they are.

**Open questions for the owner**
- Rest rule of 6 days (one match per international week) - or allow a second "double-header"
  match on Tuesdays like real international windows? This proposal assumes one per week.
- Results page size: 25 rows per page is assumed.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `national-teams`: friendly pairing gains the rest rule; the national teams page requirement is
  replaced by the round/results feeds, live-safe scores, filters, "my players" badges and the
  national team's matches tab.
- `world-cup`: group standings and scores become broadcast-safe; the display requirement gains
  full tables, matchday grouping, World Cup history and the next-World-Cup estimate.

## Impact

- Backend: `common/nationalteam.go` (pairing kickoff rule, new round/results queries),
  `common/timezones.go` (Saturday-after helper), `common/worldcup.go` (standings derived from
  finished broadcasts, history query), `common/livematches.go` (reuse of the score-as-of-minute
  enrichment), `handlers/nationalTeamHandlers.go` and the router (new endpoints, removed ones).
- Frontend: `features/nationalteams/NationalTeamsPage.tsx` (rewritten into tabs and components),
  `api/hooks.ts`, `api/types.ts`, `features/team/TeamDetailsPage.tsx` (new tab), en/es locales.
- No schema migration, no new notification, no money movement.
- Specs: `national-teams` and `world-cup` requirements and Code maps.
