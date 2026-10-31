# Tasks

## 1. Verified identity and the initialization route

- [x] 1.1 `GetUserIdFromCookie` / `GetPictureUrlFromCookie` read `ginjwt.ExtractClaims(c)` (design D1); delete or narrow `TokenClaimsNoValidation`
- [x] 1.2 Remove `GET /v1/initData`; add `POST /v1/admin/initialize` on the admin group; strip `Initialize`'s own user-count/admin check (design D2)
- [x] 1.3 Tests: forged cookie on a route without middleware → no user; forged cookie on `POST /v1/admin/initialize` → 401 and no reset; real admin → 200; `GET /v1/initData` → 404

## 2. Secrets and startup checks

- [x] 2.1 `common/security.go` `CheckDeploymentSecurity` (weak/placeholder `SECRET_JWT`, `SESSION_SECRET`, missing `GOOGLE_OAUTH_CLIENT_ID`), fatal unless dev login is on, ERROR logs + dev-login banner otherwise; call it from `service.StartSvc` (design D4)
- [x] 2.2 Table-driven unit test over secret/client-id/dev-login combinations

## 3. Google sign-in

- [x] 3.1 `DoExternalLogin` validates with `idtoken.Validate` against `GOOGLE_OAUTH_CLIENT_ID`, checks issuer and `email_verified`, reads name/email/picture from the validated payload; injectable validator for tests (design D3)
- [x] 3.2 Add `GOOGLE_OAUTH_CLIENT_ID` to `backend/docker-compose.yml`, `backend/docker-compose-svc-only.yml` and `backend/app.env.sample`
- [x] 3.3 Tests: wrong audience → 403, unverified email → 403, invalid signature → 403, valid payload → user created/logged in

## 4. Account creation endpoints

- [x] 4.1 Remove `POST /v1/registerUser`, `GET /v1/teams`, `handlers.RegisterUser`, `handlers.Teams`, `common.RegisterUserStmtSql` and the `common.RegisterUser` struct
- [x] 4.2 Test: `POST /v1/registerUser` → 404

## 5. CSRF protection

- [x] 5.1 `RequireClientHeader` middleware for POST/PUT/PATCH/DELETE except `/v1/login`; add `X-Striker-Client` to the CORS allowed headers (design D5)
- [x] 5.2 `frontend/src/api/client.ts` sends `X-Striker-Client: web` on every request; check any direct `fetch`/`navigator.sendBeacon` outside the client (e.g. push subscription, service worker) and route them through it or add the header
- [x] 5.3 Tests: backend POST without header → 403, with header → handler runs, OPTIONS preflight passes; vitest that the client sets the header

## 6. Dev-login visibility

- [x] 6.1 Warning strip in `app/Layout.tsx` when the dev-login roster is non-empty; `devLogin.warningStrip` in `frontend/src/i18n/locales/en.json` and `es.json` (design D6)

## 7. Specs and docs

- [x] 7.1 Code map section for the new `api-security` spec (in design.md: the validator does not carry a Code map from a delta spec into the main spec; copy it into the main spec when archiving)
- [x] 7.2 `backend/README.md`: initialize route (replace the `/initData` paragraph), `GOOGLE_OAUTH_CLIENT_ID`, secret requirements, the `X-Striker-Client` header
- [x] 7.3 `ARCHITECTURE.md` Auth section: identity only from verified claims, Google token validation, CSRF header, startup checks
- [x] 7.4 `HOWTO_RUN.md` "Production checklist" (design D7), including that `CORS_ALLOWED_ORIGINS` must list only the public frontend origin(s)

## 8. Verification

- [x] 8.1 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`
- [x] 8.2 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`

## Workflow follow-up

- Implement and ship this change before `user-account-lifecycle`.
- Archive with `/opsx:archive` once merged.
