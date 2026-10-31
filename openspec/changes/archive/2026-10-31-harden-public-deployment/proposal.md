# Proposal: harden-public-deployment

## Why

The API was built for a trusted dev environment. Once anyone on the internet can reach it,
several holes let an attacker sign in as other people, create accounts at will or **wipe the
whole database**. These must be closed before the app is published, independently of (and
before) the account-lifecycle and privacy work in `user-account-lifecycle`.

Findings (verified in the code):

1. **Critical — database wipe with a forged cookie.** `GET /v1/initData` sits outside the JWT
   middleware and its admin check reads the user id with `TokenClaimsNoValidation`, so a
   hand-made, unsigned cookie with `userId` of an admin (ids are sequential) runs
   `InitializeGame`. Being a GET, an `<img>` tag on any site also triggers it from a real
   admin's browser.
2. **Critical — forgeable sessions with default secrets.** Both compose files default
   `SECRET_JWT`/`SESSION_SECRET` to `dev-only-change-me`; a deployment that forgets to set them
   accepts JWTs anyone can sign for any user, admins included.
3. **High — Google token audience not checked.** `DoExternalLogin` calls Google `tokeninfo`,
   discards the answer and reads the claims unverified: any Google ID token minted for *any*
   app (and with an unverified email) signs in as that email here.
4. **High — unauthenticated account creation.** `POST /v1/registerUser` creates a user with any
   name/email/country and hands it a club, with no login. `GET /v1/teams` is a leftover stub.
5. **Medium — CSRF.** The session cookie is `SameSite=None`; endpoints such as
   `POST /v1/user/team/players/:id/release` take no body, so a cross-site form post performs
   them in a signed-in user's browser.
6. **Hygiene** — dev login (no credentials) can be left on by mistake with nothing visible, and
   there is no written production checklist.

## What Changes

- Move the HTTP database reset to `POST /v1/admin/initialize` behind the JWT and admin
  middleware; **BREAKING**: `GET /v1/initData` is removed, and an empty database is bootstrapped
  only with the existing `--initialize` CLI / compose `init` profile.
- Read the caller's identity only from the verified token (the claims the JWT middleware has
  already checked), never by parsing the cookie without verification.
- Refuse to start `--svc`/`--engine` when `SECRET_JWT` or `SESSION_SECRET` is empty, shorter than
  32 bytes or a known placeholder, unless dev login is enabled.
- Validate Google ID tokens locally (signature, expiry, issuer, `aud` = this deployment's client
  id) and require `email_verified`; new backend env var `GOOGLE_OAUTH_CLIENT_ID`.
- **BREAKING**: remove `POST /v1/registerUser` and `GET /v1/teams`.
- Require the header `X-Striker-Client: web` on every state-changing request except login; the
  frontend client always sends it.
- Make an enabled dev login loud: ERROR log at startup and a warning strip in the UI.
- Add a production checklist to `HOWTO_RUN.md`.

## Capabilities

### New Capabilities
- `api-security`: how the API authenticates callers and protects state-changing and
  administrative endpoints — verified sessions, sign-in token validation, secret strength,
  account creation, cross-site request protection, dev-login safeguards and the production
  checklist.

### Modified Capabilities
_None._

## Impact

- **Backend**: `service/fmservice.go` (routes, CSRF middleware, CORS allowed headers, startup
  checks), `handlers/handlers.go` (`DoExternalLogin`, `GetUserIdFromCookie`,
  `GetPictureUrlFromCookie`, removal of `RegisterUser`/`Teams`), `handlers/initializer.go`,
  `handlers/auth.go`, `common/database.go` (`RegisterUserStmtSql` removed), `common/fmstructs.go`
  (`RegisterUser` struct removed), new `common/security.go`, both docker-compose files.
- **Frontend**: `src/api/client.ts` (header), dev-login warning strip in `app/Layout.tsx`,
  en/es locales. (No frontend code calls `/initData`.)
- **Docs**: `backend/README.md` (initialize route, env vars), `ARCHITECTURE.md` Auth section,
  `HOWTO_RUN.md` production checklist. New spec `api-security`.
- **Operators**: must set `GOOGLE_OAUTH_CLIENT_ID` on the backend and real secrets before
  upgrading a non-dev deployment, or the services refuse to start.

## Out of scope

- Rate limiting and bot protection (the reverse proxy's job; listed in the checklist).
- Token revocation / server-side sessions (deleted-user sessions are handled in
  `user-account-lifecycle`).
- WebSocket origin checks on live-match streams (`CheckOrigin` returns true): the streams are
  read-only and behind the JWT middleware; revisit if they ever accept writes.
- Security headers (CSP, HSTS) on the static frontend: set at nginx/proxy level, noted in the
  checklist.

## Open questions for the owner

_None._ Implement this change before `user-account-lifecycle`.
