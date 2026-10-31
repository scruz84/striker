# Design: harden-public-deployment

## Context

See proposal.md for the findings. Relevant facts:

- `service/fmservice.go` builds one `gin-jwt` middleware (`Key: SECRET_JWT`, cookie `jwt`,
  `SameSite=None`, `Secure`, HTTP-only) and applies it per route group. Ungrouped `v1Group`
  routes (`/initData`, `/login`, `/dev-login/users`, `/registerUser`, `/teams`) run without it.
- `handlers.GetUserIdFromCookie` (70 call sites) and `GetPictureUrlFromCookie` re-read the cookie
  with `TokenClaimsNoValidation`. Behind the middleware this is harmless (the middleware already
  rejected bad tokens), outside it is a bypass: `handlers.Initialize` uses it for its admin check.
- `ginjwt.ExtractClaims(c)` returns the claims the middleware verified for this request, or an
  empty map when the middleware did not run.
- A sessions store (`postgres.NewStore`, `SESSION_SECRET`) is also mounted; only `Authenticator`
  writes to it.
- `DoExternalLogin` calls `oauth2.Tokeninfo` and ignores the result's `Audience`/`IssuedTo`, then
  trusts unverified claims. `google.golang.org/api v0.308.0` (already required) ships `idtoken`.
- CORS: `cors.DefaultConfig()` with `AllowOrigins = CorsAllowedOrigins()`, credentials on, and an
  explicit allowed-headers list.
- The frontend sends every request through `src/api/client.ts`.

## Goals / Non-Goals

**Goals:** close every finding with the smallest change; keep local development (dev login,
default compose secrets) working with one env var; fail closed at startup rather than run
insecurely.

**Non-Goals:** replacing `gin-jwt`, changing cookie attributes, adding a WAF, rate limiting.

## Decisions

### D1. Identity only from verified claims
`GetUserIdFromCookie(c)` and `GetPictureUrlFromCookie(c)` keep their signatures but read
`ginjwt.ExtractClaims(c)` instead of parsing the cookie, so no call site changes and a route
without the middleware simply sees no user. `TokenClaimsNoValidation` stays only where a Google
ID token's claims are read after `idtoken.Validate` returned its payload (D3) — or is deleted if
the payload's `Claims` map covers that use. A unit test runs `GetUserIdFromCookie` on a request
with a forged cookie and no middleware → nil. *Alternative*: rename to `CurrentUserId(c)` —
cleaner but touches 70 call sites for no behavior gain.

### D2. Initialization route
Delete `v1Group.GET("/initData", ...)`; register `v1AdminGroup.POST("/initialize",
handlers.Initialize(dbCtx))` (JWT + `RequireAdmin` already on the group). `Initialize` drops
its own user-count/admin logic. The empty-database bootstrap path is the existing
`--initialize` CLI (`docker compose --profile init run --rm init`), which the README already
documents. *Alternative*: keep the "no users yet → open" window — rejected: on a public host
the first visitor after a deploy against an empty DB could trigger it, and the CLI covers it.

### D3. Google ID token validation
`DoExternalLogin`:

```go
payload, err := idtoken.Validate(ctx, loginRequest.Jwt, common.GoogleOAuthClientId())
// checks signature against Google's certs (cached), exp, aud
if err != nil || (payload.Issuer != "accounts.google.com" && payload.Issuer != "https://accounts.google.com") { 403 }
if verified, _ := payload.Claims["email_verified"].(bool); !verified { 403 }
email, name, picture := payload.Claims["email"], ["name"], ["picture"]
```

`common.GoogleOAuthClientId()` reads the new backend env var `GOOGLE_OAUTH_CLIENT_ID` (added to
`docker-compose.yml` and `docker-compose-svc-only.yml` next to `SECRET_JWT`, documented in
README/HOWTO_RUN). Tests inject a validator func (`var validateGoogleIdToken = idtoken.Validate`)
so they can return a crafted payload or error without network. *Alternative*: keep `tokeninfo`
and compare `Audience` — one network round trip per login and a deprecated endpoint.

### D4. Startup checks (`common/security.go`, called from `service.StartSvc`)
`CheckDeploymentSecurity() error`:
- `weakSecret(v) = len(v) < 32 || v in {"dev-only-change-me", "my-secret", "my-session-secret"}`
  for `SECRET_JWT` and `SESSION_SECRET`;
- `GOOGLE_OAUTH_CLIENT_ID` empty.
With `DevLoginEnabled()` false any hit returns an error and `StartSvc` exits via `log.Fatalf`;
with it true each hit is logged at ERROR plus the "DEV LOGIN ENABLED — never on a public
deployment" banner. Table-driven unit test over the combinations.

### D5. CSRF header middleware
`service.RequireClientHeader()` registered on the router before the groups: for
`POST/PUT/PATCH/DELETE` with path ≠ `/v1/login`, `c.GetHeader("X-Striker-Client") != "web"` →
`403 {"message":"missing X-Striker-Client header"}`. `OPTIONS` preflights pass through (CORS
middleware answers them first). `X-Striker-Client` is appended to `corsConfig` allowed headers.
`src/api/client.ts` sets it on every request (including login, harmless). Why it works: a
cross-site page can only send a custom header with a CORS preflight, which the origin allow-list
rejects; forms and `<img>` cannot set headers at all. *Alternative*: `SameSite=Lax` — breaks
deployments whose frontend and API are on different registrable domains; double-submit token —
more moving parts for the same protection.

WebSocket live routes are GETs and unaffected.

### D6. Dev-login visibility
Startup banner from D4. Frontend: `Layout` already has `useDevLoginUsers()` data on the login
page; move the hook call into `Layout` (cached by TanStack Query) and render a thin MUI
`Alert severity="warning"` strip ("Development login enabled — not for public use",
`devLogin.warningStrip` in en/es) whenever the roster is non-empty.

### D7. Production checklist
New `HOWTO_RUN.md` section "Production checklist" with one line per item of the spec
requirement, each naming the env var or proxy setting (`DEV_LOGIN_ENABLED=false`,
`SECRET_JWT`/`SESSION_SECRET` from `openssl rand -base64 48`, `GOOGLE_OAUTH_CLIENT_ID` on both
services, `CORS_ALLOWED_ORIGINS`, HTTPS + HSTS + CSP at nginx, proxy rate limit on `/v1/login`
and `/v1`, log rotation ≤ 30 days, backup retention ≤ 30 days, secrets in the host's secret
store, never committed).

## Risks / Trade-offs

- [Existing non-dev deployments with default secrets stop starting after upgrade] → intended;
  the error names the variable and the checklist says how to generate one. Rotating
  `SECRET_JWT` signs everyone out once.
- [`idtoken.Validate` needs outbound HTTPS to fetch Google certs] → same dependency `tokeninfo`
  already had; certs are cached between logins.
- [A forgotten client (script, curl) without the header gets 403] → documented in the README's
  API section; only the bundled frontend talks to the API.
- [Removing `GET /v1/initData` breaks someone's bookmark/script] → the CLI path is documented and
  the admin POST route replaces it.

## Migration Plan

1. Set `GOOGLE_OAUTH_CLIENT_ID` (same value as the frontend's) and strong `SECRET_JWT` /
   `SESSION_SECRET` in the backend environment.
2. Deploy backend and frontend together (the frontend must send the new header before the
   backend requires it; deploying the frontend first is also safe, the header is ignored by the
   old backend).
3. Rollback: redeploy the previous images; no schema change is involved.
