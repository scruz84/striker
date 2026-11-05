# Design

## Context

See proposal.md for the motivation. Here is what exists today and constrains the approach:

- `common.liveScoreAsOfNow(dbCtx, kind, id, kickoff)` (`backend/common/livematches.go`) already
  gives the minute and score a viewer of one match's broadcast sees right now. It reads the tail of
  that match's Redis history list (`LiveMatchHub.LastMessage`, one `LRANGE -1 -1`). If there is
  nothing to read, it falls back to `CurrentMatchMinute(kickoff)` plus the score as of that minute
  from `match_events`. It reports `complete` but not half-time, and it does not look at `played`.
  Only the dashboard's suggested match uses it (`enrichSuggestedLiveMatch`).
- All matches of one league matchday share a kickoff (league kickoff schedule, see
  match-scheduling), but a match re-dated by admin or recovery can differ. That is why the
  endpoint selects by kickoff window rather than by week.
- The league page already hides still-live results. `hideStillLiveResults`
  (`frontend/src/features/division/DivisionDetailsPage.tsx`) reverts every played match of
  `previousWeekMatches` inside `mightStillBeLive`'s window (15 min before kickoff to 2 h after)
  from the standings and re-sorts them. The matchday cards show a LIVE chip for the same set.
- `Cache` (`backend/common/cache.go`) only offers fixed TTLs: 1 minute, 30 minutes, or none. A
  live-score cache needs a few seconds.
- Spoiler rule (project context): nothing outside a match page may reveal a result before that
  match's replay finishes. This feature shows the broadcast state itself, which is exactly what
  the match page shows at that moment, so it is consistent with that rule. The design keeps it
  that way structurally (decision 1).

## Goals / Non-Goals

**Goals:**
- One read-only endpoint whose cost does not grow with the number of viewers of a league.
- No code path in the endpoint can read `matches.home_goals`/`away_goals`.
- The live table is a pure, unit-tested client-side function on top of the data the page already
  has.

**Non-Goals:**
- No WebSocket fan-out per division. One socket per match is what exists, and opening 8 sockets
  per viewer is what this design avoids.
- No server-side live table. The standings stay one backend truth, and the live view is derived.

## Decisions

### 1. The endpoint's query never selects goals

`getLeagueLiveMatchesStmtSql` selects `match_id, week_number, match_date, played, home_team_id,
home name, away_team_id, away name` from `matches` joined to `teams`, with
`WHERE m.league_id = $1 AND m.match_date > $2 AND m.match_date <= $3 ORDER BY m.match_date,
m.match_id`. `$2` is `LiveBroadcastCutoff()` (now − 2 h) and `$3` is now + 15 min (a new
`common.liveEarlySimulationWindow = 15 * time.Minute`, mirroring the frontend's
`maxEarlySimulationMs`). Every goal figure in the response therefore comes from the live state
lookup.

*Alternative considered:* reuse `getLiveLeagueMatchesStmtSql` with a league filter. Rejected: it
selects the stored goals, and it excludes unplayed and not-yet-kicked-off matches, which the spec
reports as `pending`.

*Index:* the existing `matches_idx2 (league_id, week_number)` narrows to the league, about 240
rows per season. With the 5-second cache, that is cheap enough, so **no migration**. If many
seasons accumulate, an `(league_id, match_date)` index is the follow-up.

### 2. `liveStateAsOfNow` with an explicit status

Extract `liveStateAsOfNow(dbCtx, kind, id, kickoff, played) (LiveState, error)`, where
`LiveState{Status, Minute, HomeGoals, AwayGoals}`:

- If `!played` or `now < kickoff`, return `pending`, 0, 0-0, without touching Redis.
- If there is a last message with a minute (`minuteFromLiveMessage`): `halftime` → `halftime`,
  `complete` → `finished`, anything else → `live`, with that message's goals.
- Otherwise, the existing fallback (estimate the minute and take the score from events) →
  `live`.

`liveScoreAsOfNow` stays as a thin wrapper (it passes `played = true`), so
`enrichSuggestedLiveMatch` and its tests are unchanged. The status constants are
`LiveStatusPending/Live/Halftime/Finished` in `common`.

### 3. A 5-second per-league cache

Add `SetValueFor(key, value string, ttl time.Duration)` to the `Cache` interface and to
`RedisCache`, and update the four in-memory test fakes (`handlers/teamLoansHandler_test.go`,
`engine/scorer_simulation_test.go`, `engine/match_viewer_parity_integration_test.go`,
`common/market_listings_integration_test.go`). The key is `league_live_scores:<leagueId>` and the
value is the JSON response. On a hit, `GetLeagueLiveScores` returns the cached value. On a miss,
it computes the response and stores it for 5 s. A cache read or write error never fails the
request.

*Alternative considered:* no cache. A response costs 1 query plus about 8 `LRANGE` calls, and
with 15-second polling a 200-viewer league would make about 13 requests per second. That is
affordable, but the cache makes the cost independent of viewers for one small interface method.

*Alternative considered:* reuse the 1-minute `SetValue`. Rejected: a minute-old score is wrong.

### 4. Handler and route

`LeagueLiveScoresHandler` lives in `handlers/statsHandlers.go`, beside `LeagueWeekMatchesHandler`
(same id parsing, 400 on a non-numeric id). The route is
`v1TeamGroup.GET("/division/:id/live", …)` in `service/fmservice.go`, in the same authenticated
group with no ownership check. It returns `[]` (never `null`) when empty.

### 5. Frontend data layer

- `api/types.ts`: `LeagueLiveScore { matchId, weekNumber, matchDate, home, homeName, away,
  awayName, status: 'pending' | 'live' | 'halftime' | 'finished', minute, homeGoals, awayGoals }`.
  The `home`/`away` naming matches `WeekMatch`.
- `api/hooks.ts`: `useDivisionLiveScores(leagueId, enabled)`, with key
  `queryKeys.divisionLiveScores(id)`. `refetchInterval` is a function that returns `false` when
  the data is empty or every entry is `finished`, and 15 000 otherwise. Views treat `isError` as
  "no live scores" (React Query's default retries run first, so one dropped request does not
  flicker the UI).
- `utils/liveScoreLabel.ts`: maps an entry to its label ("63'" via `dashboard.liveMatchMinute`,
  `match.halftime`, `liveScores.fullTime`, or the kickoff time) and chip colour. The hero and the
  matchday cards share it.

### 6. Dashboard hero

`LiveMatchHero` gets a `divisionId` prop, which comes from `team.divisionId` (a league match is
always in the owner's own division). It calls `useDivisionLiveScores(divisionId, match.kind ===
'league')`, finds its own `matchId` to read the `weekNumber`, and passes the other entries of that
week to a new `components/DivisionLiveScores.tsx`. That component is a compact list: badge, name
and goals per side, a state chip, and a `RouterLink` to `/match/:id`. Colours use `inherit` so the
list reads on `errorContainer`. The collapse uses MUI `Collapse` behind a text button showing the
summary, and its initial state comes from `useMediaQuery(theme.breakpoints.up('md'))`. If the own
match is not found in the response, the section is hidden.

### 7. League page

- `enabled` = current season && `previousWeekMatches.some(m => m.played &&
  mightStillBeLive(m.matchDate))`. That is the same set `hideStillLiveResults` reverts, so the
  table and the cards agree on what is live.
- `FixturesCard` receives a `Map<matchId, LeagueLiveScore>`. For `stillLive` rows, it shows the
  entry's goals and label chip when the status is not `pending`, and the existing LIVE chip
  otherwise.
- Live table: a new pure module `features/division/liveTable.ts` with
  `applyLiveScores(revertedPositions, previousWeekMatches, liveById)`. It returns
  `{ positions, deltas: Map<teamId, number>, playing: Set<teamId> }`. Only matches that
  `hideStillLiveResults` would revert and whose entry is not `pending` are applied. The sort
  comparator in `hideStillLiveResults` is extracted to a shared `compareClassification` so both
  orderings stay identical (seed and team id as the last tie-breaks).
- `ClassificationTable` gets an optional `live` prop for the ▲N/▼N markers and the pulsing dot
  (reusing `utils/liveChip.ts`'s pulse). `zoneOf` already works on the row's position, so the zones
  follow live positions with no extra work. The "Live" `Switch` sits beside the season picker,
  uses local `useState(true)`, and is rendered only when at least one entry is `live`, `halftime`
  or `finished`.

### 8. Strings

New keys in `i18n/locales/{en,es}.json`:

- `liveScores.restOfMatchday`: "Rest of the matchday" / "Resto de la jornada"
- `liveScores.matchCount_one` / `_other`: "{{count}} match(es)" / "{{count}} partido(s)"
- `liveScores.goalCount_one` / `_other`: "{{count}} goal(s)" / "{{count}} gol(es)"
- `liveScores.summary`: "{{matches}} · {{goals}}"
- `liveScores.fullTime`: "FT" / "Final"
- `division.liveTable`: "Live" / "En directo"
- `division.positionUp` / `division.positionDown`: aria labels "Up {{count}}" / "Sube {{count}}",
  "Down {{count}}" / "Baja {{count}}"

`match.halftime` and `dashboard.liveMatchMinute` are reused.

### What this change does not add

- No migration file (no schema change).
- No `TRANSACTION_*` reason (no money moves).
- No notification key and no `notificationtranslations.go` entry.
- No engine job, so the engine-replicas rules do not apply. The endpoint is read-only and runs in
  `--svc`.

## Risks / Trade-offs

- [A cached response is up to 5 s stale, plus up to 15 s of polling] → The scores lag the
  match's own page by up to about 20 s. That is acceptable for a glance view. The owner's own
  match in the hero stays on the WebSocket, which is why it is excluded from the list.
- [Fallback estimate drifts from a stalled broadcast] → This only applies before the first message
  or with Redis down, the same trade-off the dashboard suggestion already makes.
  `RecoverStaleLiveBroadcasts` eventually publishes `complete`.
- [Live table disagrees with the official table after the window] → By design they converge.
  Once every match is past 2 h, `hideStillLiveResults` reverts nothing and the switch disappears,
  and by then the stored results equal the broadcast finals.
- [A re-dated match from another week shows up in the window] → The hero filters by
  `weekNumber`. The league page only applies entries that `hideStillLiveResults` reverted, so a
  stray entry cannot double-count.
- [The `Cache` interface change touches test fakes in three packages] → It is mechanical, and
  `go vet ./...` catches any fake that was missed.

## Migration Plan

Additive only. The backend ships the endpoint, and the frontend starts calling it. Older
frontends never call it. Rollback means reverting the frontend, and the unused endpoint is
harmless.
