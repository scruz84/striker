# Tasks

## 1. Data model and persistence

- [ ] 1.1 Add `SET_PIECE_NONE`, `SET_PIECE_PENALTY` and `SET_PIECE_DIRECT_FREE_KICK` (0/1/2) to `backend/common/fmconstants.go` next to the event constants, and `SetPiece int` (`json:"setPiece"`) to `common.Event` in `common/fmstructs.go`; verify with `go build ./...`
- [ ] 1.2 Create `backend/resources/sql/init82_match_event_set_piece.sql` adding `set_piece smallint NOT NULL DEFAULT 0` with `ADD COLUMN IF NOT EXISTS` to `match_events`, `cup_tie_events` and `friendly_events`, plus a header comment; verify that applying it twice on a local Postgres succeeds both times
- [ ] 1.3 Add `set_piece` to `InsertMatchEventStmtSql` (`common/database.go`) and its `Exec` in `engine/database.go` `UpdatePlayedMatch`, to `InsertCupTieEventStmtSql` (`common/cups.go`) and to `InsertFriendlyEventStmtSql` (`common/friendlies.go`), passing `e.SetPiece`; verify `go build ./... && go vet ./...`
- [ ] 1.4 Select and scan `set_piece` in `GetWeekMatchesEventsByIdStmtSql`, `GetCupTieEventsStmtSql` and `GetFriendlyEventsStmtSql`, and set `Event.SetPiece`; verify with a round-trip integration test (insert an event with `SetPiece = 1`, read it back) for at least the league table, following the existing `*_integration_test.go` pattern

## 2. Simulation engine

- [ ] 2.1 Add the set-piece constants from design.md §4 to `engine/scorer_simulation.go` with doc comments citing the real-world figures (penalties 0.24–0.41 per match, ~78% conversion; direct free kicks ~0.85 shots per match, ~5–6% conversion)
- [ ] 2.2 Implement `classifyChance(r *rand.Rand) int` and unit-test it over 1,000,000 seeded rolls: penalty share 2.2% ± 0.1 and free-kick share 6.5% ± 0.15
- [ ] 2.3 Implement `bestOnPitchOutfielder(players, byPosition, score)`, which excludes the goalkeeper, breaks ties by lowest id and returns -1 when the side is empty; unit-test the highest-Shot pick, the tie-break, the skipping of sent-off players removed from `byPosition`, and the -1 case
- [ ] 2.4 Implement `penaltyConversionProbability(shot, goalKeeper)` and `freeKickConversionProbability(value, goalKeeper)`; unit-test every numeric scenario in the spec delta (78%, 92% clamp, 60% clamp, 6%, ≈9.3%, 15% / 2% clamps)
- [ ] 2.5 Wire them into `processChance`: classify first; for a set piece pick the taker, roll the conversion, update the score and append a `GOAL`/`NO_GOAL` event with `SetPiece`, `OtherTeamPlayers = [keeper]` and the keeper's name; leave the open-play path byte-for-byte unchanged; verify with existing `scorer_simulation_test.go` plus a test that a discarded set piece (no outfielders) appends no event
- [ ] 2.6 Add a calibration test that simulates 200,000 even-matchup chances with pure functions (no DB): set-piece-adjusted goals per chance stays within ±0.004 of open-play-only (≈ ±0.05 goals per match at 12.6 chances) and penalties per 12.6 chances are 0.28 ± 0.02; verify `go test ./engine -run SetPiece`
- [ ] 2.7 Update `openspec/specs/match-simulation/spec.md` Code map (`classifyChance`, `bestOnPitchOutfielder`, the conversion functions, `init82_match_event_set_piece.sql`); verify `openspec validate add-penalties-and-free-kicks --strict`

## 3. Frontend

- [ ] 3.1 Add `setPiece?: number` to `MatchEvent` and export `SET_PIECE_PENALTY` / `SET_PIECE_DIRECT_FREE_KICK` in `frontend/src/api/types.ts`; verify `npx tsc --noEmit`
- [ ] 3.2 Add `match.commentary.penaltyGoal`, `penaltyMiss`, `freeKickGoal` and `freeKickMiss` (3–4 variants each, with `{{player}}`, `{{team}}` and `{{keeper}}`) and `match.setPiece.penaltyShort` / `freeKickShort` to `frontend/src/i18n/locales/en.json` and `es.json` (`"pen."`/`"f.k."`, `"p."`/`"f.d."`); verify both files hold the same new keys (`jq '.match.commentary | keys'` on each matches) and that `npx vitest run` passes
- [ ] 3.3 In `utils/matchCommentary.tsx`, pick the commentary key by `(eventType, setPiece)` and add a `keeper` var linked to `otherTeamPlayers[0]`; add tests that a penalty goal and a missed free kick render their own phrasing and that a missing `setPiece` falls back to open play
- [ ] 3.4 Change `goalScorers` in `utils/matchGoals.ts` to return `goals: { minute, setPiece }[]` per scorer, update `MatchScoreboard.tsx`, `MatchDetailsPage.tsx`, `CupTieDetailsPage.tsx` and `FriendlyMatchPage.tsx` to render `16' (pen.)`, and update `matchGoals.test.ts` with a mixed open-play/penalty/free-kick case
- [ ] 3.5 In `features/match/MatchEventsReport.tsx`, always show a missed penalty, even when full commentary is off, and keep missed free kicks hidden like other `NO_GOAL` events; add a test for both

## 4. Integration checks

- [ ] 4.1 Run a local league round (see HOWTO_RUN.md) and check the live replay and the finished match report: a set-piece goal shows its suffix and commentary, and top scorers and player details count it as a goal
- [ ] 4.2 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`
- [ ] 4.3 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`

## Workflow follow-up

- Archive the change with `/opsx:archive` once merged, so the match-simulation spec absorbs the delta.
