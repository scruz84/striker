# Proposal

## Why

The lineup editor has two standing lineups, "League/Cup" and "Friendly", but only one set of
scheduled substitutions: `default_lineup_substitutions`, edited from the League/Cup tab.
`common.GetDefaultFriendlyLineup` returns those same rows, the Friendly tab hides the substitutions
section, `PUT /v1/user/team/lineup/friendly` ignores substitutions, and the hint text tells the
manager "Scheduled substitutions are shared with the league/cup default". A friendly XI is usually a
different, experimental XI, so substitutions written against the competitive XI rarely make sense
there. The owner wants the two standing lineups to have independent substitutions.

There is also an inconsistency today: a friendly with no per-match override that falls back to the
friendly default plays with **no** scheduled substitutions (`engine.simulateFriendly` reads only
`friendly_lineup_substitutions`), while the per-friendly editor pre-fills its drafts with the
league/cup substitutions. Giving the friendly default its own substitutions lets the fallback play
the ones the manager actually sees.

## What Changes

- **Own substitutions for the friendly default.** New table `default_friendly_lineup_substitutions`
  (team-wide, up to `MaxScheduledSubstitutions` = 3, same shape as `default_lineup_substitutions`).
  Nothing is copied from the league/cup default: every team starts with none.
- **API.** `GET /v1/user/team/lineup/friendly` returns the friendly default's own substitutions;
  `PUT /v1/user/team/lineup/friendly` accepts `substitutions`, validated like every other lineup
  save (friendly availability rules: a suspension does not bar a player from coming on), and
  replaces them. A missing or empty list clears them.
- **League/cup unaffected.** `PUT /v1/user/team/lineup` keeps writing only
  `default_lineup_substitutions`; league matches and cup ties without an override keep falling back
  to them. Saving one tab never changes the other tab's substitutions.
- **Friendly fallback plays them.** A friendly without a per-match override that falls back to the
  friendly default plays that default's substitutions. A friendly that falls back to the computed
  best XI (no friendly default saved) still plays none. A per-match friendly override still plays
  exactly its own substitutions. The per-friendly editor pre-fills its drafts from the friendly
  default's substitutions instead of the league/cup ones.
- **Frontend.** The Friendly tab shows the substitutions planner and saves it; the hint text no
  longer says substitutions are shared (en/es).
- **Club reset.** Resetting a released club and re-initialising the game also clear the new table.

## Capabilities

### New Capabilities

- `team-lineups`: a team's standing lineups (league/cup default and friendly default) and their
  scheduled substitutions, and which lineup and substitutions each kind of match falls back to when
  it has no per-match override.

### Modified Capabilities

None. `match-simulation` ("Scheduled substitutions") governs how a scheduled substitution fires and
is unchanged; `cpu-team-management` keeps choosing CPU substitutions at kickoff.

## Impact

- Migration `backend/resources/sql/init94_default_friendly_lineup_substitutions.sql`. No new
  `TRANSACTION_*` reason, no new notification key.
- Backend: `common/lineupsubstitutions.go` (new getter/setter), `common/database.go`
  (`GetDefaultFriendlyLineup`), `common/friendlies.go` (`GetLineupsForFriendly`),
  `engine/friendlies.go` (`simulateFriendly`), `handlers/lineupHandlers.go`
  (`SetFriendlyDefaultLineupHandler`), `common/accounts.go` (club reset),
  `handlers/initializer.go` (`teamPlayerDependentTables`).
- Frontend: `features/lineup/LineupEditorPage.tsx`, `api/hooks.ts`
  (`useSetFriendlyDefaultLineup`), `features/lineup/MatchLineupForm.tsx` (comment), en/es locales.

## Out of scope

- Per-match substitutions (league match, cup tie, friendly overrides): already independent, unchanged.
- CPU kickoff planning and national-team friendlies (they set their own substitutions).
- Cleaning stale player ids out of standing substitutions after a transfer: the simulator already
  skips or replaces them, same as today for the league/cup default.
- The friendly default's lineup validation keeping league availability rules (suspensions) for the
  XI itself; only the new substitutions use friendly rules.

## Owner decisions

1. The migration does **not** copy each team's league/cup substitutions into the new friendly
   table: every team starts with no friendly substitutions.
2. A friendly that falls back to the friendly default **plays** that default's substitutions.
   Friendlies follow only the friendly lineup (the friendly's own override, else the friendly
   default) and never read the league/cup lineup or its substitutions.
