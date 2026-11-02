# Tasks

## 1. Backend: storage

- [x] 1.1 Add `backend/resources/sql/init94_default_friendly_lineup_substitutions.sql` with the `default_friendly_lineup_substitutions` table from design D1 (no data copy); verify it applies twice on a fresh test DB.
- [x] 1.2 Add the three `...DefaultFriendlyLineupSubstitution...StmtSql` constants, `GetDefaultFriendlyLineupSubstitutions` and `SetDefaultFriendlyLineupSubstitutions` (advisory lock `(teamId, 2)`) to `backend/common/lineupsubstitutions.go` per design D2. Verify with an integration test: set 2 friendly subs → read back 2 in save order; league/cup set untouched; set `nil` → empty.
- [x] 1.3 Switch `common.GetDefaultFriendlyLineup` (`common/database.go`) to `GetDefaultFriendlyLineupSubstitutions` and update its doc comment (no longer "team-wide, shared"). Verify a team with league/cup subs and no friendly subs reads an empty friendly set ("After the migration" scenario).

## 2. Backend: API

- [x] 2.1 `SetFriendlyDefaultLineupHandler` (`handlers/lineupHandlers.go`): validate `req.Substitutions` with `validateSubstitutions` against `common.GetUnavailablePlayerIdsForFriendly` before any write, then call `SetDefaultFriendlyLineupSubstitutions` after `SetDefaultFriendlyLineup` (design D3); update its doc comment. Extend `handlers/lineupHandlers_test.go`: friendly save with 1 sub leaves the league/cup set unchanged and vice versa; suspended incoming player accepted; injured incoming player and 4 subs rejected with 400 and nothing stored; empty list clears.
- [x] 2.2 Verify `GET /v1/user/team/lineup/friendly` and the per-friendly `GetFriendlyLineupHandler` fallback return the friendly set, not the league/cup set ("Opening a friendly without an override" scenario).

## 3. Backend: simulation and cleanup

- [x] 3.1 `engine.simulateFriendly` (`engine/friendlies.go`): take each team's substitutions from `GetLineupsForFriendly`'s lineup instead of `GetFriendlyLineupSubstitutions` (design D4); update the comment there and the friendly paragraph of `runMatchSimulation`'s doc comment in `engine/simulation.go`. Integration test covering the "Substitutions a match falls back to" scenarios: friendly on the friendly default plays its 2 subs and the snapshot stores them; override plays only its own; best-XI fallback plays none; a league match without override plays the league/cup set.
- [x] 3.2 Add `DeleteTeamDefaultFriendlySubstitutionsStmtSql` to `common/accounts.go` and run it in the club reset next to `DeleteTeamDefaultSubstitutionsStmtSql`; extend the reset integration test to assert the table is empty for the reset club.
- [x] 3.3 Add `"default_friendly_lineup_substitutions"` to `teamPlayerDependentTables` in `handlers/initializer.go` (before `"players"`).

## 4. Frontend

- [x] 4.1 `api/hooks.ts`: `useSetFriendlyDefaultLineup` takes and PUTs `substitutions`.
- [x] 4.2 `features/lineup/LineupEditorPage.tsx` `FriendlyDefaultLineupForm`: pass `substitutionsEnabled` and `isFriendly`, send `substitutions` from `onSave`; update the component comments. Fix the `substitutionsEnabled` comment in `MatchLineupForm.tsx`. Add a test that the Friendly tab renders the substitutions planner and saves its substitutions.
- [x] 4.3 Locales `frontend/src/i18n/locales/en.json` and `es.json`: rewrite `lineup.friendlyHint` per design D6 (no "shared" sentence).

## 5. Specs and docs

- [x] 5.1 Sync the new `team-lineups` spec into `openspec/specs/team-lineups/spec.md` with a Code map section naming `common/lineupsubstitutions.go`, `common/database.go` (`GetDefaultLineup`, `GetDefaultFriendlyLineup`), `common/friendlies.go` (`GetLineupsForFriendly`), `common/matchlineupsubstitutions.go` / `common/cuptielineupsubstitutions.go` (fallbacks), `handlers/lineupHandlers.go`, `engine/friendlies.go` (`simulateFriendly`), `features/lineup/LineupEditorPage.tsx` and the migration.
- [x] 5.2 Update the `match-simulation` spec's Code map line about standing substitutions if it names `GetDefaultLineupSubstitutions` as the only source.

## 6. Verification

- [x] 6.1 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`
- [x] 6.2 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`
