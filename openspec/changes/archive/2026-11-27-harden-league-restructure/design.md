# Design

## Context

See proposal.md (Why). Facts about the current code that shape the approach:

- `engine.applyRestructure` makes one plan (`PlanRestructure`) and then applies it in this
  order: dissolve, create the missing leagues, move teams, top up, archive empty leagues. Every
  error is logged and skipped.
- `common.DissolveTeam` returns `ok=false`, with no error, for a club that is user-owned, is a
  national team or is already dissolved. The plan's protected set is read once at the start, so a
  club that `AssignTeam` hands to a new user in the meantime goes unnoticed.
- `ArchiveLeagueIfEmpty` only archives empty leagues. A team that is neither dissolved nor moved
  keeps its old league active. That league then breaks `FollowsRatioRule`, so the country runs in
  legacy mode, and breaks `MatchesConfig`, so the country stays pending.
- `runSeasonRollover` records `rolloverStepRestructured` whatever happened. The progress marker
  already makes the rollover resumable (engine-replicas "Resumable season rollover"), and a
  rerun restructure already converges: the dissolutions and moves already done are taken as the
  new starting point.
- The whole rollover runs under `pg_try_advisory_lock(hashtext(country))` on a dedicated
  connection, so replicas never restructure the same country at once. The season rollover job
  fires every 5 minutes (`StartAdvanceSeasonScheduler`).
- `randomWeekMatches2` assumes an even number of teams. Keeping every group at its configured
  size matters for that reason too, not only for the shape check.

## Goals / Non-Goals

**Goals:**
- One restructure attempt either reaches the target shape or reports exactly why it did not.
- Transient failures repair themselves within the same rollover, and persistent failures never
  stall the country's season.
- The admin can see the cause without reading the logs.

**Non-Goals:**
- Changing how teams are ordered, dealt in snake order or topped up.
- Changing the doubling rule or the legacy-season behaviour.
- A manual "restructure now" action.

## Decisions

### D1. Replan instead of patching a failed dissolution

`applyRestructure` runs the dissolution phase as a loop of at most 5 rounds:

1. Read the teams in the country's active leagues and the protected set. Add the `pinned` set,
   which starts empty.
2. Run `PlanRestructure`. Pinned teams count as `Protected`, so `cutToPlaces` swaps them for the
   lowest-ranked unprotected team. This is the rule that already exists for user teams.
3. Dissolve the plan's `Dissolve` list. A team whose dissolution errors or returns `ok=false` is
   added to `pinned`, and the reason is logged.
4. If nothing was added to `pinned`, stop. Otherwise go back to step 1. The dissolved teams have
   left their leagues by then, so the next plan only sees the remaining surplus.

The moves, league creation and top-up then use the last plan.

Alternatives considered:
- *Put the leftover club in the bottom tier over its size.* This was rejected: it leaves a group
  with an odd number of teams, which breaks fixture generation, and the country stays pending.
- *Re-read the protected set inside `DissolveTeam` and fail the transaction.* Rejected because it
  still needs a replacement team, which is a plan-level decision.

`DissolveTeam` keeps its SQL. It gains a sibling, `common.GetDissolveSkipReason(dbCtx, teamId)`
(`owner`, `national_team`, `already_dissolved`, `missing`), used only for the log line. A team
that is already dissolved is not pinned, because it has left its league already.

Both `common.DissolveTeam` and `common.CreateLeague` are called through package variables in
`engine` (`dissolveTeam`, `createLeague`), as `restructureTarget` already is, so integration
tests can inject failures.

### D2. Check the result; retry through the progress marker

`restructureIfNeeded` returns `(moves []LeagueMove, diffs []common.ShapeDifference)`. After
applying, it re-reads `GetCountryPyramidShape` and computes `shape.Differences(target)` (D3).

In `runSeasonRollover`, `rolloverState` gains `RestructureAttempts int` (JSON
`restructureAttempts`):

- If there are no differences, record `rolloverStepRestructured` as today.
- If there are differences and `RestructureAttempts < 2` (fewer than 3 attempts so far), add one
  to the attempts and save the state while keeping `rolloverStepPromotionApplied`. Then return.
  The next tick, 5 minutes later, resumes at the restructure.
- Otherwise log an error listing every difference, record `rolloverStepRestructured` and carry
  on. The country stays pending, so its next rollover tries again.

The moves the attempts return all refer to each team's final league of the season
(`finishes`), so a later attempt simply replaces the earlier attempt's `state.Moves`.

Alternative considered: retry inside the same call. This was rejected because a failure that
lasts more than a moment, such as a database blip, would fail again at once. The 5-minute tick
also keeps the lock time short.

### D3. One pure shape comparison for the check, the log and the admin

`common.PyramidShape.Differences(target []TierConfig) []ShapeDifference` is pure:

```go
type ShapeDifference struct {
    Kind     string `json:"kind"`     // "extra" | "missing" | "size" | "ratio" | "invalid_structure"
    Tier     int    `json:"tier,omitempty"`
    Group    int    `json:"group,omitempty"`
    League   string `json:"league,omitempty"`
    Teams    int    `json:"teams,omitempty"`
    Expected int    `json:"expected,omitempty"`
    Message  string `json:"message,omitempty"` // invalid_structure: the ValidateWorldConfig error
}
```

- `invalid_structure` is reported when the target fails `ValidateWorldConfig`, for example an old
  saved 1/2/3 structure. In that case it is the only difference reported, because
  `restructureIfNeeded` refuses such a target.
- `extra` is an active league whose (tier, group) is not in the target.
- `missing` is a target slot with no active league.
- `size` is a group whose team count is not `TeamsPerLeague`.
- `ratio` is reported once when `!FollowsRatioRule()`.

`MatchesConfig` becomes `len(Differences(tiers)) == 0` for a valid target, so the existing
behaviour and tests stay.

`GetPendingRestructureCountries` returns `[]PendingRestructure{Country string; Differences
[]ShapeDifference}`. The admin overview keeps `pendingRestructure []string` (derived from it)
and adds `pendingRestructureDetails`.

### D4. Admin UI

`WorldStructureSection` renders, under the existing pending `Alert`, one line per country and
difference. It uses the new keys `admin.worldStructure.pendingDetail.extra`, `.missing`, `.size`,
`.ratio` and `.invalidStructure`, with interpolated tier, group, league, teams and expected
values, in `en.json` and `es.json`. No backend translation is needed because it is not a
notification.

### D5. Concurrency

There is no new job, and the retry stays inside the existing per-country advisory lock. The
existing three-concurrent-rollovers test pattern is extended to the retry path: three replicas,
with a league creation that fails once, still restructure once and advance the season by
exactly 1.

The race this change closes is the one with `AssignTeam` and other jobs running while the lock
is held. D1 covers it because the loop re-reads the protected set each round.

### Added by this change

- Migration: none. `rollover_state` is already JSON, and `restructureAttempts` defaults to 0.
- TRANSACTION_* reasons: none.
- Notification keys: none.

## Risks / Trade-offs

- [A later attempt can deal a team into a different group than the first, partial attempt did,
  because teams already moved rank by their new tier.] → Only group membership within a tier can
  change, which the spec does not fix across attempts. Nothing has been played yet, and the
  starting order is rebuilt from the last attempt's moves.
- [After 3 failures the season advances with a wrong shape, which means legacy mode and no
  playoff.] → This is the same outcome as today, but now visible in the admin and retried at the
  next rollover. This is the open question in the proposal.
- [The 5-round loop could run out with teams still pinned while unprotected CPU teams remain.]
  → Each round pins at least one more team and the protected set is finite. With 5 rounds still
  failing, the shape check (D2) catches the result.

## Migration Plan

Deploy the backend and frontend together. The new overview field is additive. ES needs no manual
step: it is pending, so at the end of season 4 its rollover restructures it with the new code.
To roll back, redeploy the previous version. Any `restructureAttempts` left in a progress
marker is ignored by the old code.

## Code map (merge into `openspec/specs/league-pyramid` at archive)

A delta spec cannot carry a Code map, so these additions to the league-pyramid Code map wait here
until archive:

- Shape check: `backend/common/pyramid.go` (`ShapeDifference`, the `ShapeDiff*` kinds,
  `PyramidShape.Differences`, `DescribeShapeDifferences`; `MatchesConfig` is now
  `len(Differences) == 0`), tested in `common/pyramid_test.go`.
- Restructure: `backend/engine/restructure.go` (the replanning loop in `applyRestructure`,
  `maxDissolveRounds`, `restructureTeams`, `teamFinish`, the `dissolveTeam`/`createLeague` hooks;
  `restructureIfNeeded` returns the differences left), `backend/common/dissolve.go`
  (`GetDissolveSkipReason`, `DissolveSkip*`).
- Retry: `backend/engine/season.go` (`rolloverState.RestructureAttempts`,
  `maxRestructureAttempts`, the restructure step of `runSeasonRollover`).
- Admin: `common.GetPendingRestructureCountries` returns `[]PendingRestructure`; the overview's
  `pendingRestructureDetails` (`handlers.AdminOverviewHandler`), rendered by
  `describeDifference` in `frontend/src/features/admin/WorldTab.tsx` with
  `admin.worldStructure.pendingDetail.*` strings.
- Tests: `engine/restructure_hardening_integration_test.go`, `engine/restructure_test.go`
  (`TestPlanRestructure_PinnedTeamsKeepEveryGroupFull`), `handlers/adminOverviewPending_test.go`,
  `frontend/src/features/admin/WorldTab.test.tsx`.
