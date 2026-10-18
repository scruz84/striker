---
name: Striker Rebuild Images
description: Rebuild (regenerate) the Docker images for the Striker football manager - backend svc/engine and the frontend - from scratch with --no-cache, then recreate the running containers. Use when the user asks to rebuild, regenerate, refresh or redeploy the Striker images.
---

# Striker - rebuild Docker images

Use this skill when the user wants the Striker Docker images regenerated (e.g. "rebuild the
Striker images", "regenerate backend", "redeploy the frontend after my changes").

## Requirements

This skill only contains instructions. To follow it you need a tool that can run shell commands
on the Docker host (Open Terminal, a shell/exec tool, etc.). If no such tool is available, give
the user the exact commands below instead of pretending they ran.

Never claim an image was rebuilt unless a command's output shows the build finished successfully.

## Layout facts (do not guess, these are fixed)

- Repository root: ask the user once where the `striker` checkout lives if you don't already
  know it (below it is written `$STRIKER_DIR`). The two halves have **separate** compose files
  and must each be run from their own directory.
- **Backend** - `$STRIKER_DIR/backend`
  - The backend uses the **external-services compose file**: `docker-compose-svc-only.yml`
    (no bundled Postgres/Redis - they already run elsewhere). Always pass it with `-f`.
    Do **not** use `docker-compose.yml` for the backend.
  - Compose project name: `striker-backend`.
  - Services: `svc` (API, port `1984`) and `engine` (simulation scheduler). Both build from
    `resources/Dockerfile` with `backend/` as the context, so each service gets its own image.
  - `init` is a one-off, **destructive** seed/reset job behind the `init` profile. A plain
    `build` does not touch it. Never run it as part of a rebuild.
- **Frontend** - `$STRIKER_DIR/frontend`
  - Compose file: `docker-compose.yml`, project name `striker-frontend`, service `frontend`
    (nginx, port `3000`).
  - `API_URI` and `GOOGLE_OAUTH_CLIENT_ID` are **runtime** variables read from `frontend/.env` or
    the shell when the container is created. They are not baked in at build time.

## Ask first (only if unclear)

Which part should be rebuilt: **backend**, **frontend**, or **both** (default if the user just
says "the Striker images")? Do not ask anything else; the defaults below are fine.

## Procedure

Run each step, check the exit code, and stop at the first failure (report the last ~30 lines of
the error, do not retry blindly).

### 1. Backend (svc + engine)

```bash
cd "$STRIKER_DIR/backend"

# Rebuild both images from scratch (this is the user's usual command)
docker compose -f docker-compose-svc-only.yml build --no-cache

# Recreate the containers so they run the new images
docker compose -f docker-compose-svc-only.yml up -d --no-deps svc engine
```

Optional, only if the user asks for a fresh `init` image as well (this only builds it, it does
not run it):

```bash
docker compose -f docker-compose-svc-only.yml --profile init build --no-cache init
```

### 2. Frontend

```bash
cd "$STRIKER_DIR/frontend"

docker compose build --no-cache
docker compose up -d --no-deps frontend
```

Before recreating the frontend container, check that `frontend/.env` exists (or that `API_URI`
and `GOOGLE_OAUTH_CLIENT_ID` are set in the environment). If neither is the case, the recreated
container silently falls back to `API_URI=http://localhost:1984` and an empty Google client ID,
which breaks sign-in on any non-local deployment. Warn the user and ask before continuing.

### 3. Verify

```bash
cd "$STRIKER_DIR/backend"  && docker compose -f docker-compose-svc-only.yml ps
cd "$STRIKER_DIR/frontend" && docker compose ps

# Confirm the images are new (CREATED should read "seconds/minutes ago")
docker images --filter "reference=striker-*"

# Quick health look at the backend
docker compose -f "$STRIKER_DIR/backend/docker-compose-svc-only.yml" logs --tail=30 svc engine
```

Report to the user: which images were rebuilt, their new IDs/creation times, and whether the
containers are `running`. If a container is restarting, show its logs.

## Safety rules

- Do **not** run `docker compose down -v`, `docker volume rm`, `docker system prune`, or
  `docker image prune -a`. They can destroy data or other projects' images.
- Do **not** run the `init` service (`--profile init run ...`). It wipes and recreates all game
  data. Only do it if the user explicitly asks for a database reset and confirms.
- `--no-cache` makes the build slow (Go module download, `npm ci`). That is expected; do not
  drop the flag unless the user asks for a quick/cached build.
- Old images become dangling after a rebuild. Offer `docker image prune -f` (dangling only) to
  reclaim disk space, but only run it if the user agrees.
- Don't pass secrets (`SECRET_JWT`, `DB_PASSWORD`, `GOOGLE_OAUTH_CLIENT_ID`, ...) on the command
  line or echo them back in the chat; they live in `.env` / `app.env` files and the environment.

## Quick reference

| Goal | Command (from the right directory) |
|---|---|
| Rebuild backend images | `docker compose -f docker-compose-svc-only.yml build --no-cache` |
| Recreate backend containers | `docker compose -f docker-compose-svc-only.yml up -d --no-deps svc engine` |
| Rebuild only the API image | `docker compose -f docker-compose-svc-only.yml build --no-cache svc` |
| Rebuild only the engine image | `docker compose -f docker-compose-svc-only.yml build --no-cache engine` |
| Rebuild frontend image | `docker compose build --no-cache` |
| Recreate frontend container | `docker compose up -d --no-deps frontend` |
