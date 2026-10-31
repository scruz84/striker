# Tasks

## 1. Schema and constants

- [x] 1.1 Add `backend/resources/sql/init92_user_account_lifecycle.sql` as in design D1, plus `teams.reset_reason` (only feeds the `striker.teams.reset` metric's `reason` attribute) (re-runnable: `IF NOT EXISTS`, guarded `UPDATE`)
- [x] 1.2 Add `common/accounts.go` constants (`InactivityWarningDays`, `InactivityReleaseDays`, `AccountErasureWarningDays`, `AccountErasureDays`, reset guard windows, `CurrentTermsVersion`, `DefaultTeamBudget = 400000`)
- [x] 1.3 Add `TRANSACTION_TEAM_RESET = 16` and `NOTIFICATION_ACCOUNT = 13` to `common/fmconstants.go`; add the reason label to the finances page and `finances.reasons.*` in en/es
- [x] 1.4 Extend `EnsureSchema`/schema tests so `init92` is applied twice without error

## 2. Activity tracking and deleted-user sessions

- [x] 2.1 Activity middleware (design D2) on every authenticated route group, with the 1 h per-replica throttle and the deleted-id map; write `last_seen_at` on login too
- [x] 2.2 Answer 401 and clear the cookie when the activity update hits no row
- [x] 2.3 Lock and re-check the user row inside `AssignTeam`'s transaction so a deleted user's session cannot claim a club
- [x] 2.4 Tests: throttle (50 requests → 1 write), stale session after deletion → 401 on read and on `POST /v1/auth/onboarding/country`

## 3. Release and reset

- [x] 3.1 `common.ReleaseTeam(tx, teamId)` (design D3); add `AND t.reset_pending_since IS NULL` to `GetFreeTeamStmtSql` and `BackfillAiLevelsStmtSql`
- [x] 3.2 `common.ResetReleasedTeam` (design D5 steps 1–12) with named `...StmtSql` constants for every new query
- [x] 3.3 `ReturnLoan`: honour `player_loans.release_on_return` (release to free agency instead of returning)
- [x] 3.4 `TeamsResetCounter` OTel counter with a `reason` attribute (inactivity, self_deletion, admin_deletion, erasure)
- [x] 3.5 Tests: squad → free agents with `last_team_id`, loaned-in returns, loaned-out released at loan end, academy removed, offers/listings/challenges/trainer cleared, scheduled friendly kept, settings and fanbase/socios at defaults, ledger row with the right delta, history tables untouched, AI level drawn, reset skipped while a match is inside the guard window, pending club not claimable

## 4. Engine lifecycle job

- [x] 4.1 `engine/accountlifecycle.go`: daily pass (warnings with single-statement stage claims, releases with lock + re-read, erasure warnings, erasures) behind `ClaimEngineJobSlot("account-lifecycle-daily", now, 24h)`
- [x] 4.2 15-minute reset pass over `reset_pending_since IS NOT NULL` clubs
- [x] 4.3 Wire both triggers in `engine/scheduler.go` / `StartEngine` (daily 03:17 UTC, every 15 min)
- [x] 4.4 Notification messages with keys `notifications.account.releaseWarning` (params `days`, `team`), `notifications.account.erasureWarning` (param `days`); Spanish templates in `common/notificationtranslations.go`; en/es keys in the locale files
- [x] 4.5 Concurrent integration test: two replicas run the daily pass and the reset pass at the same instant → one release, one warning per stage, one reset per club

## 5. Account deletion and export

- [x] 5.1 `common.DeleteAccount` (design D6) including group ownership handover and empty-group deletion
- [x] 5.2 `GET /v1/user/account` (profile + inactivity rule data), `DELETE /v1/user/account` with confirmation check (400 on mismatch, 204 + cleared cookie on success)
- [x] 5.3 `DELETE /v1/admin/users/:id` (403 on own id) and a delete button with confirmation on the admin users list; show `lastSeenAt` there
- [x] 5.4 `common.ExportUserData` + `GET /v1/user/account/export` as an attachment (design D7)
- [x] 5.5 Tests: deletion erases the user and cascaded rows, club pending reset, group handover by earliest `joined_at`, re-login creates a new user; export contains email and notifications but no push keys

## 6. Terms acceptance and released-club notice (backend)

- [x] 6.1 `LoggedUser` gains `termsAccepted`, `termsVersion`, `releasedTeamName`; `GetUserStmtSql` reads the new columns
- [x] 6.2 `POST /v1/auth/accept-terms` (version must be current, `ageConfirmed` true)
- [x] 6.3 `SelectCountryHandler` returns 400 without accepted terms and clears `released_team_name` after assigning the club

## 7. Owner identity

- [x] 7.1 Drop `ownerName` from `GetTeamStmtByIdSql`, `common.Team` and `frontend/src/api/types.ts`; test that a team response has no owner name
- [x] 7.2 Group page note that members see each other's names

## 8. Frontend

- [x] 8.1 `features/account/AccountPage.tsx` at `/account`: profile, the 60/365-day rules, "Download my data", delete dialog requiring the club name (or DELETE/BORRAR); "Account" entry in the header menu; hooks in `api/hooks.ts`, types in `api/types.ts`
- [x] 8.2 Legal properties (design D9): `REPLACE_LEGAL_*` placeholders in `frontend/public/config.js`, escaped substitution in `frontend/set-env.sh`, `getLegalConfig()` and `VITE_LEGAL_*` fallbacks in `src/config.ts`, the env vars in `frontend/docker-compose.yml`
- [x] 8.3 `features/legal/PrivacyPage.tsx` and `TermsPage.tsx` at `/privacy` and `/terms`, public, with version and date, filled from `getLegalConfig()`, "not configured" alert for missing required values
- [x] 8.4 `LegalNoticePage.tsx` at `/legal-notice` and its links, only when `LEGAL_NOTICE_ENABLED=true`
- [x] 8.5 `AcceptTermsPage.tsx` at `/accept-terms` and the `Layout` gate branch (before the onboarding branch)
- [x] 8.6 Onboarding: age + terms checkbox for first-time users, "your club X was released" notice when `releasedTeamName` is set (Layout sends a user who has not accepted to `/accept-terms` first; the onboarding checkbox is the backstop for anyone who still gets there)
- [x] 8.7 Links to Privacy/Terms (and Legal notice when enabled) on the login page, help page and navigation drawer
- [x] 8.8 Draft the Privacy Policy and Terms texts (en/es) covering every point of the `privacy-and-legal` spec, marked as drafts for the owner to review
- [x] 8.9 All new strings in `frontend/src/i18n/locales/en.json` and `es.json`
- [x] 8.10 Vitest: delete dialog enables only on exact confirmation, gate order (terms before onboarding), legal pages render configured values, the not-configured alert and hide the legal notice when disabled

## 9. Specs and docs

- [x] 9.1 Code map sections for the new `user-accounts` and `privacy-and-legal` specs; update `cpu-team-management` Code map (`ResetReleasedTeam`, pending-reset filter in `BackfillAiLevels`)
- [x] 9.2 `ARCHITECTURE.md` Auth section (activity tracking, account deletion, terms gate) and `backend/README.md` (new endpoints, env vars, lifecycle job, `init92`)
- [x] 9.3 Add the `LEGAL_*` properties (required vs optional, when to enable the legal notice) to the `HOWTO_RUN.md` production checklist created by `harden-public-deployment`
- [x] 9.4 Add the new engine job to `openspec/specs/engine-replicas` job list

## 10. Verification

- [x] 10.1 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`
- [x] 10.2 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`

## Workflow follow-up

- Implement after `harden-public-deployment` has shipped.
- Archive the change with `/opsx:archive` once implemented and merged.
