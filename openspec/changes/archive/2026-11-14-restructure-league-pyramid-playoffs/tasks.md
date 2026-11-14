# Tasks

## 1. Schema and structure rule

- [x] 1.1 Add `backend/resources/sql/init110_league_pyramid_playoffs.sql` (re-runnable):
  - `cups.kind` with its check constraint and the `cups_playoff_once` partial unique index.
  - `leagues.archived_season` with the `leagues_active_idx` index.
  - `season_history.via_playoff`.
  - A guarded `UPDATE world_config`, from the old 1/4/16×10 default to the new 1/2/4/8/16×10.
- [x] 1.2 Add `common/pyramid.go`:
  - Constants: `PyramidGroupRatio`, `MinPyramidTeamsPerLeague`, `DirectUpPerGroup`,
    `PlayoffUpPerGroup`, `PlayoffDownPerGroup`, `DirectDownPerGroup`.
  - `PyramidZones` (including `Legacy` and `Candidate`) and `ZoneOf`, plus a query that reads a
    country's active tier/group shape.
  - Unit tests for tiers 1, middle and bottom, a single-tier country and a legacy country.
- [x] 1.3 In `common/worldconfig.go`:
  - Change `DefaultWorldTiers` to I..V with 1/2/4/8/16 × 10.
  - Extend `ValidateWorldConfig` with contiguous tiers, `Groups(1)=1`, the 2× ratio, equal and
    even `TeamsPerLeague ≥ 8`, and the spec's exact messages.
  - Update `worldconfig_test.go`.
- [x] 1.4 Add `CountTeamsPerActiveCountryStmtSql`. Reject a structure with fewer places than any
  active country's teams in `AdminSetWorldConfigHandler`, and add a handler test.
- [x] 1.5 Filter `archived_season IS NULL` in every play listing named in design.md: the league
  listings in `database.go`, `worldreset.go`, `leaguedirectory.go`, `leagueschedule.go`,
  `regionalcups.go`, `seasonobjective.go`, `admin.go` and `engine/aibusiness.go` where it lists.
  Lookups by id stay unfiltered. Add an integration test that an archived league gets no fixtures
  and is not listed.

## 2. Moves and zones

- [x] 2.1 Rewrite `engine/promotion.go`:
  - `ComputeDirectMoves` (worst-first relegated ↔ best-first winners, swap).
  - `ComputePlayoffPairings` (worst-first upper playoff teams vs best-first runners-up, upper
    team at home).
  - `ApplyPlayoffOutcomes` (swap only when the away side won).
  - Keep today's algorithm as `computeLegacyMoves` with 1 candidate and 2 relegations.
  - Rewrite `promotion_test.go` against the spec scenarios: tier 1/2, two upper groups, a
    playoff won by each side, and the legacy country.
- [x] 2.2 Delete `common/promotionconfig.go` and its uses: `/v1/admin/promotion-relegation` in
  `adminHandlers.go` / `fmservice.go`, and the overview fields in `common/admin.go`. Update their
  tests.
- [x] 2.3 Replace `leaguePromotionRelegationSlots` in `common/database.go` with `PyramidZones`.
  `DivisionDetails` exposes `zones {directUp, playoffUp, playoffDown, directDown, candidate}`
  instead of `promotionSlots/relegationSlots`.
- [x] 2.4 Make `GetTeamSeasonSituation` (`common/seasonobjective.go`) use `PyramidZones`, with
  promotion = 2 and relegation = 4. Update `seasonobjective_test.go`, and update
  `add-cpu-season-context`'s spec delta (or the main `cpu-team-management` spec if that change
  is already archived) where it states the slot numbers.

## 3. Playoff phase

- [x] 3.1 In `common/cups.go`:
  - `kind` on `Cup`.
  - `InsertPlayoffCupStmtSql` with `ON CONFLICT DO NOTHING`.
  - `GetActiveCupsStmtSql` and `GetLatestCupForCountryStmtSql` restricted to `kind = 'cup'`.
  - `GetCountryPlayoffStmtSql`, which returns whether it exists, the unresolved count and the
    still-broadcasting count.
  - `GetPlayoffWinnersStmtSql`.
  - `SetCupCompleted` accepting a NULL champion.
- [x] 3.2 Add `engine/playoff.go` `startPlayoff`, which does pairings from the active leagues'
  ranked standings, then one transaction for the cup and ties, then kickoff via a pure
  `playoffKickoff` (the day after the last league matchday's local date, at the upper tier's league
  kickoff time, one more day if that is under 1 h away). It has unit tests for the spec scenarios and a DST change. After commit it sends `playoffQualified` notifications with a link to
  `/cup/ties/<id>`. It skips single-tier and legacy countries.
- [x] 3.3 In `engine/cups.go` / `prizemoney.go`, skip `awardCupRoundPrize` and the cup history
  line for a playoff tie, and make `IsCupFinal` return false for a playoff. Add unit or
  integration coverage.
- [x] 3.4 Add the third gate to `advanceCountrySeasonIfComplete` (create the playoff if missing
  and return, wait while unresolved or broadcasting). Then, in `applyPromotionRelegation`, apply
  direct moves and playoff outcomes, mark the playoff completed, and record `via_playoff` in
  `recordSeasonHistory` / `RecordSeasonHistoryLeagueResult`.
- [x] 3.5 Send `promotedViaPlayoff` / `relegatedViaPlayoff` from `notifyPromotionRelegation` for
  playoff moves.
- [x] 3.6 Add `competition: "cup" | "playoff"` on `CupTieDetails` and `TeamMatchSummary`
  (`common/teammatches.go`).
- [x] 3.7 Add `engine/playoff_integration_test.go`:
  - The full flow: league ends → ties created → simulated → rollover moves.
  - The rollover waits while a tie is broadcasting.
  - `AdvanceCups` ignores the playoff.
  - A concurrent test where 3 goroutines run the season job once the league ends, which must give
    exactly one playoff, one set of ties and one notification per team.

## 4. Restructure

- [x] 4.1 Move `createLeague` / `createTeam` / `createClassification` from
  `handlers/initializer.go` to `common/teamcreation.go` (`CreateLeague`, `CreateTeam`,
  `CreateClassification`). Switch the handlers to them, with no behavior change, and keep the
  existing initializer tests passing.
- [x] 4.2 Add `engine/restructure.go`:
  - Pure `PlanRestructure` (ordering, cut per tier, snake dealing, missing count).
  - `restructureIfNeeded` / `applyRestructure` (reuse or create leagues, move teams and
    classification rows, create missing CPU teams, archive empty leagues, send
    `restructureMoved` notifications).
  - Call it after `applyPromotionRelegation` and before `StartCupForCountry`.
- [x] 4.3 Make `writeSeasonSeeds` / `ComputeSeasonSeeds` take the final placement including
  restructure moves, so dropped teams count as relegated-in and risen teams as promoted-in.
- [x] 4.4 Add `engine/restructure_test.go` (pure), covering the 210 → 310 example from the spec
  (10/20/40/80/160 per tier, 100 new teams), snake order, and "already restructured → no-op".
- [x] 4.5 Add `engine/restructure_integration_test.go`:
  - A 1/4/16 country rolls over into 1/2/4/8/16, which means the legacy moves are applied, 14
    leagues are archived, 24 are created, 100 teams are added, and fixtures exist only for active
    leagues.
  - A rerun after the step that moves teams converges.
  - 3 concurrent rollovers restructure once.

## 5. Staggered, localized league kickoffs

- [x] 5.1 Add `countries.league_kickoff_minute` and the one-off `league_schedule` cleanup to
  `init110`. Add `common.GetCountryLeagueKickoff` / `SetCountryLeagueKickoff` with validation
  (quarter-hour, deepest tier ≤ 23:59) and unit tests.
- [x] 5.2 Have `NextScheduledOccurrence` / `ComputeFixtureDates` take a `*time.Location` and
  date in it. Add `common.EffectiveLeagueSchedule` (override > country kickoff + 15 min × (tier −
  1) with the default weekdays), with `LeagueKickoffTierStagger`. Use it in
  `generateMatchesForLeagueIds`, `RescheduleUnplayedMatchesForLeagues` and `ListLeaguesForAdmin`.
  Update `schedule_test.go` with the spec scenarios: tiers 14:00..15:00, a DST change, and an
  override winning.
- [x] 5.3 In `AdminAddCountryHandler`, store the kickoff on the country instead of writing
  overrides. Add `PUT /v1/admin/countries/:code/kickoff`, which saves and then calls
  `RescheduleUnplayedMatchesForCountry`. Make sure add-division and restructure-created leagues
  get no override. Add handler tests.
- [x] 5.4 Add an integration test where a fresh country with 5 tiers gets fixtures at
  14:00/14:15/14:30/14:45/15:00 local, and a kickoff change re-dates only that country's leagues
  without an override.

## 6. Notifications and history API

- [x] 6.1 Add the Spanish templates for `notifications.messages.playoffQualified`,
  `promotedViaPlayoff`, `relegatedViaPlayoff` and `restructureMoved` to
  `common/notificationtranslations.go`.
- [x] 6.2 Expose `viaPlayoff` on the season-history line and on the league past-season table
  response (`common/seasonhistory.go`, `handlers/seasonHistoryHandlers.go`). Update the handler
  tests.
- [x] 6.3 Add `common/playoffs.go` `GetLeaguePlayoffTies(leagueId)`, which returns the current
  season's playoff ties with a side in that league, and include it as `DivisionDetails.playoff`.
- [x] 6.4 Add `pendingRestructure: string[]` to the admin overview, listing countries whose
  active leagues differ from the saved structure or break the rule.

## 7. Frontend

- [x] 7.1 Update the `DivisionDetails` (`zones`, `playoff`), `SeasonHistory` (`viaPlayoff`),
  `CupTie` / `TeamMatchSummary` (`competition`) and admin overview (`pendingRestructure`;
  promotion-relegation fields removed) types in `api/types.ts`, and drop
  `useAdminSetPromotionRelegation` from `api/hooks.ts`.
- [x] 7.2 In `DivisionDetailsPage.tsx`, have `zoneFor` take the zones object and add the
  playoff/warning colour, a legend that shows only the zones present, and the "Promotion
  candidate" legend for legacy. Apply the same zones in the live table. Add
  `DivisionDetailsPage.zones.test.tsx`.
- [x] 7.3 Add `features/division/PlayoffCard.tsx` (ties, kickoff, score after the broadcast,
  penalties mark, link to `/cup/ties/:id`), shown on the current season only. Add
  `PlayoffCard.test.tsx`, including a 360 px layout check.
- [x] 7.4 Show `(playoff)` on past-season rows from `viaPlayoff`, and update
  `DivisionDetailsPage.seasons.test.tsx`.
- [x] 7.5 Show "Playoff" instead of the cup name and round wherever a tie or summary has
  `competition = "playoff"`: the cup tie page, team matches, the dashboard's next match and
  pending items.
- [x] 7.6 In admin, remove `PromotionRelegationForm` from `RulesTab.tsx` (and update
  `RulesTab.test.tsx`). In `WorldTab.tsx`, show the backend validation message, places per
  country, and the "pending at next rollover" note per `pendingRestructure`. Update
  `WorldTab.test.tsx`.
- [x] 7.7 Admin scheduling:
  - `RulesTab.tsx`'s match schedule form keeps only the weekdays ("match days").
  - `WorldTab.tsx`'s countries section shows and edits each country's league kickoff (local
    time, quarter-hour picker) through the new endpoint.
  - The per-league schedule table marks derived times ("tier 1 + 15 min × N") apart from
    overrides.
  - Update `RulesTab.test.tsx` and `WorldTab.test.tsx`.
- [x] 7.8 Update `i18n/locales/en.json` and `es.json`:
  - Add `division.legend.playoff`, `division.legend.candidate`, `division.playoff.*`, the
    `(playoff)` mark and `competition.playoff`.
  - Add the four new `notifications.messages.*` keys.
  - Add the `admin.worldStructure.*` places and pending strings.
  - Add the `admin.countries.kickoff*` and `admin.matchSchedule.*` "match days" strings.
  - Remove `admin.promotionRelegation.*`.

## 8. Specs and docs

- [x] 8.1 Update the Code maps: `league-pyramid` (new) and `league-page` (zones, `PlayoffCard`,
  strings), plus `engine-replicas` (playoff creation, restructure, `playoff_integration_test.go`,
  `restructure_integration_test.go`) and `match-scheduling` (`EffectiveLeagueSchedule`,
  `countries.league_kickoff_minute`, the location-aware `ComputeFixtureDates`).
- [x] 8.2 Check `team-finances` (prize money / wage factor "tier 3 and below" wording covers tiers
  4–5), `fanbase` (tier targets fall back to tier 3) and `season-awards` (awards per league,
  archived leagues excluded), and correct any statement the new pyramid makes false.
- [x] 8.3 Update `ARCHITECTURE.md` and `backend/README.md` wherever they describe the 1/4/16
  pyramid or the promotion-relegation admin setting.

## 9. Verification

- [x] 9.1 Run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` before and after the
  change, and note the tier 4–5 budget trend in the PR.
- [x] 9.2 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`.
- [x] 9.3 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`.
