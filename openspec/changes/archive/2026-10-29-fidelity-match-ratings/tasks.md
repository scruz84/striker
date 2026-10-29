# Tasks

These tasks follow the owner decisions in proposal.md: zone weight 3.0 capped at ±1.5, Form out
of the rating, "s.c." under 10 minutes, and per-minute contributions stored.

## 1. Schema

- [x] 1.1 Add `backend/resources/sql/init89_match_player_ratings.sql`: `CREATE TABLE IF NOT EXISTS match_player_ratings (…)` per design §3 and `ALTER TABLE … ADD COLUMN IF NOT EXISTS player_contributions JSONB` on `match_instants`, `cup_tie_instants` and `friendly_instants`. Verify the file applies twice in a row on a local database without error, and that the migration runner picks it up (check how init88 is registered).

## 2. Per-minute zone contributions (engine)

- [x] 2.1 Add `PlayerContribution {Position int; Own, Rival float32}` (compact JSON keys `p`, `o`, `r`) and `MatchInstant.PlayerContributions *InstantContributions` (`playerContributions`, omitempty; one map per side, design §1) in `common/fmstructs.go`. Verify with `go build ./...`.
- [x] 2.2 In `engine/initial_simulation.go`, factor the per-player position math out of `calculateLateralDefense` / `calculateLateralMidfield` / `calculateLateralAttack` / central sums. Add `playerContributions(match)`, which builds both sides' maps from `*PlayersByPosition` and `Effective*Players` with mirrored rivals and one-decimal rounding (design §1). Fill it on each appended instant. Verify: the existing `initial_simulation_test.go` still passes unchanged (zone totals identical), and new table tests cover a 4-4-2 vs 4-3-3 minute (central defender, full-back mean of two duels, lateral striker, goalkeeper vs mean attacker, empty rival flank after a red card), matching the spec scenarios' numbers.
- [x] 2.3 Persist the column: add `player_contributions` to `InsertMatchInstantStmtSql`, `InsertCupTieInstantStmtSql` and `InsertFriendlyInstantStmtSql`, write it in `replaceMatchEventsTx`, and read it in `GetMatchInstants` and the cup/friendly instant readers (NULL → nil map). Verify with an integration test that saves and reads back one instant per kind.

## 3. Rating formula and storage

- [x] 3.1 Rewrite `common/matchratings.go` around `ComputePlayerRatings(homeId, awayId, homeGoals, awayGoals, instants, events, upToMinute)` and `PlayerRating` (design §2). Remove `formRatingDelta`, `playerForm` and `matchRatingJitter`. Verify with `matchratings_test.go` cases mirroring every match-ratings spec scenario (substitution at 60, sent off at 35, striker 9.6, late sub 5.7, clamp 10.0, zone +0.45, cap +1.5, s.c. under 10 minutes and its as-of-minute threshold).
- [x] 3.2 Cover matches without contributions inside `ComputePlayerRatings` (design §2): starting XI ∪ incoming substitutes, minutes from substitution, red-card and injury minutes, zone 0. Verify with tests for the two "simulated before stored ratings" scenarios.
- [x] 3.3 Add `MatchResult.PlayerRatings` and the `match_player_ratings` statements (`InsertMatchPlayerRatingStmtSql`, `DeleteMatchPlayerRatingsStmtSql`, `GetMatchPlayerRatingsStmtSql`, `GetMatchPlayerRatings`). Delete and reinsert them in `replaceMatchEventsTx` with the fixture's kind. Verify with an integration test per kind: a save stores the rows, a retried save replaces them, and a save rejected with `ErrAlreadyPlayed` stores none.
- [x] 3.4 In the league, cup and friendly simulate paths (`engine/simulation.go`, the cup and friendly simulators), compute `PlayerRatings` at the last minute and pass them into the result. Verify with `simulation_test.go`: a simulated match yields one rating per player listed in any instant or rating event, and recomputing from the saved instants and events gives the same numbers.

## 4. Read paths and progression

- [x] 4.1 Add `PlayerRatings` to `WeekMatch`, `CupTieDetails` and `FriendlyMatch`. In `WeekMatchById`, `GetCupTieDetails` and `EnrichPlayedFriendly`, load stored rows, otherwise use the legacy fallback. Derive `Ratings` from them. Copy the starting lineups before substitutions are replayed, and resolve players for every rated id (design §4). Verify with `match_viewer_payload_integration_test.go` cases: stored ratings returned as-is; an old match returns fallback ratings including a substituted-off scorer.
- [x] 4.2 `common/playermatches.go`: read the player's rating from the same source. Verify with an existing or new test showing that a substituted-off player's match history has a rating.
- [x] 4.3 Add `PostMatchPayload.Ratings`, filled from `PlayerRatings`. `engine.matchRatingsForProgression` uses it and falls back to computing them from the payload's lineups and events for older payloads. Verify with `progression_test.go`: a substitute rated 7.0 gets Form +4, and a payload without ratings still drifts Form.

## 5. Frontend

- [x] 5.1 `api/types.ts`: add `PlayerRating`, `PlayerContribution`, `MatchInstant.playerContributions` and `playerRatings` on the three match payloads. Map them in `features/match/matchViewModel.ts`. Verify with `npx tsc --noEmit`.
- [x] 5.2 Rewrite `utils/liveMatchRatings.ts` as the port of `ComputePlayerRatings` (design §6). Add `backend/common/testdata/match_ratings_parity.json` and assert it from both `matchratings_test.go` and `utils/liveMatchRatings.test.ts`. Verify both suites pass on the same fixture.
- [x] 5.3 `MatchMinuteRatings` / `MatchRatingsTab` / `MatchViewer` live view: pass instants and stored `playerRatings`, show the stored records at the final minute, and keep feeding live events through `withholdSpoilerEvents`. Verify with `MatchViewer.test.tsx` cases for a live red card not affecting the live rating, and the final minute equal to the stored rating.
- [x] 5.4 `TeamRatingsColumn`: one row per `PlayerRating` (starters in lineup order, then substitutes by `cameOn`), with position abbreviation, minutes with ↑/↓, "own vs rival" (hidden when absent) and the rating chip with the breakdown tooltip or popover. Verify with a component test for the "Starter replaced at 60" and "Scorer in a win" breakdown scenarios, and visually at 360 px.
- [x] 5.5 `MatchPitchChip` / `Lineups`: confirm they still read `ratings` by id and show a substituted-off player's rating where he is drawn. Verify in the running app.
- [x] 5.6 Add the `match.ratings.*` keys from design §6 to `frontend/src/i18n/locales/en.json` and `es.json` (es: "Base", "Zona", "Goles", "Tarjetas", "Lesión", "Resultado", "Entró", "Salió", "Aporte"). Verify there are no missing-key warnings in the console.

## 6. Specs and docs

- [x] 6.1 Code maps: update the `match-viewer` Code map (ratings components) and the `player-development` Code map (ratings come from the post-match payload, not `common.ComputeMatchRatings`). The new `match-ratings` spec gets its Code map at archive time (see Workflow follow-up). Verify by reading the files.
- [x] 6.2 Update `ARCHITECTURE.md` if it lists the instants tables or match payload fields. Verify with `grep -n "match_instants\|ComputeMatchRatings" ARCHITECTURE.md backend/README.md`.
- [x] 6.3 Run `openspec validate fidelity-match-ratings --strict` and verify it passes.

## 7. Final checks

- [ ] 7.1 In the running app, simulate a league match, a cup tie and a friendly. For each, check that the live ratings move minute to minute without revealing a red card, that the finished "Notas" tab lists substitutes with ↑/↓ minutes and contributions, that the breakdown adds up to the rating, and that scrubbing to the final minute shows the stored ratings. Verify that an old match still shows fallback ratings including substituted-off players.
- [x] 7.2 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`. Verify all pass.
- [x] 7.3 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`. Verify all pass.

## Workflow follow-up

- Archive with `/opsx:archive` once merged, which creates `openspec/specs/match-ratings/spec.md`.
- Then append its Code map: formula `backend/common/matchratings.go` (`ComputePlayerRatings`,
  `PlayerRating`, `RatingsById`, `MatchFinalMinute`); contributions
  `backend/engine/initial_simulation.go` (`playerContributions`, `newSideDuelists`); storage and
  read paths `backend/common/matchresult.go` (`replaceMatchPlayerRatingsTx`,
  `GetMatchPlayerRatings`, `LoadPlayerRatings`, `addRatedPlayers`) and migration
  `init89_match_player_ratings.sql`; engine `engine/postmatch.go` (`newMatchResult`); frontend
  port `frontend/src/utils/liveMatchRatings.ts`; parity fixture
  `backend/common/testdata/match_ratings_parity.json`.
