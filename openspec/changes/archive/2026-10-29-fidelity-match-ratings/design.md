# Design

## Context

- `common.ComputeMatchRatings` (`common/matchratings.go`) runs **on read** in `WeekMatchById`,
  `GetCupTieDetails` and `EnrichPlayedFriendly`. It also runs in-memory in
  `engine.matchRatingsForProgression` for Form drift. It iterates the lineup's `Positions`:
  - On read, `WeekMatchById` first replays substitutions onto the starting XI
    (`ApplySubstitutionToLineup`), so whoever came off is missing.
  - Progression gets the starting XI from the post-match payload, so whoever came on is missing.
  It adds the current Form and a hash jitter, so an old match's ratings drift as Form changes.
- The frontend has a second copy of the formula, `utils/liveMatchRatings.ts`, for live and
  scrubbed ratings. `MatchMinuteRatings` swaps in the backend's `ratings` at the final minute.
- Every simulated minute, `InitialSimulation` (`engine/initial_simulation.go`) builds each side's
  `TeamQualification` from `*PlayersByPosition` and the fatigue-adjusted `Effective*Players`
  (`calculateQua`). It then appends a `MatchInstant`. Instants go into `match_instants`,
  `cup_tie_instants` and `friendly_instants` inside the `Save*ResultTx` transaction
  (`replaceMatchEventsTx`). They are replayed live by `BroadcastMatchLive` and after a restart by
  `livematch_recovery.go`.
- How each position feeds the zones (`calculateQua`, raw values before `blendLine`):
  - GOALKEEPER: `GoalKeeper`. The zone duel adds it to all three defensive duels.
  - DEFENSE: `Defense` → central defense.
  - LATERAL_DEFENSE_L/R: flank defense (`Lateral`/`Defense`, 0.5/0.5, or 0.35/0.65 with a back
    three) and flank midfield (`Lateral`/`Midfield`, 0.5/0.5, or 0.65/0.35).
  - MIDFIELD: `Midfield` → central midfield.
  - LATERAL_MIDFIELD_L/R: flank attack (`Lateral`/2 + `Shot`/2), only when that side has no
    lateral striker.
  - STRIKER: `Shot` → central attack.
  - LATERAL_STRIKER_L/R: flank attack, plus `Shot` into central attack.
- Duels are mirrored (`ZoneComparisonRows.tsx`, `processChance`): own left vs rival right.

## Goals / Non-Goals

**Goals:**
- One formula, computed once in the engine, stored and returned as-is. The frontend formula only
  rebuilds intermediate minutes and must land on exactly the stored number at the final minute.
- The zone component reads the very numbers the simulation used that minute.
- No change to simulation outcomes: same chances, same goals for the same random draws.

**Non-Goals:**
- Backfilling old matches. They use the read-time fallback.
- New event types (assists, saves).
- Any change to `TeamQualification`, `blendLine` or the duel display.

## Decisions

1. **Per-minute contributions live on the instant.** `MatchInstant` gains
   `PlayerContributions *InstantContributions` (JSON `playerContributions`): one map per side,
   `{"h": {id: …}, "a": {id: …}}`, so the formula knows each player's team without the lineups.
   Each value is `PlayerContribution{Position, Own, Rival}` (JSON `p`, `o`, `r`), with `Own` and
   `Rival` rounded to one decimal **before** anything uses them. A new nullable `player_contributions JSONB` column is added to the three instants
   tables, and the three `Insert*InstantStmtSql` statements and instant readers get the column.
   `InitialSimulation` fills the map right after `calculateQua`, from the same
   `*PlayersByPosition` and `Effective*Players`, through a new pure
   `engine.playerContributions(match)` in `initial_simulation.go`. The position math is factored
   out of `calculateLateralDefense` / `calculateLateralMidfield` / `calculateLateralAttack` into
   per-player helpers, so the zone totals and the per-player figures share one code path.
   - Rival value per duel:
     - Central duels: the rival line's raw total divided by its head count.
       Strikers count `allStrikers`.
     - Flank duels: the single rival flank player's value. An empty rival line or flank reads 0.
     - Goalkeeper: the mean of the rival's per-attacker values (central strikers' `Shot`, flank
       attackers' flank-attack value).
   - *Alternative:* storing a per-player skill snapshot and recomputing fatigue on the client.
     Rejected: it duplicates engine math in TypeScript.
   - *Alternative:* storing only final aggregates. Rejected: the live view and the scrubber
     would go back to an approximation.

2. **The formula is pure and shared by the engine and the read path.** It lives in
   `common/matchratings.go`:
   ```
   ComputePlayerRatings(homeTeamId, awayTeamId int64, homeLineup, awayLineup *Lineup,
                        instants []MatchInstant, events []Event, upToMinute int) []PlayerRating
   ```
   One function covers both kinds of match, so the TypeScript port is one function too:
   - **With contributions** (any instant ≤ upToMinute carries them): minutes = the number of
     those instants listing the player; the team is the side whose map lists him.
   - **Without** (a match simulated before this change - the legacy fallback): the starting
     lineups decide who started, a SUBSTITUTION brings the incoming player in at the outgoing
     one's slot, and minutes run from kickoff or `cameOn` to the final minute (capped at 90, or
     the abandonment minute) or `wentOff`. Zone, own and rival are 0 or absent. Returns nil when
     a lineup is missing.
   - Each component is rounded half away from zero to one decimal (`roundTenth`, negative zero
     normalized to 0) before they are summed, so the breakdown the UI shows adds up to the
     rating. Zone = clamp(3.0 × mean edge, ±1.5); result = ±0.3 × min(minutes, 90) / 90.
   - Events are credited to every id in `TeamOwnerPlayers` of GOAL / YELLOW_CARD / RED_CARD /
     INJURY up to `upToMinute`, as today.
   - The result uses the score of the last GOAL or MATCH_ABANDONED event ≤ upToMinute (an
     abandonment carries the forfeit score).
   - `cameOn` / `wentOff` come from the events: the SUBSTITUTION that brought him on, and the
     SUBSTITUTION, RED_CARD or INJURY that took him off.
   - `MatchFinalMinute(instants, events)` (last instant or event minute, at least 90) is the
     minute the stored ratings are computed at, the same rule as the frontend scrubber's end.
   - `rated` = minutes ≥ min(10, minutes elapsed up to `upToMinute`). An unrated record keeps
     its components, `rating` is still computed for the breakdown, but `Ratings` (id → number),
     the player's match history and Form drift treat him as having no rating.
   - `PlayerRating` = `{playerId, teamId, rated, rating, minutes, position, cameOn, wentOff, own,
     rival, zone, goals, cards, injury, result}`.
   - Constants: `matchRatingBase` 6.0, `matchRatingZoneWeight` 3.0, `matchRatingZoneCap` 1.5,
     `matchRatingResultBonus` 0.3. Event deltas are unchanged. `matchRatingFormSwing` and
     `matchRatingJitter` are removed.
   - `position` is the one listed most often, ties going to the one he played first.

3. **Storage: table `match_player_ratings`**, created by migration
   `backend/resources/sql/init89_match_player_ratings.sql` (`CREATE TABLE IF NOT EXISTS`, plus
   `ALTER TABLE … ADD COLUMN IF NOT EXISTS player_contributions JSONB` on the three instants
   tables):
   ```
   kind TEXT NOT NULL,            -- 'league' | 'cup' | 'friendly' (SimulationKind*)
   match_id BIGINT NOT NULL,      -- matches.match_id / cup_ties.tie_id / friendly_matches.match_id
   player_id BIGINT NOT NULL, team_id BIGINT NOT NULL,
   rated BOOLEAN NOT NULL, rating REAL NOT NULL, minutes INT NOT NULL, position INT NOT NULL,
   came_on INT NULL, went_off INT NULL, own_value REAL NULL, rival_value REAL NULL,
   zone_delta REAL NOT NULL, goals_delta REAL NOT NULL, cards_delta REAL NOT NULL,
   injury_delta REAL NOT NULL, result_delta REAL NOT NULL,
   PRIMARY KEY (kind, match_id, player_id)
   ```
   - It uses one table keyed by `kind`, the same pattern as `pending_post_match_effects`, so it
     has no foreign key to the three match tables.
   - `MatchResult` gains `PlayerRatings []PlayerRating`. `replaceMatchEventsTx` gets the kind,
     deletes and reinserts the ratings next to events and instants, and writes the contributions
     column. That keeps the retry-safe "replace everything" semantics and the single-transaction
     guarantee.
   - The engine computes `PlayerRatings` with `ComputePlayerRatings(…, upToMinute = last
     minute)` after `runMatchSimulation`, in each of the three simulate paths (league, cup,
     friendly) and in the default-result path (no instants, so no ratings).
   - Constants `InsertMatchPlayerRatingStmtSql`, `DeleteMatchPlayerRatingsStmtSql` and
     `GetMatchPlayerRatingsStmtSql`, wrapped by `GetMatchPlayerRatings(dbCtx, kind, id)`.

4. **Read path.** `WeekMatch`, `CupTieDetails` and `FriendlyMatch` gain
   `PlayerRatings []PlayerRating` (`playerRatings`). `Ratings` (id → rating) stays and is derived
   from `PlayerRatings`, so `common/playermatches.go`, `MatchPitchChip` and any other consumer of
   the map keep working.
   - All three read paths go through `LoadPlayerRatings(dbCtx, kind, id, …)`: the stored rows,
     or, with none, `ComputePlayerRatings` over the **starting** lineups (`WeekMatchById` copies
     them before `ApplySubstitutionToLineup` runs) plus instants and events at
     `MatchFinalMinute`.
   - The players map on read must include players who came off. `addRatedPlayers` fetches any
     rated id missing from both sides' maps through `GetPlayersByIds`, so national teams and
     transferred players still have a name.

5. **Form drift reads the engine's ratings.** `engine.newMatchResult` computes the ratings once
   and fills both `MatchResult.PlayerRatings` and `PostMatchPayload.Ratings` (`ratings`, rated
   players only). `applyPostMatchEffects` copies them into `MatchSimulation.StoredRatings`, which
   `matchRatingsForProgression` returns. When they are absent (a payload stored before this
   build, still pending), it falls back to `ComputePlayerRatings` over the payload's lineups and
   events.

6. **Frontend.**
   - `utils/liveMatchRatings.ts` is rewritten as a line-by-line port of `ComputePlayerRatings`
     (`computePlayerRatings`, same arguments), plus `startingLineup`, which undoes the
     substitutions a finished league match's lineup comes back with.
   - Parity is pinned by one shared fixture, `backend/common/testdata/match_ratings_parity.json`
     (instants, events, expected ratings at several minutes). It is read by both
     `matchratings_test.go` and `liveMatchRatings.test.ts`. vitest imports it by relative path.
   - `MatchMinuteRatings` takes `instants` (live `live.instants`, finished `match.matchInstants`)
     and `finalPlayerRatings`. At the final minute it shows the stored records.
   - `TeamRatingsColumn` renders a `PlayerRating[]` per team: starters in lineup order, then
     substitutes by `cameOn`. Each row shows a position abbreviation, minutes with "↑ N'" /
     "↓ N'", "own vs rival" with a small share bar colored like the zone duel, and the rating
     `Chip`. The breakdown is a `Tooltip` (hover) or a `ClickAwayListener` popover (tap).
   - Old matches without instants' contributions hide the contribution figure.
   - New i18n keys under `match.ratings.*`: `cameOn`, `wentOff`, `contribution`,
     `breakdown.{base,zone,goals,cards,injury,result}`, `positionShort.*` (reusing existing
     position labels where they exist).

7. **Live broadcast.** Contributions travel inside each `instant` message for free, since they
   are a field of `MatchInstant`. Both `BroadcastMatchLive` and recovery read them back from
   `Get*Instants`. Live events are still filtered by `withholdSpoilerEvents` before reaching the
   rating function.

No TRANSACTION_* reason and no notification key is added.

## Risks / Trade-offs

- **Storage:** about 23 players × ~35 bytes × 90 instants ≈ 70 KB of JSON per match before
  TOAST. Each row stays under the 2 KB TOAST threshold, so it is stored uncompressed. Mitigation:
  compact JSON keys (`p`, `o`, `r`) and one-decimal values bring it to ~40-50 KB. Open question 4
  in the proposal asks whether that is acceptable.
- **Go/TS drift:** two implementations of one formula. Mitigated by the shared parity fixture,
  which both test suites assert.
- **Spoiler via contributions:** live, a sent-off player stops appearing in the instants. The
  zone duel already reveals the same drop, so no new information leaks.
- **Rating distribution shifts:** Form drift is now ±3 wider (zone ±1.5 × 2). Players in strong
  zones climb in Form faster. Watch Form's distribution after a few matchdays. The weight is a
  single constant.
- **Legacy pending payloads** (stored before deploy, applied after): handled by the fallback in
  decision 5.
- **Concurrency:** ratings are written in the existing result transaction, which is already
  claim-guarded (`played = false` predicate), so replicas can't double-write. No new engine job.
