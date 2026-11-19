# How to run Striker

Everything runs from one Compose file at the repository root, `docker-compose.yml`: the API
(`svc`), the simulation engine (`engine`), the frontend (nginx), and - unless you use your own -
Postgres and Redis. Settings live in one `.env` file next to it.

The application images are published to the GitHub Container Registry on every merge to `main`
(see [Published images](#published-images)), so a server only needs `docker-compose.yml` and a
`.env`, not a checkout or a build. You can still build everything from source.

## Quick start

```bash
cp .env.sample .env          # then edit it - see "Configuration" below
docker compose pull          # fetch the published images
docker compose up -d

# Seed the game world (destructive - wipes and recreates all game data). Only needed once, or
# whenever you deliberately want a fresh world.
docker compose run --rm init
```

- Frontend: `http://localhost:3000` (override with `FRONTEND_PORT`)
- Backend API: `http://localhost:1984` (override with `SVC_PORT`)

Run `init` exactly as shown. Naming the service is enough to start it, and keeps the bundled
Postgres/Redis that `.env` enables; adding `--profile init` would replace those profiles and leave
the database out.

To try it without setting up Google OAuth, set `DEV_LOGIN_ENABLED=true` in `.env`: you can then
sign in as a fixed set of dev users, and the API starts with the placeholder secrets. **Never set
that on a deployment anyone else can reach** - it lets anyone sign in as any dev user, admin
included, with no credential check. Without it the API refuses to start until `SECRET_JWT`,
`SESSION_SECRET` and `GOOGLE_OAUTH_CLIENT_ID` are set (see the production checklist at the end).

The schema needs no separate step: on an empty database, `svc` and `engine` create it themselves
on startup (`common.EnsureSchema`), and they apply any newer migration on every later start.

## Published images or a local build

`docker-compose.yml` names, for each application service, both the published image and the
Dockerfile it comes from:

| Command | What runs |
|---|---|
| `docker compose pull && docker compose up -d` | The published images (`ghcr.io/scruz84/striker-backend` and `ghcr.io/scruz84/striker-frontend`) at `STRIKER_TAG` |
| `docker compose up --build -d` | Images built from this checkout, tagged with the same names |

`STRIKER_TAG` (in `.env`) picks the version: `latest` (the default) or `sha-<commit>` - the first
seven characters of a commit on `main` - to pin one exact build. Pinning is the safer choice on a
server: an upgrade then only happens when you change the tag, and rolling back is setting the
previous one again.

**Upgrading** to a newer published version:

```bash
docker compose pull          # or change STRIKER_TAG first
docker compose up -d         # recreates only the containers whose image changed
```

Compose keeps using the image it already has until you `pull`; `up` alone never fetches a newer
`latest`. After a local `--build`, the local image carries the published name, so run
`docker compose pull` to go back to the published one.

## Configuration

Every setting is an environment variable read from `.env` (or the shell). `.env.sample` lists all
of them with comments; the most common ones:

### Backend

| Variable | Default | Purpose |
|---|---|---|
| `SECRET_JWT` / `SESSION_SECRET` | dev-only placeholders | **Required, >= 32 characters** (`openssl rand -base64 48`): the API refuses to start with a placeholder or short value unless `DEV_LOGIN_ENABLED=true` |
| `GOOGLE_OAUTH_CLIENT_ID` | *(empty)* | **Required** (unless dev login): the audience Google ID tokens are validated against. The frontend uses the same variable |
| `DEV_LOGIN_ENABLED` | `false` | Opt into no-credential-check sign-in (local/CI only) |
| `ADMIN_EMAILS` | *(empty)* | Comma-separated emails granted commissioner/admin access on login |
| `CORS_ALLOWED_ORIGINS` | *(empty)* | Extra allowed origins beyond `http://localhost:3000`/`:5173`, comma-separated - needed for any deployed frontend |
| `REALM` | `Striker Manager` | JWT realm name |
| `SESSION_TIMEOUT` | `24h` | JWT session lifetime |
| `SEEDED_COUNTRIES` | *(empty = es,en,fr,de)* | Countries a fresh world (`init`) is seeded with |
| `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` | *(empty)* | Web push keypair; empty = notifications are only logged |
| `OTEL_METRICS_ENABLED` / `OTEL_EXPORTER_OTLP_ENDPOINT` | `false` / *(empty)* | OpenTelemetry metrics export (see backend/README.md) |
| `DB_*` / `REDIS_*` | the bundled containers | See [Using Postgres or Redis that already run elsewhere](#using-postgres-or-redis-that-already-run-elsewhere) |

### Frontend

| Variable | Default | Purpose |
|---|---|---|
| `API_URI` | `http://localhost:1984` | Backend URL - called directly by the browser, so it must be reachable from wherever the user's browser runs, not from inside the frontend's own container |
| `GOOGLE_OAUTH_CLIENT_ID` | *(empty)* | Google OAuth client ID for sign-in (shared with the backend) |
| `VAPID_PUBLIC_KEY` | *(empty)* | The browser needs the public key too, to subscribe to push (shared with the backend) |
| `LEGAL_*` | *(empty)* | Who runs the deployment, shown in the Privacy Policy and Terms (see the production checklist) |

The frontend writes these into `config.js` when its container starts (`frontend/set-env.sh`), so
changing one needs only `docker compose up -d`, never a rebuild.

### Ports and images

| Variable | Default | Purpose |
|---|---|---|
| `SVC_PORT` / `FRONTEND_PORT` | `1984` / `3000` | Host ports the API and the frontend are published on |
| `POSTGRES_HOST_PORT` / `REDIS_HOST_PORT` | `5432` / `6379` | Host ports of the bundled Postgres/Redis, bound to `127.0.0.1` only |
| `STRIKER_TAG` | `latest` | Version of the published images |
| `COMPOSE_PROFILES` | `postgres,redis` | Which bundled services start |
| `POSTGRES_VOLUME` | `striker_postgres-data` | Volume holding the bundled Postgres data |

Full details (every endpoint, every admin feature) live in `backend/README.md` and
`frontend/README.md` - this file only covers *running* the services.

## Using Postgres or Redis that already run elsewhere

The bundled `postgres` and `redis` services only start when their Compose profile is enabled, and
`.env.sample` enables both (`COMPOSE_PROFILES=postgres,redis`). To use a managed database, a
shared server or containers you already run, remove that service from `COMPOSE_PROFILES` and
point its variables at the real one. Each can be external on its own - for example an external
Postgres with the bundled Redis:

```bash
# .env
COMPOSE_PROFILES=redis             # only the bundled Redis; empty to bundle neither
DB_HOST=db.example.internal
DB_PORT=5432
DB_DATABASE=striker
DB_USER=striker
DB_PASSWORD=<real password>
```

and for an external Redis as well:

```bash
COMPOSE_PROFILES=
REDIS_HOST=cache.example.internal
REDIS_PORT=6379
REDIS_PASSWORD=<real password, or empty if none>
REDIS_DB=0
```

Then start and seed exactly as in the quick start - `docker compose up -d`, and
`docker compose run --rm init` once if the external database has no game world yet. An empty
external database gets its schema from the backend itself, the same as the bundled one, so no
manual `psql` step is needed.

`DB_HOST`/`REDIS_HOST` must be reachable **from inside the containers**: a hostname, an IP, or -
for a database on the Docker host itself - the host's address on the Docker network
(`172.17.0.1` on a default Linux install), since `localhost` inside a container is the container
itself. If Postgres or Redis is a container on another Docker network, attach the backend
services to that network with a `docker-compose.override.yml` next to `docker-compose.yml`
(Compose loads it automatically) and use the container's name as the host:

```yaml
# docker-compose.override.yml
services:
  svc:
    networks: [default, shared]
  engine:
    networks: [default, shared]
  init:
    networks: [default, shared]

networks:
  shared:
    external: true
    name: my-existing-network   # the network the database container is on
```

The frontend never talks to Postgres/Redis (only to the API over HTTP), so nothing about it
changes here.

## Moving from the per-half compose files

Before the all-in-one file, the backend and frontend each had their own compose file
(`backend/docker-compose.yml`, `backend/docker-compose-svc-only.yml` and
`frontend/docker-compose.yml`), with their own `.env` files and the Compose projects
`striker-backend` and `striker-frontend`. To switch a running deployment over:

1. **Merge the settings** into one `.env` at the repository root, starting from `.env.sample`:
   take the backend values from `backend/.env` and the frontend ones (`API_URI`, `LEGAL_*`, ...)
   from `frontend/.env`. `GOOGLE_OAUTH_CLIENT_ID` is now one variable for both.
2. **Choose the services:**
   - From `docker-compose-svc-only.yml` (external Postgres/Redis): set `COMPOSE_PROFILES=` (empty)
     and keep your `DB_*`/`REDIS_*` values.
   - From the bundled `backend/docker-compose.yml`: keep `COMPOSE_PROFILES=postgres,redis` and set
     `POSTGRES_VOLUME=striker-backend_postgres-data`, so the new Postgres container uses the
     existing data instead of starting empty.
3. **Stop the old projects** - without `-v`, which would delete the bundled database's volume:

   ```bash
   docker compose -p striker-backend down
   docker compose -p striker-frontend down
   ```

4. **Start the new one** from the repository root: `docker compose pull && docker compose up -d`.
   Do **not** run `init`: the database already holds your world.

## Published images

`.github/workflows/docker.yml` builds and publishes two images to the GitHub Container Registry:

- `ghcr.io/scruz84/striker-backend` - the API, engine and seed job (one image, three commands)
- `ghcr.io/scruz84/striker-frontend` - the nginx-served frontend

It runs after the CI workflow passes on a push to `main` - so an image is never published for a
commit with failing checks - and on demand from the repository's Actions tab ("Docker images" ->
"Run workflow"). Each image is built for `linux/amd64` and `linux/arm64` (an ARM server or a
Raspberry Pi pulls the right one automatically) and tagged:

- `sha-<commit>` - the first seven characters of the commit it was built from, never moved;
- `latest` - the newest build of `main` (a manual run on another branch publishes only its
  `sha-` tag).

The images carry no secrets: every password, key and client id is passed in as an environment
variable when the container starts. After the first run, check each package's visibility on its
page (the repository's "Packages" section): if it shows as private, set it to public once
("Package settings" -> "Change visibility"), so it can be pulled without logging in and doesn't
count against the account's private storage.

## Local development without Docker

```bash
# Postgres and Redis - any local install, or just the two bundled containers (published on
# 127.0.0.1:5432 and 127.0.0.1:6379):
docker compose up -d postgres redis

# Backend
cd backend
cp app.env.sample app.env   # fill in DB_*/REDIS_*/SECRET_JWT/etc (localhost for the containers above)
go run ./cmd --svc --engine   # API + scheduler in one process; split into two terminals if preferred

# Frontend
cd frontend
npm install
cp .env.development.local.sample .env.development.local   # set VITE_GOOGLE_OAUTH_CLIENT_ID
npm run dev
```

Frontend dev server defaults to `VITE_API_ENDPOINT=http://localhost:1984` - change it in
`.env.development.local` if the backend runs elsewhere.

### Mobile smoke test

`frontend/e2e/mobile-smoke.spec.ts` opens every player-facing page at 360px against a running backend
and fails on horizontal scroll, text under 11px, or a tappable control under 44px tall. It needs the
backend up on a seeded world with `DEV_LOGIN_ENABLED=true` (`go run ./cmd --initialize` seeds one) and is
not part of `npm test` or CI:

```bash
cd frontend
npx playwright install chromium   # once; or set PLAYWRIGHT_CHROMIUM_EXECUTABLE to an existing Chromium
npm run smoke:mobile              # starts `vite` itself if nothing listens on localhost:5173
```

It talks to `http://localhost:1984` by default (`SMOKE_API_URL` / `SMOKE_BASE_URL` override; keep both
on `localhost`, which is the dev origin the backend's CORS and session cookie accept). The
admin panel is not covered, and a freshly seeded world has no played matches, so the match page is
checked in its scheduled view.

## Production checklist

Before exposing a deployment to the internet:

- [ ] `DEV_LOGIN_ENABLED=false` (the default) - an enabled dev login lets anyone sign in as the admin.
- [ ] `SECRET_JWT` and `SESSION_SECRET`: two different random values of at least 32 characters
      (`openssl rand -base64 48`), never the compose placeholders. Rotating `SECRET_JWT` signs
      everyone out once.
- [ ] `GOOGLE_OAUTH_CLIENT_ID` set (one variable, used by both the backend and the frontend).
- [ ] `ADMIN_EMAILS` lists only the real administrators.
- [ ] `CORS_ALLOWED_ORIGINS` lists only the public frontend origin(s) - it also decides who can send
      the `X-Striker-Client` CSRF header.
- [ ] `STRIKER_TAG` pinned to a `sha-<commit>` tag, so the version only changes when you decide.
- [ ] With the bundled Postgres, `DB_PASSWORD` changed from the default `striker` before the first
      start (it is only applied when the database volume is created).
- [ ] HTTPS only (the session cookie is `Secure`); redirect HTTP to HTTPS at the proxy.
- [ ] Security headers at the proxy/nginx: `Strict-Transport-Security`, a `Content-Security-Policy`,
      `X-Content-Type-Options`, `Referrer-Policy`.
- [ ] Rate limiting at the proxy, at least on `/v1/login` and on `/v1` as a whole.
- [ ] Logs rotated and kept at most 30 days; database backups kept at most 30 days.
- [ ] Secrets live in the host's secret store or environment, never in the repository or an image.
- [ ] The database is bootstrapped with `docker compose run --rm init` (there is no
      unauthenticated reset route); `POST /v1/admin/initialize` is admin-only and destroys all data.
- [ ] Frontend `LEGAL_CONTROLLER_NAME` and `LEGAL_CONTACT_EMAIL` set (required: the Privacy Policy and
      Terms name the controller and the contact for privacy requests; without them the pages show
      a "legal details not configured" warning). Optional: `LEGAL_CONTROLLER_TAX_ID`,
      `LEGAL_POSTAL_ADDRESS`, `LEGAL_HOSTING_PROVIDER`, `LEGAL_HOSTING_REGION`. Set
      `LEGAL_NOTICE_ENABLED=true` (with name, tax id and address) if the deployment has economic
      activity and needs the public "Aviso legal" page. Changing them needs only a container
      restart, not a rebuild.
- [ ] The Privacy Policy and Terms texts (`legal.*` in `frontend/src/i18n/locales`) are drafts:
      have them reviewed before going public, and raise `LEGAL_VERSION` (`legalMeta.ts`) together
      with `CurrentTermsVersion` (`backend/common/accounts.go`) when they change.
