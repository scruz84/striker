# How to run Striker

Two independent halves, each with its own `docker-compose.yml`: `backend/` (API + engine +
Postgres + Redis) and `frontend/` (nginx-served production build). They don't share a compose
file - run each from its own directory.

## Quick start (bundled Postgres/Redis)

This is the default path: Docker Compose brings up its own Postgres and Redis containers for
you.

```bash
# 1. Backend - API server, background engine, Postgres, Redis
cd backend
docker compose up --build -d

# 2. Seed the game world (destructive - wipes and recreates all game data). Only needed once,
#    or whenever you deliberately want a fresh world.
docker compose --profile init run --rm init

# 3. Frontend - production build served by nginx
cd ../frontend
docker compose up --build -d
```

- Backend API: `http://localhost:1984` (override with `SVC_PORT`)
- Frontend: `http://localhost:3000` (override with `FRONTEND_PORT`)

The frontend needs to know where the backend is reachable **from the browser**, not from inside
its own container - set `API_URI` (backend's `docker-compose.yml` defaults it to
`http://localhost:1984`, which is right for both running on the same machine):

```bash
cd frontend
API_URI=http://localhost:1984 GOOGLE_OAUTH_CLIENT_ID=<your client id> docker compose up --build -d
```

Without a real `GOOGLE_OAUTH_CLIENT_ID`, Google sign-in won't work - set
`DEV_LOGIN_ENABLED=true` on the backend instead (see below) to sign in without any OAuth setup at
all. **Never set that in a real deployment** - it lets anyone sign in as any of a fixed dev-user
roster, admin included, with no credential check.

### Common backend env vars (`backend/.env`, or exported before `docker compose up`)

| Variable | Default | Purpose |
|---|---|---|
| `DB_DATABASE` / `DB_USER` / `DB_PASSWORD` | `striker` / `striker` / `striker` | Bundled Postgres credentials |
| `DB_PORT` | `5432` | Host port Postgres is published on |
| `SVC_PORT` | `1984` | Host port the API is published on |
| `SECRET_JWT` / `SESSION_SECRET` | dev-only placeholders | **Set real secrets for anything beyond local dev** |
| `REALM` | `Striker Manager` | JWT realm name |
| `SESSION_TIMEOUT` | `30m` | JWT session lifetime |
| `DEV_LOGIN_ENABLED` | `false` | Opt into no-credential-check sign-in (local/CI only) |
| `CORS_ALLOWED_ORIGINS` | *(empty)* | Extra allowed origins beyond the built-in defaults, comma-separated - needed if the frontend runs anywhere other than `:3000`/`:5173`/the deployed domains |
| `ADMIN_EMAILS` | *(unset)* | Comma-separated emails granted commissioner/admin access on login |

### Common frontend env vars (`frontend/.env`, or exported before `docker compose up`)

| Variable | Default | Purpose |
|---|---|---|
| `API_URI` | `http://localhost:1984` | Backend URL - called directly by the browser, so it must be reachable from wherever the user's browser runs, not from inside the frontend's own container |
| `GOOGLE_OAUTH_CLIENT_ID` | *(empty)* | Google OAuth client ID for sign-in |
| `FRONTEND_PORT` | `3000` | Host port nginx is published on |

Full details (every endpoint, every admin feature, the full schema-migration list) live in
`backend/README.md` and `frontend/README.md` - this file only covers *running* the services.

## Local development without Docker

```bash
# Backend - needs a reachable Postgres + Redis (local install, or `docker compose up postgres redis`
# from backend/ to get just those two containers without the app services)
cd backend
cp app.env.sample app.env   # fill in DB_*/REDIS_*/SECRET_JWT/etc
go run ./cmd --svc --engine   # API + scheduler in one process; split into two terminals if preferred

# Frontend
cd frontend
npm install
cp .env.development.local.sample .env.development.local   # set VITE_GOOGLE_OAUTH_CLIENT_ID
npm run dev
```

Frontend dev server defaults to `VITE_API_ENDPOINT=http://localhost:1984` - change it in
`.env.development.local` if the backend runs elsewhere.

## Running against Postgres/Redis that already run elsewhere

`backend/docker-compose.yml` bundles its own Postgres/Redis containers and points `svc`/`engine`/
`init` at them by container name (`DB_HOST=postgres`, `REDIS_HOST=redis`) - those two values are
**hardcoded in the compose file**, not read from your `.env`, so simply setting `DB_HOST`/
`REDIS_HOST` in `backend/.env` has no effect on the bundled setup. To point at Postgres/Redis
running on other hosts (a managed database, a shared dev server, separate containers you already
run), override the app services' environment and skip starting the bundled ones.

1. Create an override file, e.g. `backend/docker-compose.external.yml`:

   ```yaml
   services:
     svc:
       environment:
         DB_HOST: db.example.internal
         DB_PORT: "5432"
         DB_DATABASE: striker
         DB_USER: striker
         DB_PASSWORD: <real password>
         REDIS_HOST: cache.example.internal
         REDIS_PORT: "6379"
         REDIS_PASSWORD: <real password, or "" if none>
         REDIS_DB: "0"
     engine:
       environment:
         DB_HOST: db.example.internal
         DB_PORT: "5432"
         DB_DATABASE: striker
         DB_USER: striker
         DB_PASSWORD: <real password>
         REDIS_HOST: cache.example.internal
         REDIS_PORT: "6379"
         REDIS_PASSWORD: <real password, or "" if none>
         REDIS_DB: "0"
     # Only include this if you'll run the seed/reset job too
     init:
       environment:
         DB_HOST: db.example.internal
         DB_PORT: "5432"
         DB_DATABASE: striker
         DB_USER: striker
         DB_PASSWORD: <real password>
         REDIS_HOST: cache.example.internal
         REDIS_PORT: "6379"
         REDIS_PASSWORD: <real password, or "" if none>
         REDIS_DB: "0"
   ```

   (Compose merges `environment` maps per-service, key by key, across `-f` files - an override
   file only needs to list the keys that actually change; every other key from the base file's
   `environment` block, like `SECRET_JWT` or `DEV_LOGIN_ENABLED`, still applies. The full
   `DB_*`/`REDIS_*` set is listed above for clarity, but e.g. `DB_DATABASE`/`DB_USER`/
   `DB_PASSWORD` can be left out if they're already correct via `backend/.env`.)

2. Bring up just the app services, with `--no-deps` so Compose doesn't also start the bundled
   Postgres/Redis containers it would otherwise pull in via `depends_on`:

   ```bash
   cd backend
   docker compose -f docker-compose.yml -f docker-compose.external.yml up --build --no-deps -d svc engine
   ```

   Run the seed job the same way, once, if the external database is empty:

   ```bash
   docker compose -f docker-compose.yml -f docker-compose.external.yml --profile init run --rm --no-deps init
   ```

3. If Postgres doesn't already have the schema loaded (a genuinely empty external database), the
   `init` run above creates it (`--initialize` calls `common.EnsureSchema`, same as the bundled
   case) - no manual `psql` step needed for a first-time setup. If instead it's an
   **already-initialized** external database missing a schema file added since it was last
   initialized, apply just that file manually - see backend/README.md's "Schema changes on an
   already-running database" section, adjusting the `psql` connection details to the external
   host.

The frontend doesn't talk to Postgres/Redis at all (only to the backend API over HTTP), so
nothing about it changes in this scenario - point its `API_URI` at wherever `svc` ends up
reachable, same as the quick-start section above.
