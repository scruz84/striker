# Proposal

## Why

Spain (ES) came out of a restructure only partly reshaped. Groups were removed, but extra groups
stayed active (it now reads 1/2/3). Because that shape breaks the doubling rule, the country plays
in legacy mode with no playoff, and the admin keeps listing it as pending. The restructure records
itself as done even when something failed. Every failure is only written to the log:

- a dissolution that errors;
- a dissolution that silently does nothing, for example a CPU club that a user claimed between
  the plan and its dissolution;
- a group that cannot be created;
- a team that cannot be moved.

A team that is neither dissolved nor moved keeps its old league alive, because only empty leagues
are archived. Nothing tells the admin why the country is still pending.

## What Changes

- **No team is left behind.** A team the plan meant to dissolve that is still in a league
  afterwards becomes protected. A CPU team ranked lower is dissolved in its place, so every group
  keeps its size. The same applies to a team a move left in a league the structure no longer has.
- **The result is checked.** After a restructure the country's leagues are compared with its
  structure. If they differ, the rollover does not move on. The next tick, 5 minutes later, runs
  the restructure again, which is already safe to repeat. After 3 failed attempts the rollover
  continues so the country's season is not held up. It logs an error naming every league that
  differs, and the country stays pending for its next rollover.
- **The admin sees why.** The admin overview lists, for each pending country, the leagues that
  differ from its structure: groups that are extra, missing or the wrong size, or a shape that
  breaks the doubling rule. The World tab shows that list under the "pending restructure" notice.
- **Silent no-ops are logged.** A dissolution that changes nothing is logged with the reason, such
  as the club now having an owner.
- Concurrency stays as it is. The rollover, restructure included, already runs under a Postgres
  advisory lock per country, so two engine replicas cannot restructure at the same time. The race
  this change closes is with other work that runs during the restructure: the API assigning a
  club to a new user, or another replica's jobs.

## Capabilities

### New Capabilities
- (none)

### Modified Capabilities
- `league-pyramid`: adds three requirements. A restructure leaves no team in a league outside the
  structure. A restructure checks its result and retries before giving up. The admin sees why a
  country is pending.

## Impact

- Backend:
  - `engine/restructure.go`: the replanning loop, the result check, and the leftover-team sweep.
  - `engine/season.go`: the restructure attempt count in `rolloverState`, and not recording the
    restructure step while the result is wrong.
  - `common/pyramid.go`: a pure `PyramidShape.Differences(target)`.
  - `common/admin.go` and `handlers/adminHandlers.go`: a new `pendingRestructureDetails` overview
    field. The existing `pendingRestructure` list is kept.
  - `common/dissolve.go`: no SQL change, but the reason for a no-op is now returned so it can be
    logged.
- Frontend: `features/admin/WorldTab.tsx`, the API types, and the en/es strings.
- No schema migration, money movement or notification is added.
- ES itself repairs at its next rollover, the end of season 4. Season 4 still finishes in legacy
  mode, with no playoff.

## Out of scope

- Restructuring a country mid-season, or giving the admin a "restructure now" button.
- A group that ends up with an odd number of teams. Today that only happens when no CPU team is
  left to dissolve in place of a protected one. It stays as it is and is shown as "wrong size" in
  the admin details.
- Making the doubling rule configurable, for example allowing 1/2/3.

## Open questions

- Is 3 attempts, about 15 minutes, the right limit before the rollover moves on with a wrong
  shape? The alternative is to block the country's rollover until an admin acts. That stops its
  season, so this proposal does not do it.
