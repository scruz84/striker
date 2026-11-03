# user-accounts Specification

## Purpose
How a user's account lives and ends: activity tracking, release of clubs whose owner stopped
playing, the reset that turns a released club into a fresh CPU club, self-service and admin
account deletion, data export, and automatic erasure of long-inactive accounts.

## Requirements

### Requirement: Activity tracking
The system SHALL record for each user when the account was created and when it was last active. Any successful sign-in or authenticated API request MUST count as activity; the stored value MAY lag by at most 1 hour. Push notifications, CPU actions or matches played by the user's club MUST NOT count as activity.

#### Scenario: Request refreshes activity
- **WHEN** a user whose last activity is 3 hours old opens their dashboard
- **THEN** their last activity becomes the current time

#### Scenario: Throttled write
- **WHEN** a user makes 50 requests within 10 minutes
- **THEN** their last activity is written at most once

#### Scenario: Existing users on upgrade
- **WHEN** the change is deployed on a database with existing users
- **THEN** every existing user's last activity starts at the deployment time, so no club is released before 60 days after it

### Requirement: Inactivity warnings
A club owner SHALL be warned that their club will be released when they reach 46 days without activity (14 days left) and again at 53 days (7 days left). Each warning MUST be sent once per inactivity period, as an in-app notification also delivered by push when enabled, in the user's language. Any activity MUST reset the warnings.

#### Scenario: First warning
- **WHEN** the daily lifecycle run finds an owner whose last activity was 46 days and 2 hours ago
- **THEN** they get one "your club will be released in 14 days" notification

#### Scenario: No duplicate
- **WHEN** the next daily run finds the same owner still inactive at 47 days
- **THEN** no new warning is sent

#### Scenario: User comes back
- **WHEN** a warned owner signs in at day 50 and then goes inactive again
- **THEN** the 14-day warning is sent again 46 days after that sign-in

### Requirement: Release of inactive clubs
A club whose owner has been inactive for 60 days or more SHALL be released: the owner loses the club, the account is kept with no club, and the club is reset into a CPU club. The inactivity MUST be re-checked at release time, so an owner active since the check keeps the club.

#### Scenario: Released at 60 days
- **WHEN** the daily lifecycle run finds an owner inactive for 60 days and 1 hour
- **THEN** the club has no owner and is queued for reset, and the user's account still exists without a club

#### Scenario: Active at the last moment
- **WHEN** an owner signs in between the run selecting them and their club being released
- **THEN** the club is not released

#### Scenario: Two engine replicas
- **WHEN** two engine replicas run the daily lifecycle run at the same instant
- **THEN** each inactive club is released exactly once and each warning is sent exactly once

### Requirement: Returning after release
A user whose club was released for inactivity SHALL be told so on their next sign-in, naming the released club, and MUST go through country onboarding again to get a new club. A released club MUST never be handed back automatically.

#### Scenario: Sign-in after release
- **WHEN** a user whose club "Atlético Riverside" was released signs in
- **THEN** onboarding shows that "Atlético Riverside" was released after 60 days of inactivity and asks them to pick a country

### Requirement: Club reset squad
Resetting a club SHALL make every senior player it owns a free agent (contract ended, no shirt number, last club recorded, free-agency clock at 0), remove its academy prospects, send its loaned-in players back to their parent clubs, and generate a brand-new senior squad of the standard size and skill range for its country.

#### Scenario: Squad becomes free agents
- **WHEN** a club with 22 senior players is reset
- **THEN** those 22 players appear on the free-agent market with this club as their last club, and the club has a newly generated squad

#### Scenario: Loaned-in player
- **WHEN** the reset club has a player on loan from another club
- **THEN** the player returns to his parent club immediately and that club's owner is notified

#### Scenario: Loaned-out player
- **WHEN** the reset club has a player out on loan
- **THEN** he stays at the borrowing club until the loan ends and then becomes a free agent instead of returning

### Requirement: Club reset pending business
Resetting a club SHALL cancel its open transfer and loan listings, every pending transfer offer it made or received, and every pending friendly challenge it sent or received, and MUST release its hired trainer to the trainer market and clear its watchlist. Already scheduled matches and friendlies MUST still be played by the reset club.

#### Scenario: Offer for one of its players
- **WHEN** another club has a pending offer for a player of the reset club
- **THEN** the offer disappears and the player can be signed as a free agent instead

#### Scenario: Scheduled friendly
- **WHEN** the reset club has a friendly scheduled for tomorrow
- **THEN** the friendly is played by the reset club's new squad

#### Scenario: Watchlist
- **WHEN** a club that followed 12 players is reset
- **THEN** its watchlist is empty

### Requirement: Club reset settings
Resetting a club SHALL give it a new random name and stadium name for its country and restore every owner-controlled setting to its default: budget 400000, stadium capacity 1000, ticket price 10, formations, sponsor deal, no stadium upgrade in progress, academy promotion cooldown cleared, rename allowance restored, fanbase 50 and 500 socios. The budget change MUST be recorded in the club's ledger.

#### Scenario: Rich club reset
- **WHEN** a club with a budget of 3,250,000 and a 12,000-seat stadium is reset
- **THEN** its budget is 400000 with a ledger entry of -2,850,000 for the reset, and its stadium holds 1000

#### Scenario: Club history kept
- **WHEN** a club with two league titles is reset
- **THEN** its trophy cabinet, past results, past ledger entries and its former players' career histories still show them

### Requirement: Club reset timing
A released club SHALL be reset only when none of its league matches, cup ties or friendlies has a kickoff between 2 hours ago and 30 minutes from now; otherwise the reset MUST wait. Until it is reset, a released club MUST NOT be offered to new users and MUST NOT be managed by the CPU, and it plays its matches with its existing squad and default lineup.

#### Scenario: Match about to start
- **WHEN** a club is released 10 minutes before one of its league matches
- **THEN** it plays that match with its old squad and is reset once 2 hours have passed since kickoff

#### Scenario: Not claimable while pending
- **WHEN** a new user picks the country of a club waiting for its reset
- **THEN** they are given a different free club

#### Scenario: Reset exactly once
- **WHEN** two engine replicas look for released clubs to reset at the same instant
- **THEN** each club is reset exactly once

### Requirement: Self-service account deletion
A signed-in user SHALL be able to delete their account from the Account page after typing a confirmation (their club's name, or the word DELETE/BORRAR if they have none). Deletion MUST release their club for reset, erase their stored name, email, language, notifications, notification preferences, push subscriptions and group memberships, and sign them out.

#### Scenario: Owner deletes account
- **WHEN** the owner of "CD Norte" types "CD Norte" and confirms deletion
- **THEN** their account no longer exists, they are signed out, and "CD Norte" is queued for reset

#### Scenario: Wrong confirmation
- **WHEN** the user types a different text
- **THEN** nothing is deleted and the API answers 400

#### Scenario: Signing in again later
- **WHEN** a deleted user signs in again with the same Google account
- **THEN** a brand-new account is created and they go through onboarding as a first-time user

### Requirement: Groups of a deleted user
When an account is deleted, every friend group it owns SHALL pass to the member who joined it earliest among the remaining members, and a group left with no members MUST be deleted.

#### Scenario: Owner leaves a group of three
- **WHEN** the owner of a group with two other members deletes their account
- **THEN** the group still exists, owned by the member who joined first

#### Scenario: Group of one
- **WHEN** the only member of a group deletes their account
- **THEN** the group is deleted

### Requirement: Admin account deletion
An admin SHALL be able to delete any other user's account from the admin users list, with the same effects as self-service deletion, to handle erasure requests received outside the app. An admin MUST NOT delete their own account from the admin page.

#### Scenario: Erasure request by email
- **WHEN** an admin deletes user 42 from the admin page
- **THEN** user 42's account is erased and their club is queued for reset

### Requirement: Sessions of deleted users
Any authenticated request carrying a still-valid session for a user that no longer exists SHALL be answered 401, and MUST NOT create or modify any data, including claiming a club.

#### Scenario: Second device
- **WHEN** a user deletes their account on the phone and their laptop then calls onboarding with the old session
- **THEN** the laptop gets 401 and no club is assigned

### Requirement: Data export
A signed-in user SHALL be able to download, from the Account page, a JSON file with all personal data stored about them: profile (name, email, provider, country, language, created and last-active times, terms acceptance), their club summary, notification preferences, push subscription endpoints, groups, notifications and their club's watchlist.

#### Scenario: Download my data
- **WHEN** a user clicks "Download my data"
- **THEN** their browser saves `striker-my-data.json` containing their email and their notifications

#### Scenario: Watchlist in the export
- **WHEN** a user who follows 3 players downloads his data
- **THEN** the file lists those 3 players

### Requirement: Erasure of long-inactive accounts
An account inactive for 365 days or more SHALL be deleted automatically with the same effects as self-service deletion. Its user MUST be warned once by in-app notification and push, when enabled, at 351 days without activity.

#### Scenario: Idle account
- **WHEN** the daily lifecycle run finds an account inactive for 365 days and 3 hours
- **THEN** the account is deleted

#### Scenario: Warning before erasure
- **WHEN** an account reaches 351 days without activity
- **THEN** its user is notified once that the account will be deleted in 14 days

## Code map

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
- Watchlist: `ResetReleasedTeam` empties it with `common.ClearWatchlist`; `ExportUserData` adds
  `UserDataExport.Watchlist` (`common.exportWatchlist`, `ExportedWatchedPlayer`). Covered by
  `common/watchlist_integration_test.go` (`TestWatchlist_ClubResetAndExport`).
- Tests: `common/accounts_test.go`, `common/accounts_integration_test.go`,
  `engine/accountlifecycle_concurrency_test.go`, `service/security_test.go`.
