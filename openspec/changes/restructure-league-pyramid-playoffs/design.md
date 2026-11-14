# Design

## Context

See proposal.md (Why) for the motivation. The current state that shapes the approach:

- **Moves today.** `engine.ComputePromotionRelegation` (`engine/promotion.go`) swaps the bottom
  `RelegationSlotsPerGroup` of each upper league with the best `PromotionSlotsPerGroup`
  candidates of the tier below, one for one. Both counts come from
  `common.GetPromotionRelegationConfig` (`season.promotion_slots_per_group` /
  `relegation_slots_per_group`, defaults 1/2).
- **Zones today.** The league page draws its zones from the same per-group config
  (`common.leaguePromotionRelegationSlots`, `DivisionDetails.PromotionSlots/RelegationSlots`).
  That is why all 4 tier-2 leaders look promoted.
- **Rollover.** `advanceCountrySeasonIfComplete` (`engine/season.go`) runs under
  `pg_try_advisory_lock(hashtext(country))`. Its gates are that all of the country's league
  matches are played (`GetSeasonMatchCounts`) and that all broadcasts are done
  (`CountSeasonMatchesStillBroadcasting`). Then it runs aging → loans →
  `applyPromotionRelegation` → domestic cup → classification reset → seeds → season bump →
  fixtures.
- **Cup ties** (`cups` / `cup_ties`) are single matches that go to penalties on a draw. They
  already have per-tie lineups, AI kickoff planning, a live broadcast, a match page
  (`/cup/ties/:id`), gate and sponsorship, post-match effects, and a per-tie simulation lock.
  `AdvanceCups` walks `GetActiveCups` and builds the next round whenever a round is resolved.
- **Structure.** `common.TierConfig` is saved in `world_config.tiers` (jsonb) and is read only
  when leagues are created (`handlers.createCountryLeagues`). Team creation (`createTeam`,
  `createLeague`, `createClassification`) lives in `handlers/initializer.go`, which the engine
  cannot import.
- **History.** `season_history` keys rows by (team, season) with `league_id`, `league_name`,
  `promoted` and `relegated`. Past tables and matches are read by `league_id` + `season`.

## Goals / Non-Goals

**Goals:**
- A single source for zones. The engine's moves, the league page colours and the CPU season
  objective all derive from one pure function of (tier, number of tiers, group size).
- Reuse the cup-tie stack for the playoff instead of building a second match type.
- A restructure that is idempotent under replicas and never deletes a team or a league row.

**Non-Goals:**
- Making the 2× ratio configurable. `ValidateWorldConfig` hard-codes it.
- Changing how the domestic, regional or world cups work, beyond excluding playoffs from them.

## Decisions

### Zones are derived, not configured

New `common/pyramid.go`:

- `PyramidZones(tier, tierCount, groupSize) Zones` returns, for one group:
  - `DirectUp = 1` and `PlayoffUp = 1` when `tier > 1`, otherwise 0.
  - `PlayoffDown = 2` and `DirectDown = 2` when `tier < tierCount`, otherwise 0.
- For a country whose active leagues break the ratio rule (the season in progress at deploy),
`PyramidZones` returns `Legacy = true`: `DirectDown = 2` above the bottom tier and
`Candidate = 1` below tier 1. The page draws a candidate in the playoff colour with the legend
"Promotion candidate". The engine uses `computeLegacyMoves`, which is today's one-for-one swap
with 1 candidate and 2 relegations per group.

A pure helper `ZoneOf(position, zones, groupSize)` returns `direct_up`, `playoff_up`,
  `playoff_down`, `direct_down` or `""`.

Constants: `PlayoffUpPerGroup = 1`, `DirectUpPerGroup = 1`, `PlayoffDownPerGroup = 2`,
`DirectDownPerGroup = 2`, `PyramidGroupRatio = 2`, `MinPyramidTeamsPerLeague = 8`.

Because the ratio is fixed at 2, each pair of tiers balances exactly. Upper tier with G groups:
2G direct down and 2G playoff down. Lower tier with 2G groups: 2G direct up and 2G playoff up.

`tierCount` is read from the country's **active** leagues (`MAX(tier) WHERE archived_season IS
NULL`), not from the saved config. Zones therefore describe the pyramid the season is actually
played in, even while a new config is pending.

Removed: `common/promotionconfig.go`, `GetPromotionRelegationConfig` and its admin endpoint and
`RulesTab` form. The `season.*_slots_per_group` columns are no longer read but are left in place,
so a rollback to the previous binary still works. A later change can drop them.

*Alternative considered:* keep the admin slots and add "playoff slots". Rejected, because it
lets an admin configure a pyramid where the table is wrong again, which is the very bug.

### Validation of the structure

`ValidateWorldConfig` adds these checks to the existing bounds:
- Tiers are numbered 1..T with no gaps.
- `Groups(1) == 1` and `Groups(t) == 2 × Groups(t−1)`.
- Every tier has the same `TeamsPerLeague`, which is even and ≥ 8.

`AdminSetWorldConfigHandler` also rejects a structure whose places (Σ groups × teams) are fewer
than `COUNT(teams)` of any active country, using a new `CountTeamsPerActiveCountryStmtSql`. The
messages are the ones the spec scenarios show. They are returned in English and the admin UI
shows them as-is, following the existing world-config errors.

`DefaultWorldTiers` becomes I..V with 1/2/4/8/16 groups × `TeamsPerLeague` (10).
`MaxGroupsPerTier = 32` already allows a sixth tier.

### Playoff as a cup of kind `playoff`

- **New column** `cups.kind varchar(16) NOT NULL DEFAULT 'cup'`, with
  `CHECK (kind IN ('cup','playoff'))`. There is one playoff cup per (country, season), enforced
  by the partial unique index `cups_playoff_once ON cups (country, season) WHERE kind = 'playoff'`.
- **Every tier pair's ties live in that one cup**, as round 1. The upper-tier team is always
  `home_team_id`. Rollover needs nothing more, because team leagues do not change before rollover:
  the home side's league is the upper league and the away side's league is the lower league.
- **Kickoff** (owner decision: the day after the league ends, at the upper tier's league time) is
  `playoffKickoff(country, upperTier, lastMatchdayKickoff, now)`. It takes the last league
  matchday's `match_date` in the country's time zone, adds one calendar day, and sets the local
  time to the country's league kickoff + `LeagueKickoffTierStagger × (upperTier − 1)`. Per-league
  overrides are ignored for the playoff. When the result is earlier than
  `now + PlayoffMinimumLead` (1 h), it moves one more day. This is pure and tested across DST
  changes, using the same zone helpers as `common.NextLocalKickoffAfter`. The last matchday's
  kickoff is `MAX(match_date)` of the country's season matches.
- **Where playoff cups are excluded:**
  - `GetActiveCupsStmtSql`, so `AdvanceCups` and the admin "active cups" count skip them.
  - `GetLatestCupForCountryStmtSql`, so the cup page never shows the playoff.
  - `awardCupRoundPrize` is not called when the tie's cup kind is `playoff`. `simulateCupTie`
    reads the kind with the existing `GetCup`.
  - `IsCupFinal` returns false for a playoff.
  - The cup-tie history line (`RecordSeasonHistoryCupResult`) is not written for a playoff.
- **Where playoff cups are included as-is:** `GetSimulableCupTies`, `FinishStuckCupTies`,
  lineups, AI kickoff, broadcast, gate and sponsorship, post-match effects, and rivalry points
  (as a normal non-final tie).
- **Labelling.** `CupTieDetails` and `TeamMatchSummary` gain `competition: "cup" | "playoff"`.
  The frontend shows "Playoff" / "Promoción" instead of the cup name and round, while routes stay
  `/cup/ties/:id`. The stored `cups.name` is `"Promotion playoff"` and is used only in logs and
  admin.

*Alternative considered:* separate `playoff_ties` tables. Rejected, because they would duplicate
simulation, lineups, broadcast, match viewer and the replica locks for a single round a season.

### Rollover phases

`advanceCountrySeasonIfComplete` stays one function under the same lock. After the existing two
gates it adds a third:

1. Re-read `GetCountryPlayoff(country, season)` → (exists, unresolved count, still-broadcasting
   count).
2. If the country has more than one active tier and no playoff exists, call `startPlayoff`, then
   return. `startPlayoff` computes pairings from `GetRankedClassification` of every active league
   (pure `ComputePlayoffPairings`), then in one transaction inserts the cup with
   `ON CONFLICT DO NOTHING` on the partial index and inserts its ties. Notifications are sent after
   commit, and only when the insert created the row.
3. If the playoff exists and any tie is unresolved, or any tie's `match_date > LiveBroadcastCutoff()`,
   return.
4. Otherwise run the existing rollover. `applyPromotionRelegation` now does the following:
   1. Builds `ComputeDirectMoves` from the final standings.
   2. Reads the playoff winners and builds `ApplyPlayoffOutcomes`: a swap for each tie the away
      side won.
   3. Applies both sets of moves.
   4. Marks the playoff cup `completed` with `champion_team_id NULL`.
   5. Records history with `via_playoff`.
5. `restructureIfNeeded` runs right after `applyPromotionRelegation` and before
   `StartCupForCountry`, so the new season's cup and fixtures see the new leagues.

`writeSeasonSeeds` takes the final placement, including restructure moves. A team that dropped
in the restructure counts as "relegated into" its league, and one that rose counts as "promoted".

The final standings used for prizes, income, awards and history are read before any move, as
today. The playoff does not change them.

### Restructure

New `engine/restructure.go`:

- **`PlanRestructure(teams []RankedTeam, target []TierConfig) Plan`** is pure.
  - It sorts by (tier after this season's moves, final position, −points, −GD, −GF, team_id).
  - It cuts the list into tiers by `Groups × TeamsPerLeague`, deals each tier's slice to groups
    in snake order, and returns per team `(tier, group)`.
  - `missing = places − teams` is the number of CPU teams to create in the last tier, filling the
    groups that are short.
- **`applyRestructure`** runs under the rollover lock and is idempotent by state:
  1. For every target (tier, group) it reuses the active league with the same `tier` and
     `group_number`, or creates it (`common.CreateLeague`).
  2. It moves teams with `UpdateTeamLeague` and `UpdateClassificationLeague`, sending a
     restructure notification when the tier changed.
  3. It creates `missing` CPU teams with `common.CreateTeam`, plus their classification rows.
  4. It sets `archived_season = <closing season>` on every active league left with no teams.

  A rerun after a crash sees a partly reshaped country. `restructureIfNeeded` compares the active
  leagues against the target again and plans from the current placement. Already-placed teams sort
  first, because tier is the first key. The rerun therefore converges, and team creation counts
  only what is still missing.
- **Team creation moves to `common/teamcreation.go`** (`CreateLeague`, `CreateTeam`,
  `CreateClassification`). `handlers/initializer.go` calls these instead of its private copies,
  so a new world and a restructure create teams identically.
- **Archived leagues** (`leagues.archived_season int NULL`):
  - Every listing used for play filters `archived_season IS NULL`. That covers
    `GetAllLeagueIdsStmtSql`, `GetLeaguesWithTierStmtSql`, `GetLeaguesWithTierForCountryStmtSql`,
    the adjacency query, `GetLeagueIdsForCountry*`, `leaguedirectory.go`, `leagueschedule.go`,
    `regionalcups.go`, the `MAX(tier)` in `seasonobjective.go` and the admin league count.
  - Lookups by id (`GetDivisionName`, the division page for a past season, history) do not filter,
    so old links keep working.
  - The division page of an archived league shows its last season and no current-season zones.

*Alternative considered:* re-tier existing league rows (II.3 → III.x). Rejected, because a
league id's past seasons would then show up under a different tier and name.

### Season history

`season_history.via_playoff boolean NOT NULL DEFAULT false` is set for teams moved by a playoff
outcome. `SeasonHistoryLine` and the league season-history response gain `viaPlayoff`. Teams moved
only by the restructure keep `promoted = relegated = false`, because it is not a sporting result.

### CPU season objective

`GetTeamSeasonSituation` reads `PyramidZones` instead of the removed config:
- `PromotionSlots = DirectUp + PlayoffUp`, which is 2 below tier 1. Tier 1 stays at 1, the title.
- `RelegationSlots = DirectDown + PlayoffDown`, which is 4 above the bottom tier.

That is a change in the numbers fed to `add-cpu-season-context`'s requirement, not in its rule.

### Staggered, localized league kickoffs

**Today:**
- `AdminAddCountryHandler` turns the requested local time (default `DefaultKickoffHour` 14:00)
  into a **server-clock** hour with `LocalTimeToServerSchedule`, anchored on today's date, and
  writes it as a `league_schedule` override on every league of the country.
- `NextScheduledOccurrence` builds dates in `from.Location()`, which is the server clock. A
  kickoff therefore drifts by an hour across a DST change.
- Leagues created later (restructure, add-division) would get no override and fall back to the
  app-wide default of 16:00 server time.

**New:**
- **Country setting.** `countries.league_kickoff_minute smallint NOT NULL DEFAULT 840` stores the
  local minute of day for tier 1. `common.GetCountryLeagueKickoff` / `SetCountryLeagueKickoff`
  validate it as a quarter-hour, 0 ≤ m and m + 15 × (maxTier − 1) ≤ 1439.
  `AdminAddCountryHandler` stores the requested time there and no longer writes overrides.
- **Effective schedule.** `common.EffectiveLeagueSchedule(league, override, default, kickoff)`:
  - An override is returned as today: server clock, admin-set.
  - Otherwise `LocalMatchSchedule{WeekdaysMask: default.WeekdaysMask, Hour/Minute: kickoff +
    LeagueKickoffTierStagger × (tier − 1), Location: countryLocation(country)}`, with
    `LeagueKickoffTierStagger = 15 * time.Minute`.
  - `ComputeFixtureDates` / `NextScheduledOccurrence` take the location explicitly and build each
    date with `time.Date(..., loc)` from `from.In(loc)`. Fixtures are then DST-correct, like
    `NextLocalKickoffAfter`.
  - `generateMatchesForLeagueIds` and `RescheduleUnplayedMatchesForLeagues` both go through it,
    and so does `ListLeaguesForAdmin`, so the admin table shows the derived time and the fact that
    it is derived.
- **The app-wide default** keeps only its weekdays for leagues without an override. Its hour and
  minute no longer date league fixtures, so the `RulesTab` "match schedule" form drops the time
  field and becomes "match days".
- **Changing a country's kickoff.** `PUT /v1/admin/countries/:code/kickoff` saves the value, then
  calls `engine.RescheduleUnplayedMatchesForCountry`, which already exists for the "reschedule
  country" button and re-dates only leagues without an override.
- **Rollout.**
  - The migration adds the column (every country gets 14:00) and deletes every `league_schedule`
    row. Owner assumption: all current overrides are the automatic 14:00 ones from adding a
    country. An override the admin sets afterwards wins again.
  - Unplayed matches of the season in progress keep their dates until the restructure rollover
    regenerates fixtures. The admin can press "reschedule country" to move them at once.
- **Interaction with the playoff.** With a 14:00 kickoff, the tier-5 broadcasts end around 16:50.
  The playoff is the next day at 14:00 to 14:45, by tier pair, and the last one (IV–V) ends around
  16:35. The new season's first matchday is the following day at 14:00, about 21 h later, so no
  extra pause is needed.

*Alternative considered:* keep writing server-clock overrides per league, with the tier offset
applied at creation. Rejected, because it keeps the DST drift, and every league a restructure or
add-division creates would need its own conversion.

### Migration `init110_league_pyramid_playoffs.sql`

Re-runnable:
1. `ALTER TABLE cups ADD COLUMN IF NOT EXISTS kind varchar(16) NOT NULL DEFAULT 'cup'`. A guarded
   `DO` block adds `cups_kind_check`, and
   `CREATE UNIQUE INDEX IF NOT EXISTS cups_playoff_once ...` follows.
2. `ALTER TABLE leagues ADD COLUMN IF NOT EXISTS archived_season int`, and
   `CREATE INDEX IF NOT EXISTS leagues_active_idx ON leagues (country, tier) WHERE archived_season IS NULL`.
3. `ALTER TABLE season_history ADD COLUMN IF NOT EXISTS via_playoff boolean NOT NULL DEFAULT false`.
4. `UPDATE world_config SET tiers = '<new default>'::jsonb WHERE tiers = '<old 1/4/16×10 default>'::jsonb`.
   A world whose admin saved a custom structure is left alone. If that structure breaks the new
   rule, the next rollover still plays the old pyramid's zones. The admin UI flags it as invalid
   until it is fixed (see Risks).
5. `ALTER TABLE countries ADD COLUMN IF NOT EXISTS league_kickoff_minute smallint NOT NULL DEFAULT 840`,
   then `DELETE FROM league_schedule`. The delete is guarded to run once: a `DO` block only runs it
   when the column did not exist before this migration.

### Money and notifications

- **No new `TRANSACTION_*` reason** (owner decision: no playoff prize). A playoff tie earns only
  the existing gate and sponsorship through `UpdateTeamFinancesAfterMatch`.
- **New notification keys**, each with a Spanish template in `notificationtranslations.go` and
  en/es strings in the frontend locales:
  - `notifications.messages.playoffQualified` {team, opponent}, `NOTIFICATION_CUP`, linking to
    `/cup/ties/<id>`.
  - `notifications.messages.promotedViaPlayoff` {team}, `NOTIFICATION_PROMOTION`.
  - `notifications.messages.relegatedViaPlayoff` {team}, `NOTIFICATION_RELEGATION`.
  - `notifications.messages.restructureMoved` {team, league}. It uses `NOTIFICATION_RELEGATION`
    when the tier number grew and `NOTIFICATION_PROMOTION` when it shrank.
- **Visibility.** All of them are sent only after the relevant broadcasts have finished. That
  follows from gates 1–3, so nothing leaks before a replay ends.

### Frontend

- **`DivisionDetailsPage.tsx`.**
  - `zoneFor` takes `{directUp, playoffUp, playoffDown, directDown}` from `DivisionDetails`. That
    type replaces `promotionSlots/relegationSlots` with a `zones` object.
  - A third zone colour, `theme.palette.warning.main`, gets a legend key `division.legend.playoff`.
  - A new `PlayoffCard` renders `DivisionDetails.playoff` (ties involving this league, built by
    `common.GetLeaguePlayoffTies`).
  - Past-season rows show `(playoff)` from `viaPlayoff`.
- **Admin.**
  - `RulesTab` drops `PromotionRelegationForm` and its hook.
  - `WorldTab` shows the backend validation message, the places per country, and a "Pending:
    applied at each country's next season rollover" note whenever the saved structure differs from
    a country's active leagues. The overview gains `pendingRestructure: string[]` (country codes).
- **Cup tie and dashboard labels** use `competition` to show "Playoff".

## Risks / Trade-offs

- **[Risk] A custom saved structure is invalid under the new rule.** The rollover keeps playing
  it with derived zones, but `ComputePlayoffPairings` assumes the ratio is 2. → `restructureIfNeeded`
  and `startPlayoff` skip a country whose active leagues break the rule, and log an error. The
  admin overview lists it, so the admin re-saves a valid structure.
- **[Risk] 100 new CPU teams per country are created inside the rollover lock.** That means
  players, a lineup and trainer candidates for each team, and the rollover runs for minutes. →
  This is acceptable, because the lock is non-blocking for other replicas and a new world already
  does the same work synchronously. Teams are created in batches with progress logs.
- **[Risk] A big economy shift.** Tier 3 shrinks from 160 to 40 teams, and 240 teams now sit in
  tiers 4–5 with tier-3 money. → Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v`
  before and after, and record the result in the PR. Re-tuning stays out of scope.
- **[Trade-off] Relegation zones are a big share of a 10-team group.** In a middle tier, 6 of 10
  teams are in some zone. The owner accepted this for 10-team groups.
- **[Risk] The season runs a few days longer.** The playoff adds one matchday, at the league times of
  the day after the last league matchday, plus its own broadcast window, before the next season's fixtures start.

## Migration Plan

1. Deploy the migration and the binary together. Existing countries keep playing the season in
   progress. Their zones switch to the derived ones, so the current 1/4/16 table shows 1 direct
   promotion plus a playoff place.
   - A 1/4/16 country breaks the ratio rule, so step 2 of the rollover gate skips the playoff for
     it this one season.
   - Its rollover applies the old-shaped direct moves: the 2 best of the 4 tier-2 leaders go up
     and tier 1's 9th and 10th go down, through `computeLegacyMoves` (`ComputeSeasonMoves` picks it for a
     pyramid that breaks the ratio rule). Then it restructures.
2. From the next season on, every country plays the playoff.
3. **Rollback.** The previous binary ignores `cups.kind`, which would make it advance a playoff as
   a cup. Roll back only between seasons, when no playoff cup is active. Archived leagues would
   reappear in listings with zero teams. This is cosmetic, because they get no fixtures without
   teams.

## Code map (for the new league-pyramid spec at archive; the league-page, engine-replicas and match-scheduling Code maps are already updated)

### league-pyramid

- Structure: `backend/common/worldconfig.go` (`DefaultWorldTiers`, `ValidateWorldConfig`,
  `ValidateWorldConfigFitsTeams`, `CountTeamsPerActiveCountry`), checked by
  `handlers.AdminSetWorldConfigHandler`; mirrored for the editor in
  `frontend/src/features/admin/pyramidRules.ts`.
- Zones: `backend/common/pyramid.go` (`PyramidZones`, `ZoneOf`, `PyramidShape` with
  `FollowsRatioRule`/`MatchesConfig`, `GetCountryPyramidShape`, `GetLeagueZones`,
  `ArchiveLeagueIfEmpty`); used by `common.GetDivisionDetails` and `common.GetTeamSeasonSituation`.
- Moves: `backend/engine/promotion.go` (`ComputeDirectMoves`, `ComputePlayoffPairings`,
  `ApplyPlayoffOutcomes`, `computeLegacyMoves`, `ComputeSeasonMoves`), applied by
  `applyPromotionRelegation` (`engine/season.go`).
- Playoff: `backend/engine/playoff.go` (`startPlayoff`, `playoffKickoff`, `playoffResults`,
  `notifyPlayoffQualified`), the gate in `advanceCountrySeasonIfComplete` (`engine/season.go`),
  `backend/common/playoffs.go` (`CreatePlayoff`, `GetCountryPlayoff`, `IsPlayoffCup`,
  `GetLastLeagueMatchDate`, `GetLeaguePlayoffTies`), `common/cups.go` (`Cup.Kind`, `CupKind*`,
  `SetCupCompletedWithoutChampion`), `engine.awardCupRoundPrize` (no prize for a playoff),
  `common.IsCupFinal` (never a playoff).
- Restructure: `backend/engine/restructure.go` (`PlanRestructure`, `restructureIfNeeded`,
  `applyRestructure`, `notifyRestructureMoved`, the `restructureTarget` hook that
  `engine/main_test.go` switches off), `backend/common/teamcreation.go` (`CreateLeague`,
  `CreateTeam`, `CreateClassification`); pending countries in `common.GetPendingRestructureCountries`
  (admin overview `pendingRestructure`).
- History: `season_history.via_playoff` (`common.RecordSeasonHistoryLeagueResult`,
  `SeasonHistoryEntry.ViaPlayoff`, `LeagueSeasonPosition.ViaPlayoff`).
- Data: `backend/resources/sql/init110_league_pyramid_playoffs.sql` (`cups.kind`,
  `cups_playoff_once`, `leagues.archived_season`, `season_history.via_playoff`, the world_config
  move to the new default, `countries.league_kickoff_minute`).
- Notifications: `notifications.messages.playoffQualified`, `promotedViaPlayoff`,
  `relegatedViaPlayoff`, `restructureMoved` (`common/notificationtranslations.go`, frontend locales).
- Tests: `backend/common/pyramid_test.go`, `common/worldconfig_test.go`,
  `common/archivedleagues_integration_test.go`, `engine/promotion_test.go`, `engine/playoff_test.go`,
  `engine/restructure_test.go`, `engine/pyramid_integration_test.go`,
  `handlers/worldConfigHandler_test.go`.
