# Design

## Context

- `DashboardPage` builds everything from `useTeamMatches` (league, cup and friendly rows as
  `TeamMatchSummary`). `nextMatch` is the earliest row with `!isPast(matchDate)`, so a match whose
  kickoff has passed silently drops out of the hero even while it is still broadcasting.
- The live state is shown today by `LiveMatchBanner` (an info `Alert`, gated on
  `played && mightStillBeLive`) and by `LastResultCard`'s LIVE chip.
- `TeamMatchSummary.homeGoals/awayGoals` are the *final* result as soon as `played` flips, so they
  cannot be shown during the broadcast. The broadcast itself is available through
  `useLiveMatch(path)` (WebSocket; catches up on everything already published, then streams
  `minute`, `homeGoals`, `awayGoals`, `halftime`, `complete`, `giveUp`). The viewer's
  `matchViewModel` already maps each kind to its socket path
  (`/v1/teams/match/:id/live`, `/v1/cups/ties/:id/live`, `/v1/friendlies/:id/live`).
- `theme.ts` builds the MUI palette from an M3 `Scheme` and exposes `primaryContainer` etc., but not
  `errorContainer` / `onErrorContainer`.
- League subtitle: `fromWeekMatch` builds `match.matchday` ("Jornada N de T") from the
  `DivisionDetails` the league route already fetches (`division.name` is available, e.g. "Primera.1").

## Goals / Non-Goals

**Goals:**
- A spoiler-safe live hero driven by the same broadcast the viewer uses, so the dashboard and the
  viewer always agree on score and minute.
- No backend, API or schema change.

**Non-Goals:**
- Reworking `selectRecentForm` (the form strip keeps its 2-hour window rule).
- Live data for other teams' matches on the dashboard (the worldwide card is unchanged).

## Decisions

1. **Live score from the WebSocket, not a new REST endpoint.** The hero calls
   `useLiveMatch(liveMatchPath(match), enabled)` for the selected live match. Alternative: a
   `/v1/teams/team/:id/matches/live` endpoint reusing `enrichSuggestedLiveMatch` and polled every
   5 s. Rejected: it needs backend code and tests for something the socket already delivers in real
   time, and polling would lag the viewer by up to 5 s. The socket is only opened while a candidate
   live match exists, so idle dashboards open nothing.

2. **Live candidate selection is a pure, exported helper.** `selectLiveMatch(teamMatches)` returns
   the row with the latest kickoff among those where `isPast(matchDate) && mightStillBeLive(matchDate)`
   (kickoff reached, within the 2-hour window). It does not require `played`: a match whose kickoff
   passed but whose simulation is lagging is still "on" for the owner, and the socket will deliver it
   when it starts. Unit-tested like `selectRecentForm`.

3. **Completion handled in the page, not the hero.** `DashboardPage` owns the
   `useLiveMatch` state (via a small `useOwnLiveMatch` hook returning `{ match, live }`) so it can
   decide the hero (live vs. next) and the "Last result" source from one place. Once
   `live.complete` is true the candidate is treated as finished: the hero renders `NextMatchHero`,
   and `teamMatches` is invalidated so "Last result" picks up the final score. Completion is
   remembered per `kind:matchId` in component state so the hero does not flip back to live on a
   re-render while the 2-hour window is still open.

4. **"Last result" excludes only the match currently shown as live.** `lastMatch` becomes the most
   recent `played && isPast` row whose `kind:matchId` differs from the active live match. After
   completion the finished match qualifies again and shows its real score (its broadcast is over, so
   it is no longer a spoiler). `LastResultCard`'s `stillLive` LIVE-chip branch is no longer reachable
   for the live match and is removed together with `LiveMatchBanner`.

5. **Colour via M3 error container.** Add `errorContainer` / `onErrorContainer` to the palette
   module augmentation and `buildPalette` (`hex(scheme.errorContainer)` etc.), mirroring the
   existing container tokens. Alternative: a hand-picked `red[50]`. Rejected: it would not adapt to
   dark mode or the generated scheme, and M3 container pairs are designed to meet contrast. The
   "LIVE" label reuses the pulsing animation from `MatchScoreboard` (extract the `sx` into a shared
   constant rather than copying it).

6. **Shared live-path helper.** Add `liveMatchPath(match: Pick<TeamMatchSummary,'kind'|'matchId'>)`
   to `utils/matchSummary.ts` next to `matchSummaryLink`, and use it from `matchViewModel` too, so
   the dashboard and viewer cannot drift.

7. **Subtitle label.** In `fromWeekMatch`, when `division` is loaded, the label becomes
   `` `${division.name} · ${t('match.matchday', …)}` `` (same separator `matchSubtitle` uses), so no
   new i18n key is required for the label itself. While `division` is loading, the existing
   `division.weekChip` fallback stays.

8. **i18n keys added** (en/es): `dashboard.liveNow` ("Live now" / "En directo"). The hero reuses
   existing keys for the rest: `match.halftime`, `dashboard.liveMatchMinute`, `dashboard.watchLive`.
   `dashboard.liveMatchBanner` is removed from both locales. No notification keys, no
   `TRANSACTION_*` reasons, and no migration (`resources/sql/initN_*.sql`) are added: the change is
   frontend-only.

## Risks / Trade-offs

- [The dashboard's socket replays the whole history on connect, including events it does not use]
  → Acceptable: one match, at most ~110 messages; the viewer already does the same.
- [Socket unreachable (`giveUp`) while the match is still in its window] → Hero stays "LIVE" with
  the "Watch live" button but hides score/minute (spec scenario); it falls back to the next match
  once the 2-hour window ends.
- [A live match whose simulation lags far behind kickoff could keep the hero "LIVE" at 0-0 for a
  while] → Same behaviour the viewer shows; the 2-hour cap bounds it.
- [Owner navigates away and back] → The socket reconnects and catches up instantly; completion
  state is recomputed from the `complete` message.

## Migration Plan

Frontend-only deploy; rollback is reverting the commit. No data migration.
