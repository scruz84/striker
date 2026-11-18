# api-security Specification

## Purpose
How the API authenticates callers and protects state-changing and administrative endpoints so
that it is safe to expose on the public internet: verified sessions, sign-in validation, secret
strength, account creation, cross-site request protection and deployment safeguards.

## Requirements

### Requirement: Verified session identity
Every endpoint that acts on behalf of a user SHALL take the user's identity only from a session token whose signature and expiry the API has verified. A cookie that is unsigned, signed with another key or expired MUST be treated as no session.

#### Scenario: Forged cookie
- **WHEN** a request carries a hand-made `jwt` cookie with `userId` "1" and an invalid signature
- **THEN** the API answers 401 and acts as nobody

#### Scenario: Valid session
- **WHEN** a signed-in user calls an endpoint with their real cookie
- **THEN** the endpoint acts as that user

### Requirement: Protected database initialization
Resetting the game database over HTTP SHALL be possible only through `POST /v1/admin/initialize` by a signed-in admin. There MUST be no unauthenticated or GET route that resets the database; an empty database is initialized with the `--initialize` command-line mode.

#### Scenario: Forged admin cookie
- **WHEN** someone calls `POST /v1/admin/initialize` with a forged cookie naming an admin's id
- **THEN** the API answers 401 and the database is untouched

#### Scenario: Old route
- **WHEN** anyone calls `GET /v1/initData`
- **THEN** the API answers 404

#### Scenario: Admin resets
- **WHEN** a signed-in admin calls `POST /v1/admin/initialize`
- **THEN** the game is re-initialized as before

### Requirement: Strong signing secrets
The API (`--svc`) SHALL refuse to start when `SECRET_JWT` or `SESSION_SECRET` is empty, shorter than 32 bytes or equal to a known placeholder (`dev-only-change-me`, `my-secret`, `my-session-secret`), unless dev login is enabled, in which case it MUST start and log an error naming the weak secret.

#### Scenario: Default secret in production
- **WHEN** `--svc` starts with `SECRET_JWT=dev-only-change-me` and dev login disabled
- **THEN** it exits with an error naming `SECRET_JWT`

#### Scenario: Local development
- **WHEN** `--svc` starts with the default secrets and `DEV_LOGIN_ENABLED=true`
- **THEN** it starts and logs an error that the secrets are not safe for a public deployment

### Requirement: Google sign-in verification
A Google sign-in SHALL be accepted only for an ID token with a valid Google signature, not expired, issued by `accounts.google.com`, whose audience equals the configured `GOOGLE_OAUTH_CLIENT_ID`, and whose email is verified. Any other token MUST be rejected with 403 without creating or updating an account.

#### Scenario: Token issued for another app
- **WHEN** someone submits a valid Google ID token whose audience is a different app's client id
- **THEN** the login is rejected with 403

#### Scenario: Unverified email
- **WHEN** a token has `email_verified` false
- **THEN** the login is rejected with 403

#### Scenario: Missing client id
- **WHEN** `--svc` starts without `GOOGLE_OAUTH_CLIENT_ID` and dev login disabled
- **THEN** it exits with an error naming `GOOGLE_OAUTH_CLIENT_ID`

### Requirement: No unauthenticated account creation
Accounts SHALL be created only by a successful sign-in (Google, or dev login when explicitly enabled). There MUST be no endpoint that creates a user or assigns a club without a verified session.

#### Scenario: Old registration endpoint
- **WHEN** anyone calls `POST /v1/registerUser`
- **THEN** the API answers 404 and no user is created

### Requirement: Cross-site request protection
Every POST, PUT, PATCH or DELETE request to the API, except `POST /v1/login`, SHALL carry the header `X-Striker-Client: web`; a request without it MUST be rejected with 403 before any handler runs. The header MUST be in the CORS allowed headers so the app's own origins can send it.

#### Scenario: Forged form post
- **WHEN** a third-party page auto-submits a form to `POST /v1/user/team/players/17/release` in a signed-in user's browser
- **THEN** the API answers 403 and the player is not released

#### Scenario: Normal app request
- **WHEN** the Striker frontend releases a player
- **THEN** the request carries the header and succeeds

### Requirement: Visible dev login
When dev login is enabled, the API process SHALL log an error at startup saying it must never be enabled publicly, and the frontend MUST show a persistent warning strip on every page.

#### Scenario: Dev login left on
- **WHEN** a visitor opens any page of a deployment with `DEV_LOGIN_ENABLED=true`
- **THEN** they see a "Development login enabled" warning strip

### Requirement: Production deployment checklist
The deployment documentation SHALL list everything a public deployment must set: dev login disabled, strong `SECRET_JWT`/`SESSION_SECRET`, `GOOGLE_OAUTH_CLIENT_ID` on backend and frontend, HTTPS only, the allowed CORS origins, security headers and rate limiting at the proxy, log and backup retention of at most 30 days, and secrets kept out of the repository.

#### Scenario: Operator prepares a deployment
- **WHEN** an operator follows the production checklist
- **THEN** every item above is covered with the environment variable or setting to use

## Code map

- Verified identity: `backend/handlers/handlers.go` (`GetUserIdFromCookie`, `GetPictureUrlFromCookie`
  read `ginjwt.ExtractClaims`); the unverified-cookie parser `TokenClaimsNoValidation` is deleted.
  Tests: `handlers/security_test.go`, `service/security_test.go`.
- Database initialization: `handlers/initializer.go` (`Initialize`), route
  `POST /v1/admin/initialize` in `service/fmservice.go` (`registerV1Routes`); the CLI path is
  `--initialize` in `cmd/striker.go`.
- Startup checks and secrets: `common/security.go` (`CheckDeploymentSecurity`,
  `DeploymentSecurityProblems`, `GoogleOAuthClientId`), called from `service.StartSvc`; tests in
  `common/security_test.go`; env wiring in the root `docker-compose.yml`, `.env.sample` and
  `app.env.sample`.
- Google sign-in: `handlers/handlers.go` (`verifyGoogleIdToken`, `validateGoogleIdToken`,
  `DoExternalLogin`).
- Account creation: `RegisterUser`, `Teams`, `RegisterUserStmtSql` and the `RegisterUser` struct
  are removed; accounts come from `DoExternalLogin` / `DoDevLogin` and
  `SelectCountryHandler` only.
- CSRF: `backend/service/clientheader.go` (`RequireClientHeader`, `ClientHeader`), CORS allowed
  header in `service/fmservice.go`; frontend `src/api/client.ts` (`CLIENT_HEADER`).
- Dev-login visibility: startup ERROR in `common.CheckDeploymentSecurity`; frontend
  `src/app/DevLoginWarning.tsx` mounted in `Layout.tsx`, text `devLogin.warningStrip` (en/es).
- Production checklist: `HOWTO_RUN.md` ("Production checklist"); security notes in
  `backend/README.md` ("Security") and `ARCHITECTURE.md` ("Auth").
