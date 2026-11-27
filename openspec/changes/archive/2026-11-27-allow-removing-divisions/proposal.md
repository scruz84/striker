# Proposal

## Why

The number of teams drives almost all the data a season stores: with groups of 10, every team adds
9 home matches a season, and each match stores ~200 KB. Today the league pyramid can only grow.
There is one global structure, and saving one with fewer places than a country's teams is rejected,
because a restructure never removes a team. An admin who wants a smaller world, or a small country
with fewer divisions, has no way to get there. Removing the bottom tier of the default structure
(16 of 31 groups) halves a country's teams and its per-season data.

## What Changes

- **Per-country structure.** A country can have its own full structure (tiers, groups, teams per
  group) that overrides the global one. It follows the same pyramid rules. A country without one
  keeps following the global structure, and removing the override puts it back on the global one.
  The admin edits it from the country's row in the admin World tab.
- **Shrinking is allowed.** A structure, global or per country, may have fewer places than a
  country's teams. At that country's next rollover, the surplus teams are **dissolved**: they are
  the worst-ranked CPU teams after the season's moves, and never a user's team.
- **Dissolving a CPU team**: its players become free agents, its trainer goes back to the market,
  its pending offers, listings, loans and unplayed friendlies are cancelled, and it stops playing
  for good. The team row, its past matches and its history stay, so old results and history pages
  still work, and its page reads "Dissolved in season N".
- **Saving asks for confirmation.** Saving a structure that would dissolve teams answers with the
  teams each country would lose, and only goes through once the admin confirms. A structure with
  fewer places than a country's **user-owned** teams is still rejected.
- **BREAKING (rule change)**: "Teams are never deleted" no longer holds. CPU teams can be dissolved.

## Capabilities

### New Capabilities
- (none)

### Modified Capabilities
- `league-pyramid`: per-country structures; "Structure fits existing teams" becomes "user teams must
  fit, CPU surplus needs confirmation"; restructure detection uses the country's effective
  structure; the restructure dissolves surplus CPU teams instead of always keeping every team. New
  requirements describe a dissolved team.

## Impact

- Schema: a migration adding `countries.structure jsonb NULL` (the override) and
  `teams.dissolved_season int NULL`.
- Backend: `common/worldconfig.go` (effective structure, validation per country), `engine/restructure.go`
  (dissolution in the plan and its application, idempotent inside the resumable rollover),
  a new `common.DissolveTeam`, and the club-listing queries that must skip dissolved teams (team
  assignment, AI activity, friendlies, market, wages, sponsors, national call-ups). New admin
  endpoints `GET/PUT/DELETE /v1/admin/countries/:code/structure`. `PUT /v1/admin/world-config`
  gains the confirmation step.
- Frontend: World tab (per-country structure editor, the confirmation dialog listing teams to
  dissolve, "missing division" checks against the effective structure), the team page label, and
  en/es strings.
- Economy: dissolving a tier puts its players on the free-agent market at once, so this needs the
  `ECON_SIM` check.

## Out of scope

- Deleting team rows or their past data (dissolution keeps them; the instant archive handles old
  per-minute data).
- Merging countries or moving teams between countries.
- Undoing a dissolution (a later, larger structure tops up with new CPU teams as today).

## Open questions for the owner

- None left: dissolving the surplus CPU teams and full per-country overrides were chosen by the owner.
