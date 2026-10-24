# Tasks

## 1. Persistence: instants and flavor for cup ties and friendlies

- [x] 1.1 Create `backend/resources/sql/init83_match_viewer_parity.sql` with a header comment:
  - `CREATE TABLE IF NOT EXISTS cup_tie_instants`, with the same columns as `friendly_instants`, keyed `(tie_id, minute)` and referencing `cup_ties(tie_id)`, plus an index on `tie_id`.
  - `ADD COLUMN IF NOT EXISTS weather varchar(20)`, `attendance int`, `referee_name` / `assistant1_name` / `assistant2_name varchar(120)` on both `cup_ties` and `friendly_matches`.

  Verify that applying it twice on a local Postgres succeeds both times.
- [x] 1.2 In `common/cups.go`:
  - Add `InsertCupTieInstantStmtSql` and `GetCupTieInstantsStmtSql`, plus `GetCupTieInstants`.
  - Extend `SetCupTiePlayed` with an `instants []MatchInstant` parameter that inserts them the way `SetFriendlyPlayed` does.

  Verify with `go build ./... && go vet ./...`.
- [x] 1.3 In `common/matchflavor.go`, add `UpdateCupTieFlavorStmtSql` / `UpdateCupTieFlavor` and `UpdateFriendlyFlavorStmtSql` / `UpdateFriendlyFlavor`. Verify with `go build ./...`.
- [x] 1.4 Generalize `engine.updateMatchReportFlavor` to take a store function so league, cup and friendly share the referee-name drawing. Keep failures log-only, so the result stands when a name can't be drawn. Verify that the existing league flavor path behaves the same (`go test ./engine`).
- [x] 1.5 In `engine/cups.go` `simulateCupTie`:
  - Draw the weather once and pass it to `UpdateTeamFinancesAfterMatch`.
  - Keep the returned attendance and store the flavor.
  - Pass `*matchSimulation.MatchInstants` to `SetCupTiePlayed`.

  Verify with a test that a simulated tie stores one instant per simulated minute and a non-empty weather and attendance (extend `engine/cups_test.go` or an integration test).
- [x] 1.6 In `engine/stadium.go`, change `creditFriendlyTicketRevenue` to take `weather` as a parameter. In `engine/friendlies.go` `simulateFriendly`, draw the weather once, pass it in, and store the flavor with the returned attendance. Verify with a test that the stored attendance equals the attendance in the friendly's ticket-revenue description.

## 2. Read side: detail payloads

- [x] 2.1 In `common/cups.go`:
  - Add `MatchInstants`, `HomeStadiumName`, `Weather`, `Attendance`, `RefereeName`, `Assistant1Name` and `Assistant2Name` (`omitempty`) to `CupTieDetails`.
  - Fill them in `GetCupTieDetails`. Instants and flavor are filled only when the tie is played. The stadium name comes from the home team's `stadium_name`.

  Verify with an integration test that reads a played tie back with its instants and flavor.
- [x] 2.2 In `common/friendlies.go`:
  - Add `HomeStadiumName`, the five flavor fields, `WorldCupGroupName` and `WorldCupRound` (`omitempty`) to `FriendlyMatch`.
  - Extend `GetFriendlyMatchByIdStmtSql` with the flavor columns, the home stadium, and a `LEFT JOIN` on `world_cup_group_matches` / `world_cup_groups`.
  - Keep `EnrichPlayedFriendly` gating events and instants on `played`.

  Verify with an integration test covering a linked World Cup group friendly and an ordinary one (the group fields are empty for the ordinary one).
- [x] 2.3 Mirror the new fields in `frontend/src/api/types.ts` (`CupTieDetails`, `FriendlyMatch`). Verify with `npx tsc --noEmit`.

## 3. Frontend: the shared viewer

- [x] 3.1 Create `features/match/matchViewModel.ts` with the `MatchView` type and the pure adapters `fromWeekMatch`, `fromCupTie` and `fromFriendly`. Move `weeksInASeason` into it, and replace `cupTieToWeekMatch` and `friendlyToWeekMatch`. Unit-test each adapter's `subtitleLabel`:
  - "Jornada 3 de 38"
  - "Copa del Rey · Cuartos de final"
  - "Mundial · Grupo B · Jornada 2"
  - "Amistoso"
- [x] 3.2 Change `utils/matchSubtitle.ts` to `matchSubtitle(label, matchDate, weather)`, which drops the parts that are missing. Verify with a unit test for "label · date · emoji" and for "label · date" without weather.
- [x] 3.3 Create `features/match/MatchViewer.tsx`:
  - Move the phase switch, the live-completion effect, and the scheduled, live and finished views verbatim from `MatchDetailsPage.tsx`, keeping their comments. Use `MatchView` throughout.
  - The live scoreboard shows scorers; the stadium entry and flavor lines appear whenever their data exists.
  - Accept a `lineupSection` that is rendered only in the scheduled view, and a `belowScoreboard` slot for the penalty line.

  Verify with `npx tsc --noEmit`.
- [x] 3.4 In the live view, pass `goalkeeperRating(live.homeLineup, live.homePlayers)` and the away equivalent to `MatchMinuteZoneComparison` once the lineups have arrived. Verify with a test that the live and finished panels give the same numbers for the same instant and lineups.
- [x] 3.5 Reduce `MatchDetailsPage.tsx` to an adapter that fetches, maps with `fromWeekMatch`, and builds the lineup section only for a participant. Keep exporting `positionOf`, `Lineups` and `goalkeeperRating`. Verify that `MatchDetailsPage.test.ts` still passes.
- [x] 3.6 Reduce `CupTieDetailsPage.tsx` to an adapter:
  - Use `useCupDetails(tie.cupId)` for the round label.
  - Keep the `ByeTie` card, and render the penalty-shootout line through `belowScoreboard`.
  - Build the lineup section only for a participant.

  Verify that a played tie shows the zone duel when it has instants and only the pitch without them.
- [x] 3.7 Reduce `FriendlyMatchPage.tsx` to an adapter whose lineup section is built only when `userTeam` plays the match, which fixes the editor being offered to any owner. Verify manually or in a test that a non-participant sees no editor and no locked notice.

## 4. Frontend: zone duel strength figures

- [x] 4.1 In `features/match/ZoneComparisonRows.tsx` `DuelCell`, render each half as two lines: the share (`body2`, bold when stronger) and the rounded raw strength (`caption`, 12 px, opacity 0.85). Make the tooltip always show one decimal. Update the component comment that explains why raw values were removed. Verify with a unit test of the formatting helper: `42.6 → "43"` and tooltip `"42.6 - 28.4"`. Both sides at 0 give 50% / 50% with "0".
- [x] 4.2 Check the panel at 360 px wide with the Playwright mobile smoke setup (`frontend/e2e`): no horizontal scroll, and every number in the duel at least 12 px. Verify with a screenshot attached to the PR.

## 5. i18n and specs

- [x] 5.1 Add `match.competition.friendly` ("Friendly" / "Amistoso") and `match.competition.worldCupGroup` ("World Cup · Group {{group}} · Matchday {{round}}" / "Mundial · Grupo {{group}} · Jornada {{round}}"), and reword `match.positionBattle.help` to explain both numbers, in `frontend/src/i18n/locales/en.json` and `es.json`. Verify that both files hold the same new keys.
- [x] 5.2 Update the Code map of `openspec/specs/match-simulation/spec.md`: `SetCupTiePlayed` instants, `UpdateCupTieFlavor` / `UpdateFriendlyFlavor`, `cup_tie_instants`, `init83_match_viewer_parity.sql`. Update the Code map of `socios` (the `creditFriendlyTicketRevenue` weather parameter) and of `world-cup` (`MatchViewer`, the group label join). Verify with `openspec validate unify-match-viewer --strict`.

## 6. Verification

- [x] 6.1 Run a local league round, cup round and friendly (see HOWTO_RUN.md). For each one, check the scheduled, live and finished views:
  - the same sections in the same order;
  - the competition subtitle;
  - the flavor lines;
  - live scorers;
  - the zone duel with percentage and strength, with the same figures at minute 90 live and finished.
- [x] 6.2 Backend: `go build ./... && go vet ./... && go test ./...` in `backend/`.
- [x] 6.3 Frontend: `npx tsc --noEmit && npx oxlint && npx vitest run` in `frontend/`.

## Workflow follow-up

- Archive with `/opsx:archive unify-match-viewer` after merge, which creates `openspec/specs/match-viewer/spec.md`. Then add its `## Code map`: `features/match/MatchViewer.tsx`, `matchViewModel.ts`, the three route adapters (`MatchDetailsPage.tsx`, `CupTieDetailsPage.tsx`, `FriendlyMatchPage.tsx`), `ZoneComparisonRows.tsx`, `MatchMinuteZoneComparison.tsx` and `utils/matchSubtitle.ts`.
