# Design

## Context

See proposal.md - Why for the four defects. Facts that shape the approach:

- National-team fixtures live in three places: weekly friendlies and World Cup group matches in
  `friendly_matches` (group matches linked through `world_cup_group_matches`), knockout ties in
  `cup_ties`. Each has its own live channel (`FriendlyMatchChannel`, cup-tie channel).
- Matches are simulated up to 15 minutes before kickoff (`GetSimulableFriendlies`), and their
  broadcast runs ~105 minutes; `LiveBroadcastWindow` (2 h) / `LiveBroadcastCutoff()` is the
  existing "definitely finished" boundary used by head-to-head, top scorers, cup resolution.
- `common.enrichSuggestedLiveMatch` already turns a live match into "minute + score as of that
  minute", from the live hub's last message or, failing that, `CurrentMatchMinute` plus
  `get*ScoreAsOfMinuteStmtSql` over `match_events`.
- `world_cup_group_teams` is written inside the simulation transaction
  (`UpdateWorldCupStandingsAfterMatchTx`) and the knockout draw depends on it, so it must keep
  being written there.
- ~210 active countries means ~105 friendlies per Saturday; feeds currently cap at 10.

## Goals / Non-Goals

**Goals:**
- One broadcast-safe "national match" shape shared by every national-team feed.
- A pairing rule that makes same-evening double-booking impossible, not just unlikely.
- Masking done on the server, so no client can leak a result by forgetting a check.

**Non-Goals:**
- Changing how matches are simulated, when call-ups happen, or what the World Cup engine does.
- Real-time push of feed updates: the page polls, as it does today.

## Decisions

### 1. Rest rule computed from the latest kickoff, not from `played`

`scheduleNationalTeamFriendliesLocked` loads, in one query, each eligible country's latest
national-team kickoff across `friendly_matches` and `cup_ties`
(`getLatestNationalTeamKickoffsStmtSql`, `MAX(match_date)` grouped by country). For a pair it
calls a new `NextLocalSaturdayKickoffAfter(home, hour, minute, earliest)` where
`earliest = max(now, lastKickoff(home) + 6 days, lastKickoff(away) + 6 days)`.
`NextLocalSaturdayKickoff` becomes `NextLocalSaturdayKickoffAfter(..., time.Now())`, so the World
Cup group scheduler keeps its behaviour.

*Alternatives:* treating a match as pending until its broadcast ends (`played = false OR
match_date > cutoff`) only delays re-pairing by 2 hours - a country in UTC-8 paired at 22:00 UTC
with a home side in UTC-10 can still land on the same Saturday. A DB constraint cannot express
"one per local Saturday". The rest rule is simple and covers both.

Six days rather than seven: local Saturday 21:00 differs by up to ~26 h between timezones, so
a 7-day rule could push a country two weeks out; 6 days still excludes the same weekend.

### 2. One `NationalMatch` view model built on the server

```
NationalMatch {
  kind: "friendly" | "worldCupGroup" | "worldCupKnockout"
  id (matchId or tieId), link kind
  homeTeamId, homeTeamName, homeCountry, awayTeamId, awayTeamName, awayCountry
  kickoff
  status: "scheduled" | "live" | "finished"
  minute?            // live only
  homeGoals?, awayGoals?   // absent when scheduled
  groupName?, round? // world cup context for the label
  myPlayers?         // signed-in owner only, see 5
}
```

A single `common.BuildNationalMatchStatus(dbCtx, *NationalMatch, played bool, finalHome, finalAway)`
applies the rule: not played → `scheduled`, no score; played and `kickoff > LiveBroadcastCutoff()`
→ `live`, minute and score via the same code path as `enrichSuggestedLiveMatch` (refactored into
`liveScoreAsOfNow(dbCtx, kind, id, kickoff)` and reused by both); otherwise `finished` with the
final score. Frontend never sees a final score it would have to hide.

*Alternative:* keep returning raw rows and mask in React with `mightStillBeLive`. That is exactly
what produced defect 3; the hard rule is easier to keep in one Go function with a unit test.

### 3. Endpoints

- `GET /v1/national-teams/round` - union of three queries (national friendlies not linked to a
  group, group matches, knockout ties of the active World Cup's cup), filtered by
  `played = false OR match_date > now() - interval '24 hours'`, ordered by `match_date`. Uncapped:
  at most one unplayed fixture per country plus a day of results keeps it near 250 rows.
- `GET /v1/national-teams/results?page=&country=` - same union restricted to
  `played AND match_date <= LiveBroadcastCutoff()`, `ORDER BY match_date DESC, id DESC`,
  `LIMIT 26 OFFSET page*25` (the 26th row only sets `hasMore`). `country` resolves to the national
  team id and filters `home_team_id = $id OR away_team_id = $id`, which uses the existing
  `friendly_matches_home_idx`/`_away_idx`. Unfiltered, the friendly branch restricts
  `home_team_id = ANY(national team ids)` so it also walks the index. No migration.
- `GET /v1/national-teams/world-cups` - `world_cups` joined with the champion team.
- `world-cup/current` gains `nextWorldCupEstimate` (only when none is active and
  `GetActiveCountries >= MinWorldCupCountries`): `season.last_international_cup_check +
  InternationalCupCadence * cyclesToNextMultipleOf(WorldCupFrequencySeasons)`.
- `friendlies/upcoming` and `friendlies/recent` are removed with their handlers, hooks and types.

Redis cost of the live branch: one `LRANGE -1 -1` per live match (≤ ~110 on a Saturday), done
only for rows inside the window. Acceptable for a page polled every 30 s; if it shows up in the
observability dashboard, add a 10 s in-process cache keyed by the round response.

### 4. Broadcast-safe World Cup standings

`GetWorldCupGroups` stops reading `world_cup_group_teams` for display. It already loads every
group match; it derives each team's record in Go from the matches that are played and whose
kickoff is at or before `LiveBroadcastCutoff()`, then ranks with the existing
`RankGroupStandings`. The stored table stays the source for the knockout draw (which runs only
after every match is played, and through the cup job, long after broadcasts end). Group matches
are returned as `NationalMatch` rows through the same status function.

*Alternative:* delay `UpdateWorldCupStandingsAfterMatchTx` until after the broadcast - touches
the simulation path and the knockout trigger; deriving for display is local and reversible.

### 5. "My players" count

The `/national-teams` route group already runs behind the JWT middleware, so the handler knows
the user; when that user owns a club, one extra query: for the user's team id, count
`friendly_lineup_positions`/`cup_tie_lineup_positions` rows of players whose `team_id` is the
owner's club, grouped by match, for the match ids in the round. Saved lineups exist only within
24 h of kickoff, so the badge appears with the call-up notification, never earlier.

### 6. Frontend structure

`features/nationalteams/` splits into `NationalTeamsPage.tsx` (tabs in `?tab=`),
`RoundTab.tsx`, `ResultsTab.tsx`, `WorldCupTab.tsx`, `NationalMatchRow.tsx` (the compact
two-line row, shared with the team page tab), `NationalMatchFilters.tsx` (search + region chips +
"only my players" toggle, filtering client-side for the round and passing `country` to the server
for results; region from the existing `regionForCountry`). Round groups rows by kickoff in the
viewer's timezone, live first. `useNationalTeamRound` polls every 30 s while any row is live or
due within 15 minutes, else every 5 min. `useNationalTeamResults` is an infinite query.
`TeamDetailsPage` adds a `matches` tab for national teams, built from the round (next fixture)
and `results?country=` (last 10, form strip from the first 5).

No new notification, TRANSACTION_* reason or migration.

## Risks / Trade-offs

- [Rest rule leaves a country idle for a week when paired late on a Saturday] → that is the
  intended cadence; the only change is that it no longer gets a second match that evening.
- [Two endpoints removed] → the frontend is the only consumer and ships in the same release;
  a stale cached client gets a 404 on the old feed and the next deploy reload fixes it.
- [Derived standings differ from the stored table if they ever drift] → add a test asserting
  that, once every broadcast has finished, the derived table equals `world_cup_group_teams`.
- [Round feed size on the first Saturday after a World Cup] → bounded by one unplayed fixture per
  country; measured in the integration test with 210 countries.

## Migration Plan

Deploy backend and frontend together. No data migration: double-booked fixtures already played
stay in history; any still-unplayed duplicate fixture is legitimate under the new rule only if a
week apart, so none is cancelled. Rollback is a plain revert.
