# Design

## Context

- The global structure is the singleton `world_config.tiers` (jsonb), read by
  `common.GetWorldConfig` and checked by `ValidateWorldConfig` and `ValidateWorldConfigFitsTeams`
  in `AdminSetWorldConfigHandler`. `engine.restructureTarget` returns it for every country.
- The restructure runs as step 4 (`rolloverStepRestructured`) of the resumable rollover
  (`engine/season.go`), under the country's rollover lock, after the season's moves.
  `PlanRestructure` is pure. With more teams than places it currently piles the extra teams into
  the bottom tier.
- `common.ResetReleasedTeam` already does most of what a dissolution needs, in one transaction:
  it settles loans, clears offers, listings, challenges, trainer and lineups, and frees the senior
  squad. It then also generates a new squad and identity, which a dissolution does not.
- National teams show that the codebase already has teams outside the club pyramid
  (`is_national_team`, `league_id = -1`), and club-listing queries already filter them.

## Goals / Non-Goals

**Goals:**
- One function, `common.EffectiveStructure(dbCtx, country)`, that every reader uses: the country's
  own structure, else the global one.
- A dissolution that is atomic per team and idempotent, so an interrupted rollover can re-run the
  restructure step.
- No dissolved team ever reappears in play, assignment, AI activity or the market.

**Non-Goals:**
- Deleting team or player rows. Freeing disk space from past matches is the instant archive's job.

## Decisions

### 1. Migration `init119_country_structures.sql`
- `countries.structure jsonb NULL`: the country's own tiers in the `world_config.tiers` format.
- `teams.dissolved_season int NULL`, plus a partial index `teams (country) WHERE dissolved_season
  IS NULL AND NOT is_national_team` for the club listings.
- Re-runnable (`ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`). No `TRANSACTION_*`
  reason. No notification key: dissolved teams are CPU teams with no owner.

*Alternative:* a separate `country_structures` table. Rejected: it holds one value per country,
the same as `countries.league_kickoff_minute`.

### 2. Effective structure and validation
- `restructureTarget` becomes `func(dbCtx, country)` and returns `EffectiveStructure`.
  `GetPendingRestructureCountries` and the World tab's "missing division" check compare against it
  too.
- `ValidateWorldConfigFitsTeams` becomes `PlanStructureChange(tiers, countries)`. It counts each
  affected country's teams and user-owned teams (`owner_id IS NOT NULL`, not dissolved), rejects
  the save if places < user teams, and returns `{country: teamsToDissolve}` for the rest. A global
  save applies to the countries without their own structure. A country save applies to that
  country only.
- `PUT /v1/admin/world-config` and the new `PUT /v1/admin/countries/:code/structure` take
  `confirmDissolve` (bool). When the plan dissolves teams and it is false, they answer 409 with
  `{ "dissolve": { "es": 160, ... } }` and save nothing. `DELETE /v1/admin/countries/:code/structure`
  goes through the same check against the global structure. `GET` returns the override or `null`.

### 3. Planning the dissolution
`PlanRestructure` gains a `userOwned` flag on `RankedTeam` and a `Dissolve []int64` result. After
ordering, it keeps the first `places` teams. For every user team beyond that cut, from the best
ranked down, it swaps the user team with the lowest-ranked CPU team inside the cut, so the user
team takes that CPU team's slot (always in the bottom tier). The teams left beyond the cut are
dissolved. The function stays pure and covered by table tests.

### 4. Applying it: `common.DissolveTeam(dbCtx, teamId, season)`
In one transaction, with the team locked `FOR UPDATE` and skipped if `dissolved_season` is already
set:
1. Set `dissolved_season = season`, `league_id = -1`, `ai_level = 0`, and remove its
   `classification` row for the new season.
2. Run the clearing half of `ResetReleasedTeam`, extracted into a shared helper: offers, listings,
   challenges, lineups, trainer back to the market, and unplayed friendlies deleted with their lineups.
3. Players loaned in go back to their clubs now. Players loaned out stay on loan. When the loan
   ends, the existing return path finds a dissolved parent and releases the player as a free agent
   (`free_agent_seasons = 0`).
4. Senior players become free agents through the existing release path. Academy prospects retire.

The restructure step dissolves before moving the placed teams, so a re-run after a crash sees the
dissolved teams already out of the active leagues and plans the same result for the rest.

### 5. Keeping dissolved teams out
Every query that lists clubs for play, assignment or the market gains `AND t.dissolved_season IS
NULL`, next to the existing `is_national_team` filters: `GetFreeTeamStmtSql`, `GetAllTeamIds`, AI
activity and friendly candidates, transfer and loan buyers, sponsor and wage batches, national
call-ups (only through players, which are already free agents), and the CPU market. The audit
covers the ~46 `FROM teams` statements in `common`, `engine` and `handlers`. A test seeds a
dissolved team and checks it is absent from each listing.

### 6. Frontend
- World tab: each active country row gains "Own structure" with an editor that reuses
  `WorldStructureSection`'s tier rows and `pyramidRules.ts`, plus a "Use global structure" action.
- A shared confirmation dialog shows the 409's per-country counts. Confirming resends with
  `confirmDissolve: true`.
- Team page: a "Dissolved in season N" chip when `dissolvedSeason` is set, and no squad, market or
  challenge actions.
- Strings: `admin.countryStructure.*`, `admin.worldStructure.confirmDissolve*`, `team.dissolved` in
  `en.json` and `es.json`.

## Risks / Trade-offs

- [A large shrink floods the free-agent market (e.g. 160 teams × 18 players per country)] →
  free agents already retire after 2 unsigned seasons (`FreeAgencyRetirementSeasons`). Check with
  `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` on a shrink from 5 to 4 tiers, and
  retire the oldest released players if wages collapse.
- [A missed club listing lets a dissolved team act] → the audit task plus the absence test, and
  `league_id = -1` already keeps it out of every per-league query.
- [The admin shrinks by mistake] → the 409 confirmation names the numbers, and nothing happens
  until each country's rollover. Until then, saving the old structure again cancels it.
- [Removing a tier in the middle of a season] → it never applies mid-season. The rollover order
  (moves, then restructure) means this season's relegations into the removed tier dissolve those
  teams, the same as finishing in it.

## Migration Plan

`init119` is applied on startup. Nothing changes until an admin saves a smaller structure, which
takes effect at each country's next rollover. Rollback: save a larger structure again. Dissolved
teams are not revived, and the larger structure tops up with new CPU teams.
