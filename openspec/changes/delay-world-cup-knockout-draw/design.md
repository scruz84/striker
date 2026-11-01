# Design

## Context

See proposal.md - Why. `advanceWorldCupLocked` (engine/worldcup.go) starts the knockout stage
when `common.IsWorldCupGroupStageComplete` reports every linked group friendly as played. Cup
rounds already wait for their broadcasts: `GetUnresolvedCupTieCountStmtSql` treats a tie whose
`match_date` is after `LiveBroadcastCutoff()` as unresolved.

## Goals / Non-Goals

**Goals:**
- The knockout cup, its first-round call-ups and notifications never appear before every group
  broadcast has ended.

**Non-Goals:**
- Changing knockout scheduling (`cupRoundGap`) or the draw itself.

## Decisions

### Extend the completeness query instead of adding a second check

`getWorldCupGroupMatchCountsStmtSql` additionally returns `MAX(fm.match_date)`.
`IsWorldCupGroupStageComplete` becomes `total > 0 && total == played &&
!latestKickoff.After(LiveBroadcastCutoff())`. A single query keeps the check atomic with the
existing advisory lock in `advanceWorldCup`. Any caller automatically gets the broadcast-safe
meaning. Today the only caller is the engine; the frontend reads `worldCup.stage`.

*Alternative:* a separate "has every broadcast finished" query in `advanceWorldCupLocked`. It
adds a second round trip, and it leaves `IsWorldCupGroupStageComplete` with a name that no longer
matches what the World Cup display needs.

### Same cutoff as cup rounds

Use `LiveBroadcastCutoff()` (2 h), not the frontend's `mightStillBeLive` or the live hub's
"complete" message. It is the boundary every other engine step already uses (cup round resolution,
season advance, head-to-head) and it does not depend on Redis being reachable.

No migration, notification key or `TRANSACTION_*` reason is added.

## Risks / Trade-offs

- [Knockout starts up to ~2 h later than today] → Negligible on a multi-week tournament. The first
  round still kicks off 24 h after creation.
- [A stuck, never-simulated group match] → Unchanged. `FinishStuckSimulationsJob` already covers
  it, and the new condition only adds a time bound that always eventually passes.

## Migration Plan

Deploy normally. A World Cup that is mid-broadcast at deploy time simply waits until its last
broadcast ends. Rollback is a plain revert.
