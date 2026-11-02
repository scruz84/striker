# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- `default_lineup_substitutions (team_id, slot 1-3, player_out_id, player_in_id, minute)` is the
  only standing substitutions table. `common.GetDefaultLineup` and `common.GetDefaultFriendlyLineup`
  both read it through `GetDefaultLineupSubstitutions`; only `SetLineupHandler` writes it
  (`SetDefaultLineupSubstitutions`).
- The friendly default XI lives in `default_friendly_lineup_positions` + `teams.friendly_formation`
  (`init60_default_friendly_lineup.sql`), saved by `SetFriendlyDefaultLineupHandler` →
  `SetDefaultFriendlyLineup`. The handler already binds `SetLineupRequest`, which has
  `Substitutions`, but ignores it.
- `common.GetLineupsForFriendly` resolves override → friendly default → `BestLineupForTeam` (empty
  substitutions). `GetFriendlyLineupHandler` mirrors that order for the per-friendly editor, so its
  fallback currently carries the league/cup substitutions into the editor's drafts.
- `engine.simulateFriendly` ignores `lineups[teamId].Substitutions` and reads
  `GetFriendlyLineupSubstitutions` (override rows only), then passes them through `planAiKickoffs`
  (which replaces them for CPU teams) and `persistFriendlyLineupSnapshot` (which freezes XI + subs
  as the friendly's override).
- League matches and cup ties fall back through `GetSubstitutionsForMatch` /
  `GetSubstitutionsForCupTie` to `GetDefaultLineupSubstitutions`; that stays as is.

## Goals / Non-Goals

**Goals:**
- The friendly default owns its substitutions end to end: storage, API, editor, simulation fallback.
- The substitutions a friendly plays always come from the same source as the XI it plays.

**Non-Goals:**
- No change to `default_lineup_substitutions`, to per-match override tables, or to how the
  simulator fires a scheduled substitution.

## Decisions

### D1. New table, same shape
Migration `backend/resources/sql/init94_default_friendly_lineup_substitutions.sql`:

```sql
CREATE TABLE IF NOT EXISTS default_friendly_lineup_substitutions (
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    player_out_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    minute        smallint NOT NULL CHECK (minute BETWEEN 1 AND 90),
    PRIMARY KEY (team_id, slot)
);
```

No data copy (proposal, owner decision 1). Re-runnable by `IF NOT EXISTS`.
*Alternative:* add a `kind` column to `default_lineup_substitutions` (0 = league/cup, 1 = friendly)
and widen the primary key. Rejected: it rewrites the primary key of a live table, every existing
query needs a new predicate, and the codebase already models each lineup kind with its own table
(`friendly_lineup_substitutions`, `cup_tie_lineup_substitutions`).

### D2. Store functions
In `common/lineupsubstitutions.go`, next to the league/cup pair:
`GetDefaultFriendlyLineupSubstitutionsStmtSql` / `DeleteDefaultFriendlyLineupSubstitutionsStmtSql` /
`InsertDefaultFriendlyLineupSubstitutionStmtSql`, `GetDefaultFriendlyLineupSubstitutions(dbCtx,
teamId)` and `SetDefaultFriendlyLineupSubstitutions(dbCtx, teamId, subs)`. The setter copies
`SetDefaultLineupSubstitutions` (delete-then-insert in one transaction, slots by save order) with
the non-blocking guard `pg_try_advisory_xact_lock(teamId, 2)`: key `(teamId, 1)` is the friendly
XI save and `(teamId)` the league/cup saves, so no friendly save contends with or falsely no-ops a
different save. `GetDefaultFriendlyLineup` switches to `GetDefaultFriendlyLineupSubstitutions`;
`GetDefaultLineup` is untouched.

### D3. Saving through the existing endpoint
`SetFriendlyDefaultLineupHandler` validates `req.Substitutions` with `validateSubstitutions`
against `common.GetUnavailablePlayerIdsForFriendly` (only injuries bar a friendly; the XI
validation keeps its current map), before any write. After `SetDefaultFriendlyLineup` it calls
`SetDefaultFriendlyLineupSubstitutions(team.Id, req.Substitutions)`; a nil slice clears, like the
league/cup handler. Same order of writes and error handling as `SetLineupHandler`. No new route.
*Alternative:* one transaction for formation + XI + substitutions. Rejected for this change:
`SetLineupHandler` uses the same three separate writes and both should change together if at all.

### D4. One source for a friendly's substitutions
`engine.simulateFriendly` uses `(*lineups)[teamId].Substitutions` from `GetLineupsForFriendly`
instead of the two `GetFriendlyLineupSubstitutions` reads. `GetLineupsForFriendly` already returns:
the override with its own substitutions (`GetFriendlyLineup`), the friendly default with (after D2)
the friendly set, or the best XI with an empty set. So the XI and its substitutions can never come
from different sources, and a friendly never reads the league/cup lineup or substitutions
(proposal, owner decision 2). `planAiKickoffs` and `persistFriendlyLineupSnapshot` are unchanged, so the
snapshot freezes the substitutions that were played and a retried simulation reads them back as
the override. National-team friendlies keep their call-up override. `GetFriendlyLineupHandler`
needs no code change: its fallback now carries the friendly set automatically.
*Alternative:* a `GetSubstitutionsForFriendly` helper like the league/cup ones. Rejected: it would
re-run the same override/default checks `GetLineupsForFriendly` just did.

### D5. Club reset and game re-initialisation
`common/accounts.go`: `DeleteTeamDefaultFriendlySubstitutionsStmtSql = "DELETE FROM
default_friendly_lineup_substitutions WHERE team_id = $1"`, run by the reset next to
`DeleteTeamDefaultSubstitutionsStmtSql`. `handlers/initializer.go`: add
`"default_friendly_lineup_substitutions"` to `teamPlayerDependentTables` before `"players"` (it
references both `players` and `teams`).

### D6. Frontend
- `LineupEditorPage.tsx` `FriendlyDefaultLineupForm`: pass `substitutionsEnabled` and `isFriendly`
  (so the planner's availability matches D3: a suspended bench player is selectable), and send
  `substitutions` in `onSave`. Update both component comments.
- `api/hooks.ts` `useSetFriendlyDefaultLineup`: mutation takes and PUTs `substitutions`.
- `MatchLineupForm.tsx`: fix the `substitutionsEnabled` comment (both standing editors use the
  flag; friendlies fall back to the friendly default's own set).
- Locales: `lineup.friendlyHint` drops the "shared" sentence: en "This is the default lineup and
  substitutions used for friendly matches that don't have their own lineup set. They are
  independent of the league/cup default."; es "Esta es la alineación y las sustituciones por
  defecto para los amistosos que no tengan una alineación propia. Son independientes de las de
  liga/copa." No new keys.

No new `TRANSACTION_*` reason and no notification key. No engine job is added, so no new
replica-safety work: the snapshot path is unchanged.

## Risks / Trade-offs

- [Behaviour change: friendlies on the friendly default start playing scheduled substitutions] →
  The set starts empty (D1), so nothing changes until a manager saves some in the Friendly tab.
- [Old frontend bundle cached in a browser PUTs the friendly default without `substitutions`] →
  It clears the friendly set, which that bundle could not edit anyway; the league/cup set is safe.
- [Stale player ids after a transfer or loan] → Same as the league/cup set today: the simulator
  skips a substitution whose outgoing player is not on the pitch and replaces an ineligible
  incoming player; `ON DELETE CASCADE` covers deleted players.

## Migration Plan

Deploy backend and frontend together; the migration only creates an empty table. Rollback: revert
the code; the table can stay (nothing else reads it) or be dropped by hand.
