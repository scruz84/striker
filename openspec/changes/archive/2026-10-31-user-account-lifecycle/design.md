# Design: user-account-lifecycle

## Context

See proposal.md for the motivation. What exists today and shapes the approach:

- `users` (`name`, `provider`, `email`, `country`, `is_admin`, `language`) has no timestamps.
  Child tables `notifications`, `notification_preferences`, `push_subscriptions`, `groups`
  (owner) and `group_members` reference it `ON DELETE CASCADE`. `teams.owner_id` has **no** FK,
  so deleting a user leaves a dangling owner unless it is cleared first.
- The session is a `gin-jwt` HTTP-only cookie (`SameSite=None`, `Secure`, 24 h timeout, 1 h max
  refresh) carrying `userId` and the Google picture URL. There is no server-side session store,
  so a deleted user's token keeps validating until it expires.
- `common.AssignTeam` already does half of a reset when a human claims a club: it runs
  `UnAssignPlayersStmtSql` (old squad to free agency), generates a new squad with
  `doCreatePlayers`, and resets columns with `UpdateTeamDetailsStmtSql` — but it writes `budget =
  DEFAULT` without a ledger row and does not reset `fanbase`, `socios` or `friendly_formation`.
- CPU routines act only on `owner_id IS NULL AND ai_level >= 1`; `BackfillAiLevels` (service
  startup) gives a level to every ownerless `ai_level = 0` league team.
- `GetFreeTeamStmtSql` hands out any `owner_id IS NULL` club of the chosen country.
- A loaned player has `players.team_id` = the borrower; `player_loans` keeps `from_team_id` and
  `common.ReturnLoan` moves him back at season rollover.
- `DefaultNotifier` is `LogNotifier`: email is log-only, Web Push is real when VAPID is set.
- This change assumes `harden-public-deployment` has shipped: identity comes only from verified
  JWT claims, every state-changing request carries `X-Striker-Client`, and `HOWTO_RUN.md` has a
  production checklist.
- The frontend's runtime configuration is `public/config.js` (`window.extended.*` placeholders
  rewritten by `frontend/set-env.sh` from env vars at container start) read through
  `src/config.ts`'s `runtimeValue`.

## Goals / Non-Goals

**Goals:**
- One reset routine used by every way a club loses its owner (inactivity, self-deletion, admin
  deletion, long-idle erasure), safe with several engine replicas and with matches in flight.
- Deleting an account removes every column that identifies the person in one transaction.
- Every piece of legal identity is a deployment property, so the same image serves any operator.

**Non-Goals:**
- Changing what `AssignTeam` does when a human claims a club (it keeps regenerating the squad).
- A server-side session store or token revocation list.
- Pseudonymising data the game keeps about clubs (team names, ledger, match history are game
  data, not personal data, once the owner link is gone).

## Decisions

### D1. Schema: `resources/sql/init92_user_account_lifecycle.sql`
(`init91_*` is taken by the generated name-pool files.) Re-runnable:

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen_at timestamptz;
UPDATE users SET last_seen_at = now() WHERE last_seen_at IS NULL;   -- grace period on upgrade
ALTER TABLE users ALTER COLUMN last_seen_at SET DEFAULT now();
ALTER TABLE users ALTER COLUMN last_seen_at SET NOT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS inactivity_warning_stage smallint NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS terms_version int;            -- NULL = never accepted
ALTER TABLE users ADD COLUMN IF NOT EXISTS terms_accepted_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS released_team_name varchar(200); -- shown once at onboarding
ALTER TABLE teams ADD COLUMN IF NOT EXISTS reset_pending_since timestamptz;
ALTER TABLE player_loans ADD COLUMN IF NOT EXISTS release_on_return boolean NOT NULL DEFAULT false;
CREATE INDEX IF NOT EXISTS users_last_seen_idx ON users(last_seen_at);
CREATE INDEX IF NOT EXISTS teams_reset_pending_idx ON teams(reset_pending_since)
  WHERE reset_pending_since IS NOT NULL;
```

`inactivity_warning_stage`: 0 none, 1 = 14-day club warning sent, 2 = 7-day club warning sent,
3 = 14-day erasure warning sent. Any activity write sets it back to 0.

### D2. Activity + existence middleware (`service/fmservice.go`, `handlers/activity.go`)
A Gin middleware registered right after `authMiddleware.MiddlewareFunc()` on every
authenticated group. Per replica it keeps a `sync.Map[userId]lastWrite`; when the entry is
older than 1 h (or missing) it runs

```sql
UPDATE users SET last_seen_at = now(), inactivity_warning_stage = 0 WHERE user_id = $1
```

and, if it affected 0 rows, the user no longer exists: it clears the cookie and aborts with
401. Between writes, a second per-replica `sync.Map` of deleted ids (filled by that 0-row
result and by the deletion code on the same replica) answers 401 immediately. Login also
writes `last_seen_at`. *Alternative considered*: updating only in `/auth/me` — rejected, a tab
left open for days never re-fetches `me`. *Alternative*: checking existence on every request —
an extra query per request for an event that happens a handful of times.

`SelectCountryHandler` (the one path that writes `teams.owner_id`) additionally checks the user
row exists `FOR UPDATE` inside `AssignTeam`'s transaction, so a stale session racing the
deletion cannot leave a club owned by a vanished id.

### D3. Releasing vs. resetting are two steps
`common.ReleaseTeam(tx, teamId)` only does
`UPDATE teams SET owner_id = NULL, ai_level = 0, reset_pending_since = now() WHERE team_id = $1`.
It is cheap, runs inside the caller's transaction (deletion in the API, inactivity in the
engine), and needs no match-timing logic. The heavy work happens later in
`common.ResetReleasedTeam` (D5), run by the engine. While pending, `ai_level = 0` keeps every
CPU routine away and two queries get `AND t.reset_pending_since IS NULL`:
`GetFreeTeamStmtSql` (not claimable) and `BackfillAiLevelsStmtSql` (not given a level early).
*Alternative*: reset synchronously inside the deletion request — rejected: a deletion right
before kickoff would swap the squad under a simulation in progress, and the API process would
duplicate engine logic.

### D4. Engine job `AccountLifecycleJob` (`engine/accountlifecycle.go`)
Two quartz triggers in `engine/scheduler.go`:

- **Daily at 03:17 UTC** — `common.ClaimEngineJobSlot(dbCtx, "account-lifecycle-daily", now,
  24h)`; only the winner runs. Then, in order:
  1. Club warnings: select owners with `last_seen_at <= now() - 46 days` and stage 0, and with
     `<= now() - 53 days` and stage < 2. For each, a single-statement claim
     `UPDATE users SET inactivity_warning_stage = $2 WHERE user_id = $1 AND
     inactivity_warning_stage < $2 AND last_seen_at <= $3 RETURNING user_id` and notify only on
     a returned row (so a re-run never double-sends).
  2. Releases: for each owner with `last_seen_at <= now() - 60 days`, one transaction that locks
     the user `FOR UPDATE`, re-reads `last_seen_at` inside the lock, then calls `ReleaseTeam`
     and stores `users.released_team_name`.
  3. Erasure warnings at 351 days (stage 3, same claim pattern) and erasures at 365 days via
     `common.DeleteAccount` (D6).
- **Every 15 minutes** — reset pass: list `teams WHERE reset_pending_since IS NOT NULL` and run
  `ResetReleasedTeam` on each. No slot claim; per-team safety comes from D5's lock + re-check, so
  several replicas can work through the queue in parallel.

Constants live in `common/accounts.go`: `InactivityWarningDays = []int{46, 53}`,
`InactivityReleaseDays = 60`, `AccountErasureWarningDays = 351`, `AccountErasureDays = 365`,
`ResetMatchGuardBefore = common.LiveBroadcastWindow` (2 h), `ResetMatchGuardAfter = 30m`.

Concurrent integration test (`engine/accountlifecycle_concurrency_test.go`): two goroutines run
the daily pass and the reset pass at once against the same fixtures; assert one release, one
warning per stage and one reset per club (no duplicate squad).

### D5. `common.ResetReleasedTeam(dbCtx, teamId)` — one transaction
1. `SELECT ... FROM teams WHERE team_id = $1 AND reset_pending_since IS NOT NULL AND owner_id IS
   NULL FOR UPDATE` — no row means another replica already did it (or someone claimed it): stop.
2. Match guard inside the lock: any `matches`, `cup_ties` or `friendly_matches` row for the club
   with `in_simulation = true` or `match_date BETWEEN now() - 2h AND now() + 30m` → rollback, try
   again on the next pass.
3. Loaned-in players (`player_loans.to_team_id = T`): reuse `ReturnLoan`'s steps inside this tx
   (move to parent, delete lineup positions, delete loan) and queue the existing loan-return
   notification for after commit.
4. Loaned-out players: `UPDATE player_loans SET release_on_return = true WHERE from_team_id = T`.
   `ReturnLoan` checks the flag and, when set, runs `ReleaseFromTeamStmtSql` on the player
   instead of `movePlayerToTeam`, and clears his `free_agent_seasons`.
5. Academy prospects: `DELETE FROM players WHERE team_id = T AND in_academy = true` (they have no
   match, history or lineup rows; FKs from lineup tables are `ON DELETE SET NULL` anyway).
6. Delete `transfer_listings` and `loan_listings` for the club's players, `transfer_offers` with
   `from_team_id = T OR to_team_id = T`, `friendly_challenges` with either side = T; then
   `UPDATE trainers SET team_id = NULL WHERE team_id = T`.
7. Lineups: delete the club's default lineup, default substitutions and default friendly lineup,
   and its overrides for **unplayed** league matches, cup ties and friendlies (positions and
   planned substitutions). Played-match lineups stay: match pages read them.
8. Release the senior squad:
   `UPDATE players SET last_team_id = team_id, team_id = NULL, contract_years_remaining = 0,
   shirt_number = NULL, free_agent_seasons = 0 WHERE team_id = $1 AND in_academy = false`.
   `player_club_history` already has their stint, so their career page stays correct.
9. New squad with `doCreatePlayers(..., SquadSize, SeniorSkillMin, SeniorSkillMax, SeniorAgeMin,
   SeniorAgeMax, country, false)` — the same call `AssignTeam` makes.
10. Budget: read the current budget, write `budget = 400000` (`DefaultTeamBudget`, a new
    constant matching the column default) and `insertLedgerRow(tx, ..., delta,
    TRANSACTION_TEAM_RESET, "Club reset after owner left")`. `TRANSACTION_TEAM_RESET = 16` is
    added to `common/fmconstants.go`, the finances page's reason labels and both locales.
11. `ResetTeamSettingsStmtSql`: new name/stadium (`RandomTeamName`/`RandomStadiumName`),
    `name_changed = false`, `stadium_capacity`, `ticket_price`, `formation`,
    `friendly_formation`, `sponsor_deal`, `sponsor_deal_season`, `fanbase`, `socios` to
    `DEFAULT`, upgrade/academy timestamps to NULL, `ai_level = RandomAiLevel(tier)`,
    `reset_pending_since = NULL`.
12. Commit, then best-effort: seed the default lineup (`SetDefaultLineup(SelectBestLineup)`,
    as `AssignTeam` does), send loan-return notifications, invalidate the club's Redis cache
    keys, increment `TeamsResetCounter` (new OTel counter, attribute `reason`).

The ledger row and budget write happen in the same tx, which keeps the "money only moves through
`AdjustTeamBudget`-style ledgered writes" rule; `AdjustTeamBudget` itself opens its own tx, so
the in-tx `insertLedgerRow` helper is used, as other multi-step operations already do.

### D6. `common.DeleteAccount(dbCtx, userId)` — one transaction
Lock the user `FOR UPDATE`; `ReleaseTeam` the club if any; for each group owned by the user,
`UPDATE groups SET owner_id = (SELECT user_id FROM group_members WHERE group_id = g AND user_id
<> $1 ORDER BY joined_at LIMIT 1)` and delete the groups where that is NULL; then
`DELETE FROM users WHERE user_id = $1` (cascades notifications, preferences, push subscriptions,
memberships). After commit: add the id to the replica's deleted-id map, drop cached user data in
Redis. Used by `DELETE /v1/user/account` (body `{"confirm": "..."}`, compared case-sensitively
to the club name, or `DELETE`/`BORRAR` with no club; responds 204 and clears the cookie),
`DELETE /v1/admin/users/:id` (403 for one's own id) and the 365-day erasure. Users do not need to
re-authenticate: deletion needs the session plus the typed confirmation, and cross-site forgery
is blocked by the `X-Striker-Client` header from `harden-public-deployment` (and DELETE is not a
simple request anyway).

Nothing personal remains elsewhere: notification texts and ledger descriptions name clubs and
players, never users; logs carry user ids, not emails (the one `LogNotifier` line logs message
text — acceptable, covered by log retention).

### D7. Data export `GET /v1/user/account/export`
`common.ExportUserData(userId)` builds a struct `{exportedAt, profile, club, notificationPreferences,
pushSubscriptions: [{endpoint, createdAt}], groups: [{name, role, joinedAt}], notifications}`;
the handler sets `Content-Disposition: attachment; filename="striker-my-data.json"`. Push keys
(`p256dh`, `auth`) are omitted — they are secrets, not information about the person.

### D8. Terms acceptance
`common.CurrentTermsVersion = 1` (Go constant; the frontend gets it from `/auth/me`).
`LoggedUser` gains `termsAccepted bool` (`terms_version >= CurrentTermsVersion`) and
`releasedTeamName`. `POST /v1/auth/accept-terms` with `{"version": n, "ageConfirmed": true}`
stores both columns (400 if the version is not current or age is not confirmed).
`SelectCountryHandler` returns 400 unless terms are accepted. Frontend: `Layout`'s gate gets a
second branch — not accepted → `/accept-terms` (shows both documents' summaries, the checkbox
and a link to delete the account) — evaluated before the onboarding branch. Onboarding itself
shows the checkbox for first-time users and calls accept-terms before selecting the country.
After onboarding consumes it, `released_team_name` is cleared.

### D9. Public pages and legal properties
`frontend/src/features/legal/PrivacyPage.tsx`, `TermsPage.tsx`, `LegalNoticePage.tsx`,
`AcceptTermsPage.tsx`; `features/account/AccountPage.tsx` (profile, inactivity rule, export,
delete dialog). Routes `/privacy`, `/terms`, `/accept-terms`, `/account`, and `/legal-notice`
only when enabled. Links: login page footer, help page, nav drawer, header account menu
("Account").

The texts are i18n'd under `legal.privacy.*`, `legal.terms.*`, `legal.notice.*` with
interpolation placeholders (`{{controllerName}}`, `{{contactEmail}}`, ...). The values are
deployment properties added to the runtime config:

| Property | Required | Used in |
|---|---|---|
| `LEGAL_CONTROLLER_NAME` | yes | privacy, terms, legal notice |
| `LEGAL_CONTACT_EMAIL` | yes | privacy (rights section), terms, legal notice |
| `LEGAL_CONTROLLER_TAX_ID` | no (needed for the legal notice) | legal notice |
| `LEGAL_POSTAL_ADDRESS` | no (needed for the legal notice) | privacy, legal notice |
| `LEGAL_HOSTING_PROVIDER` / `LEGAL_HOSTING_REGION` | no | privacy (recipients) |
| `LEGAL_NOTICE_ENABLED` | no, `true` to enable | route + links |

Each gets a `REPLACE_LEGAL_*` placeholder in `public/config.js`, a block in `set-env.sh`, a
`window.extended` type entry and a getter in `src/config.ts` (`getLegalConfig()` returning one
typed object, empty strings for unset values; `runtimeValue`'s key union is widened), and a
`VITE_LEGAL_*` equivalent for `npm run dev`. Because addresses and names can contain `|`, `&` or
`/`, `set-env.sh` escapes the value for `sed` (or writes the file with `envsubst`) rather than
substituting it raw. A missing required value renders an inline `Alert` ("legal details not
configured") instead of the value; an optional one drops its sentence. The policy version
(`CurrentTermsVersion`) stays a code constant: the texts are versioned with the code, only the
identity is configured.

### D10. Owner identity
`GetTeamStmtByIdSql` stops joining `users`; `common.Team.OwnerName` and `api/types.ts`
`ownerName` are removed (no component reads it). The group page keeps `userName` (disclosed).

## Risks / Trade-offs

- [A mass of free agents after an inactivity wave (22 per club)] → they flow into the existing
  free-agent market, whose 2-season clock retires the unsigned; watch the free-agent count
  gauge after the first releases. The upgrade grace period (D1) delays the first wave to 60
  days after deploy.
- [Push warnings only reach users who enabled push; email is log-only] → the inactivity rule is
  shown on the Account page, onboarding and in the Terms; real email delivery is a follow-up.
- [A stale JWT on another replica serves up to 1 h of reads for a deleted user before its
  deleted-id map knows] → reads only; every write path that could recreate data
  (`SelectCountryHandler`, activity update) checks the row, and the token expires within 24 h.
- [Reset guard keeps postponing a club with back-to-back matches] → the guard window is 2.5 h;
  match spacing is far larger, so at most a few passes are skipped.
- [Group ownership handover surprises remaining members] → acceptable; documented in the
  Terms.
- [An operator deploys without the `LEGAL_*` properties] → the pages show a visible warning and
  the production checklist lists them.
- [Policy texts are drafts, not legal advice] → stated in tasks; the owner fills the
  controller fields and decides on a lawyer's review.

## Migration Plan

1. Deploy: `EnsureSchema` applies `init92` on start; existing users get `last_seen_at = now()`
   and `terms_version = NULL`, so on next visit they see the accept-terms screen once.
2. Before exposing the app publicly, set the `LEGAL_*` properties and follow the production
   checklist in `HOWTO_RUN.md`.
3. Rollback: the migration is additive; an older binary ignores the new columns. A club already
   pending reset on rollback stays ownerless with `ai_level = 0` until the old startup
   `BackfillAiLevels` gives it a level (old squad kept) — harmless.

## Code map: user-accounts

- Schema: `backend/resources/sql/init92_user_account_lifecycle.sql` (`users.created_at`,
  `last_seen_at`, `inactivity_warning_stage`, `terms_version`, `terms_accepted_at`,
  `released_team_name`; `teams.reset_pending_since`, `reset_reason`; `player_loans.release_on_return`).
- Constants, activity, release and reset: `common/accounts.go` (`InactivityWarningDays`,
  `InactivityReleaseDays`, `AccountErasureDays`, `ActivityTracker`, `TouchUserActivity`,
  `ReleaseTeam`, `ReleaseUserTeam`, `ReleaseInactiveOwner`, `ResetReleasedTeam`, `ClaimWarningStage`).
  `common/loans.go` (`ReturnLoan` and `returnLoanInTx` honour `release_on_return`);
  `GetFreeTeamStmtSql`/`BackfillAiLevelsStmtSql` skip pending clubs; `common.AssignTeam` locks the
  user row and clears `released_team_name`.
- Deletion, export, terms: `common/accountdeletion.go` (`DeleteAccount`, `DeleteAccountIfIdle`,
  `ExportUserData`, `GetAccountInfo`, `AcceptTerms`, `DeleteConfirmationMatches`).
- Activity and deleted-user sessions: `handlers/accountHandlers.go` (`ActivityMiddleware`,
  `Activity`), registered after the JWT middleware on every authenticated group in
  `service/fmservice.go`; logins touch activity in the `Authenticator`.
- API: `GET/DELETE /v1/user/account`, `GET /v1/user/account/export`, `POST /v1/auth/accept-terms`,
  `DELETE /v1/admin/users/:id` (`handlers/accountHandlers.go`); `SelectCountryHandler` refuses
  (400) until the terms are accepted and answers 401 for a deleted user.
- Engine: `engine/accountlifecycle.go` (`RunAccountLifecycle`, `ResetReleasedTeams`, the two jobs),
  triggers in `engine/scheduler.go` (daily 03:17 UTC, every 15 minutes), wired in `engine/fmengine.go`;
  cache drop `engine.InvalidateTeamCaches`; loan-return notice `engine/loans.go`.
- Ledger/metrics/notifications: `TRANSACTION_TEAM_RESET`, `NOTIFICATION_ACCOUNT`
  (`common/fmconstants.go`), `TeamsResetCounter` and its dashboard panel
  (`resources/grafana/striker-dashboard.json`), Spanish templates in
  `common/notificationtranslations.go`.
- Frontend: `src/features/account/` (`AccountPage`, `DeleteAccountDialog`, `deleteConfirmation`),
  hooks in `src/api/hooks.ts` (`useAccount`, `useDeleteAccount`, `downloadMyData`,
  `useAdminDeleteUser`), admin users tab (`features/admin/UsersTab.tsx`), header menu entry.
- Tests: `common/accounts_test.go`, `common/accounts_integration_test.go`,
  `engine/accountlifecycle_concurrency_test.go`, `service/security_test.go`.

## Code map: privacy-and-legal

- Legal identity: `frontend/src/config.ts` (`getLegalConfig`), placeholders in
  `frontend/public/config.js`, escaped substitution in `frontend/set-env.sh`, env pass-through in
  `frontend/docker-compose.yml`, `VITE_LEGAL_*` in `.env.development.local.sample`.
- Pages: `frontend/src/features/legal/` (`PrivacyPage`, `TermsPage`, `LegalNoticePage`,
  `LegalDocument`, `LegalLinks`, `AcceptTermsPage`, `useLegalValues`, `legalMeta`), routes in
  `src/app/App.tsx` (the legal notice route only when `noticeEnabled`); texts under
  `legal.*` in `src/i18n/locales/{en,es}.json` (drafts for the owner to review).
- Links: login page, help page, navigation drawer (`LegalLinks`), Account page.
- Acceptance: `Layout.tsx` gate (terms before onboarding, legal pages exempt),
  `SelectCountryPage` (age/terms box and the "club released" notice), backend
  `common.AcceptTerms`, `CurrentTermsVersion` (keep in step with `legalMeta.LEGAL_VERSION`),
  `SelectCountryHandler` (400 until accepted).
- Owner identity: `GetTeamStmtByIdSql` no longer joins `users`; `Team.OwnerName` removed; friend
  group note `groups.namesNote`.
- Tests: `features/legal/*.test.tsx`, `app/Layout.test.tsx`, `common/accounts_test.go`
  (`TestTeamJSONHasNoOwnerName`).
