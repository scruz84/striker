# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- Public data already stored: results and fixtures (`common/teammatches.go` `TeamMatchSummary`),
  standings, the kickoff snapshot of every played match in `lineup` / `cup_tie_lineup` /
  `friendly_lineup` (written by `persistResolvedLineupSnapshot` and friends) and stored match
  ratings (match-ratings "Ratings stored at simulation time").
- A match is marked played up to 15 minutes before its kickoff and must not be revealed before its
  replay finishes; `common/headtohead.go` and the team match list already exclude unfinished
  replays, and that is the filter this preview must reuse.
- Zone ratings come from `engine/initial_simulation.go` (`calculateQua`, `blendLine`,
  `ZoneSpillover`) on a `MatchSimulation`; the viewer's zone duel renders shares from instants with
  `ZoneComparisonRows` / `zoneDuel.ts`.
- Handlers already import `engine` for read-only helpers (e.g. `accountHandlers.go`).
- Lineup lock: 20 minutes before kickoff (match-scheduling "Lineup lock before kickoff").

## Goals / Non-Goals

**Goals:**
- The projection uses exactly the simulation's rating code, so it can never drift from what the
  engine does (including home advantage and mentality once they exist).
- Strict privacy: the opponent side is built only from finished matches.

**Non-Goals:**
- Caching; the preview is computed on request (one squad read per side, cheap).

## Decisions

### D1. Endpoint
`GET /v1/teams/preview/:kind/:id` (`kind` ∈ `league`, `cup`, `friendly`) in a new
`handlers/previewHandlers.go`. It resolves the fixture, refuses (403) when the caller's club is not
in it, and returns 409 once the lock window has started. Response:
`{opponent: {id, name, position, form: TeamMatchSummary[]}, likely: {formation, positions,
matchId} | null, inForm: [{playerId, name, position, averageRating, appearances}], projection:
{own, opponent, ownGoalkeeper, opponentGoalkeeper} | null}`, where `own` / `opponent` are the
`TeamQualification` of each side at minute 1 (the shape the viewer's zone duel already renders, so
`ZoneComparisonRows` is reused as is).

### D2. Opponent data (`common/opponentpreview.go`)
- Form: the opponent's last 5 league/cup `TeamMatchSummary` rows whose replay has finished
  (same filter as the team match list).
- Likely XI: the kickoff snapshot of the most recent of those matches (`GetMatchLineupOverride` /
  `GetCupTieLineup` on that fixture), with players who left the club dropped.
- In-form players: stored ratings of those 5 matches, averaged per player, ≥ 3 appearances, top 3.

### D3. Projection (`engine.ProjectZoneDuel`)
A pure function that takes the two lineups, both squads, which side is home and (when present)
each side's mentality and expected crowd fill, builds a `MatchSimulation` at minute 1 and runs only
`InitialSimulation.Apply`, returning each side's `TeamQualification` and goalkeeper rating (the nine zone pairs are derived from them by `ZoneComparisonRows`). The owner's side uses his saved lineup for
that fixture (override, else default / friendly default) after the same projected kickoff refill
the editor already shows (`ProjectKickoffReplacements`); the opponent's likely XI is used as-is
(its availability is not public). Home advantage is included: the handler passes the multiplier from the
expected crowd (`engine.ExpectedHomeAdvantage`: modelled demand in average weather, no random swing;
1 for a World Cup fixture) and which side is at home; mentality, when it ships, will be passed the same way (until then both sides are Balanced).

### D4. Frontend
`OpponentPreviewCard` (`features/lineup/`) with form chips (`FormStrip` pattern from the
dashboard), the likely XI as lines from attack to goalkeeper (a compact list rather than a second
pitch, so the card stays short on phones), the in-form list and
`ZoneComparisonRows` fed from the projection. It is placed above `MatchLineupForm` in
`LineupEditorPage` (next unlocked fixture from `useTeamMatches`) and in the scheduled view of
`MatchViewer` for participants. Every lineup save hook invalidates the `opponentPreview` query prefix, so a save
refreshes the projection. Keys `preview.*` in en/es.

## Risks / Trade-offs

- [The likely XI is stale or wrong] → it is labelled "XI used in <match>" with a link, never
  presented as the opponent's real lineup.
- [Projection read as a prediction] → no win probability is shown; the card says it is based on
  ratings at kick-off without fatigue.
- [Extra load on lineup pages] → one request per open editor, a few indexed queries.

## Migration Plan

No schema change. Rollback: redeploy the previous service and frontend.

## Code map (for the new `opponent-preview` spec)

- Public data: `backend/common/opponentpreview.go` (`GetOpponentPreview`, `recentFinishedCompetitive`,
  `GetOpponentLikelyLineup`, `aggregateInForm`) reading `GetTeamMatches`, `GetMatchLineupOverride` /
  `GetCupTieLineup` (kickoff snapshots), `GetMatchPlayerRatings` and `LiveBroadcastCutoff` (the filter that
  keeps a replaying match out).
- Projection: `backend/engine/zoneduel.go` (`ProjectZoneDuel`, `ZoneDuelProjection`, `ExpectedHomeAdvantage`) running
  `StaminaSimulation` and `InitialSimulation` at minute 1; the owner's refilled XI comes from
  `common.ProjectKickoffReplacements` + `common.ApplyProjectedReplacements` (`common/kickoffrefill.go`).
- Endpoint: `backend/handlers/previewHandlers.go` (`OpponentPreviewHandler`, 403 / 409 rules), routed as
  `GET /v1/teams/preview/:kind/:id` in `backend/service/fmservice.go`.
- Frontend: `frontend/src/features/lineup/OpponentPreviewCard.tsx` and `opponentPreviewLines.ts` (likely XI
  lines), `useOpponentPreview` / `queryKeys.opponentPreviews` in `api/hooks.ts` (invalidated by every lineup
  save), the zone duel drawn by `features/match/ZoneComparisonRows.tsx`.
- Placement: `features/lineup/LineupEditorPage.tsx` (next fixture whose lock has not started,
  `hasLineupLockStarted` in `api/useLiveMatch.ts`) and the `previewSection` slot of
  `features/match/MatchViewer.tsx`, filled by the three route adapters.
