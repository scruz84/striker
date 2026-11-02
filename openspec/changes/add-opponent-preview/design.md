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
{zones: [{zone, own, opponent}]} | null}`.

### D2. Opponent data (`common/opponentpreview.go`)
- Form: the opponent's last 5 league/cup `TeamMatchSummary` rows whose replay has finished
  (same filter as the team match list).
- Likely XI: the kickoff snapshot of the most recent of those matches (`GetMatchLineupOverride` /
  `GetCupTieLineup` on that fixture), with players who left the club dropped.
- In-form players: stored ratings of those 5 matches, averaged per player, ≥ 3 appearances, top 3.

### D3. Projection (`engine.ProjectZoneDuel`)
A pure function that takes the two lineups, both squads, which side is home and (when present)
each side's mentality and expected crowd fill, builds a `MatchSimulation` at minute 1 and runs only
`InitialSimulation.Apply`, returning the nine zone pairs. The owner's side uses his saved lineup for
that fixture (override, else default / friendly default) after the same projected kickoff refill
the editor already shows (`ProjectKickoffReplacements`); the opponent's likely XI is used as-is
(its availability is not public). Without home advantage shipped, fill is ignored; without
mentality, both sides are Balanced.

### D4. Frontend
`OpponentPreviewCard` (`features/lineup/`) with form chips (`FormStrip` pattern from the
dashboard), a mini pitch for the likely XI (`SoccerPitch` read-only), the in-form list and
`ZoneComparisonRows` fed from the projection. It is placed above `MatchLineupForm` in
`LineupEditorPage` (next unlocked fixture from `useTeamMatches`) and in the scheduled view of
`MatchViewer` for participants. The query key includes the lineup's saved version, so a save
invalidates it. Keys `preview.*` in en/es.

## Risks / Trade-offs

- [The likely XI is stale or wrong] → it is labelled "XI used in <match>" with a link, never
  presented as the opponent's real lineup.
- [Projection read as a prediction] → no win probability is shown; the card says it is based on
  ratings at kick-off without fatigue.
- [Extra load on lineup pages] → one request per open editor, a few indexed queries.

## Migration Plan

No schema change. Rollback: redeploy the previous service and frontend.
