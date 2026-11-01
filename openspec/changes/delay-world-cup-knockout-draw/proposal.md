# Proposal

## Why

The World Cup knockout bracket is created as soon as every group match is *simulated*
(`IsWorldCupGroupStageComplete` only counts `played`). A match is simulated up to 15 minutes
before kickoff and its live broadcast runs ~105 minutes. So the bracket, its call-up
notifications and the "Knockout stage" section can appear while the last group matches are still
being broadcast, up to ~2 hours early. That reveals which countries qualified before the results
are visible. This breaks the project rule that nothing outside a match page may reveal a result
before its live replay ends. It was found while doing `improve-national-teams-page`, which made
the group tables themselves broadcast-safe.

## What Changes

- The group stage only counts as complete when every group match is played **and** the latest
  group-match kickoff is at least `LiveBroadcastWindow` (2 h) in the past, which is the same cutoff
  cup rounds already use (`GetUnresolvedCupTieCountStmtSql`).
- Nothing else changes:
  - The cup job (every minute) creates the knockout cup on its first tick after that point.
  - Its first round still kicks off 24 h after creation (`cupRoundGap`).
  - Call-ups for the first knockout round are therefore saved, and their notifications sent,
    only after the last group broadcast has ended.

**Out of scope**
- Later knockout rounds: they already wait for the broadcast (`IsCupRoundResolved` uses the
  same cutoff).
- The stored group totals (`world_cup_group_teams`), which keep being written at simulation time
  and are read only by the draw.

**Open questions for the owner**
- None. The delay is at most ~2 hours on a tournament that lasts weeks.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `world-cup`: the Knockout stage requirement gains the "after the last group broadcast"
  condition.

## Impact

- Backend: `common/worldcup.go` (`getWorldCupGroupMatchCountsStmtSql`,
  `IsWorldCupGroupStageComplete`), and an engine integration test around `advanceWorldCup`.
- No API, frontend, schema, notification or money change.
