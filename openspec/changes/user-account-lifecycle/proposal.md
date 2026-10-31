# Proposal: user-account-lifecycle

## Why

Clubs belong to a human forever once claimed: an owner who stops playing keeps the club frozen
(no CPU management, no transfers, a stale squad) and blocks it from new sign-ups, and there is no
way at all for a person to leave the game and have their personal data erased. Before the app is
deployed publicly it also has to meet the basic privacy obligations of the GDPR/LOPDGDD (right to
erasure, access, information, storage limitation, minimum age). The security holes found while
planning this are fixed separately, and first, by `harden-public-deployment`.

## What Changes

- **Activity tracking**: record when each user was last active (`users.last_seen_at`, updated by
  any authenticated request, throttled to once per hour), plus `created_at`.
- **Inactive clubs are released**: an owner inactive for **60 days** loses their club. Warnings
  go out 14 and 7 days before (in-app + push when enabled). The account itself is kept; on their
  next sign-in the user is told their club was released and goes through onboarding again.
- **Club reset**: a released club becomes a fresh CPU club — its whole senior squad becomes free
  agents, loaned-in players go back to their parent club, loaned-out players become free agents
  when their loan ends, academy prospects are removed, pending offers/listings/challenges are
  cancelled, the trainer returns to the market, and a new squad, name, stadium name, budget and
  default settings are generated, with a weighted-random AI level by tier. Club history (results,
  trophy cabinet, ledger, player career histories) is kept.
- **Self-service account deletion** ("unregister"): a new Account page lets a user delete their
  account after typing a confirmation. Their club is released and reset exactly as above, and the
  `users` row with their name and email is deleted, along with their notifications, preferences,
  push subscriptions and group memberships (groups they own pass to another member). Admins get
  the same action for erasure requests received by email.
- **Data export** (right of access/portability): "Download my data" returns a JSON file with
  everything stored about the user.
- **Storage limitation**: an account with no activity for **365 days** is erased automatically,
  with a push warning 14 days before.
- **Privacy & legal**: public Privacy Policy and Terms pages (en/es) linked from the login page,
  footer/help and Account page; sign-up requires confirming the minimum age and accepting the
  terms, recorded with the policy version; a policy version bump asks existing users to accept
  again. The real Google name of a club's owner stops being exposed by the public team API.
- **Legal identity by configuration**: the controller's name, contact email, optional tax id and
  postal address, and the hosting provider/region shown in the policy come from deployment
  properties (`LEGAL_*` in the frontend's runtime `config.js`), not from code. A property
  `LEGAL_NOTICE_ENABLED` turns on a public "Aviso legal / Legal notice" page (LSSI-CE
  identification) for deployments with economic activity.
- **Deleted users' sessions**: a request whose still-valid JWT names a deleted user is answered
  401 and cannot recreate data.

## Capabilities

### New Capabilities
- `user-accounts`: a user's account lifecycle — activity tracking, inactivity warnings and club
  release, the club reset itself, self-service and admin account deletion, data export,
  automatic erasure of long-inactive accounts, and handling of sessions for deleted users.
- `privacy-and-legal`: what the public deployment must disclose and enforce — privacy policy,
  terms and optional legal notice pages with operator-configured identity, age/terms acceptance
  and re-acceptance, data minimisation in public APIs, and storage limited to what is strictly
  necessary.

### Modified Capabilities
- `cpu-team-management`: the "AI levels" requirement also covers a club released from a human
  owner (it draws its AI level by tier like a newly created CPU club).

## Impact

- **Schema**: new migration `resources/sql/init92_user_account_lifecycle.sql` (users activity,
  terms and release columns; `teams.reset_pending_since`; `player_loans.release_on_return`).
- **Backend**: `common/accounts.go` (new: release, reset, delete, export, erasure), auth glue in
  `handlers/handlers.go` and `service/fmservice.go` (activity/existence middleware),
  `common/database.go` (`GetFreeTeamStmtSql`), `common/aimanager.go`
  (`BackfillAiLevelsStmtSql`), `common/loans.go` (`ReturnLoan`), `common/groups.go`, engine
  jobs in `engine/accountlifecycle.go` + `engine/scheduler.go`, a new `TRANSACTION_TEAM_RESET`
  and `NOTIFICATION_ACCOUNT`.
- **API**: new `GET/DELETE /v1/user/account`, `GET /v1/user/account/export`,
  `POST /v1/auth/accept-terms`, `DELETE /v1/admin/users/:id`; `GET /v1/auth/me` gains
  `termsAccepted`, `releasedTeamName`; team responses drop `ownerName`.
- **Frontend**: new `features/account/` (Account page, delete dialog, export), `features/legal/`
  (Privacy, Terms, Legal notice, accept-terms gate), `LEGAL_*` properties in `public/config.js`,
  `set-env.sh` and `src/config.ts`, onboarding consent checkbox and "your club was released"
  notice, header menu entry, en/es locales.
- **Specs/docs**: new `user-accounts` and `privacy-and-legal` specs, `cpu-team-management`
  delta, ARCHITECTURE.md Auth section, backend README; the `LEGAL_*` properties are added to the
  production checklist that `harden-public-deployment` creates in HOWTO_RUN.md.
- **Depends on**: `harden-public-deployment` (ship it first; the deletion endpoint relies on its
  CSRF header and verified-identity fixes).

## Out of scope

- A real email channel (email stays log-only, so warnings reach users only in-app or by push).
- Security fixes (token audience, CSRF, unauthenticated endpoints, secrets, production
  checklist): planned in `harden-public-deployment`.
- A cookie-consent banner: the app only sets strictly necessary storage (session cookie, theme,
  language) and Google's sign-in cookies, which only need disclosure, not consent. Any future
  analytics or ads would need a banner first.
- A lawyer's review of the policy texts, a DPIA, and a formal records-of-processing document.
- Moderating user-chosen club or group names; bans and abuse tooling.

## Decisions taken by the owner

1. **Thresholds**: 60 days to release a club (warnings at 46 and 53 days) and 365 days to erase
   an idle account (warning at 351) — accepted.
2. **Controller details**: configurable per deployment through properties (`LEGAL_*`), including
   whether the legal notice page is shown — nothing hard-coded.
3. **Minimum age**: 16.
4. **Loaned-out players** of a reset club finish their loan and then become free agents.
5. **Club identity on reset**: new random club and stadium names.
