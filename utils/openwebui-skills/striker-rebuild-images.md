---
name: Striker Rebuild Images
description: Update or rebuild the Docker images for the Striker football manager - backend svc/engine and the frontend - either by pulling the published images or by building them from scratch with --no-cache, then recreate the running containers. Use when the user asks to update, rebuild, regenerate, refresh or redeploy the Striker images.
---

# Striker - update or rebuild Docker images

Use this skill when the user wants the Striker containers running newer images (e.g. "update
Striker", "rebuild the Striker images", "redeploy the frontend after my changes").

## Requirements

This skill only contains instructions. To follow it you need a tool that can run shell commands
on the Docker host (Open Terminal, a shell/exec tool, etc.). If no such tool is available, give
the user the exact commands below instead of pretending they ran.

Never claim an image was pulled or rebuilt unless a command's output shows it finished
successfully.

## Layout facts (do not guess, these are fixed)

- Repository root: ask the user once where the `striker` checkout lives if you don't already
  know it (below it is written `$STRIKER_DIR`). **Every command runs from `$STRIKER_DIR`**, the
  repository root: there is a single `docker-compose.yml` there for everything (there are no
  compose files under `backend/` or `frontend/` any more).
- Compose project name: `striker`. Settings live in `$STRIKER_DIR/.env`.
- Services:
  - `svc` (API, port `1984`) and `engine` (simulation scheduler): both run the backend image
    `ghcr.io/scruz84/striker-backend`.
  - `frontend` (nginx, port `3000`): the image `ghcr.io/scruz84/striker-frontend`.
  - `postgres` and `redis`: bundled databases, only started when `.env`'s `COMPOSE_PROFILES`
    lists them. On a deployment with external Postgres/Redis, `COMPOSE_PROFILES` is empty and
    these two do not exist - that is expected, never "fix" it.
  - `init`: a one-off, **destructive** seed/reset job. Never run it as part of an update.
- `STRIKER_TAG` in `.env` selects the published version (`latest` or `sha-<commit>`).
- `API_URI`, `GOOGLE_OAUTH_CLIENT_ID` and the `LEGAL_*` frontend settings are **runtime**
  variables read from `.env` when the container is created. They are not baked into the image.

## Ask first (only if unclear)

1. **Published or local build?** By default, update to the **published** images (they are built
   by GitHub on every merge to `main`). Build from source only if the user asks for it, or wants
   to deploy local changes that are not merged yet.
2. Which part: **backend**, **frontend**, or **both** (default if the user just says "the Striker
   images")?

Do not ask anything else; the defaults below are fine.

## Procedure

Run each step, check the exit code, and stop at the first failure (report the last ~30 lines of
the error, do not retry blindly).

### 0. Check the settings file

```bash
cd "$STRIKER_DIR"
test -f .env && grep -E '^(COMPOSE_PROFILES|STRIKER_TAG|API_URI)=' .env
```

If `.env` does not exist, stop and tell the user: the recreated containers would fall back to
placeholder secrets, `API_URI=http://localhost:1984` and an empty Google client ID. Do not print
any other line of `.env` (it holds secrets).

### 1a. Update to the published images (default)

```bash
cd "$STRIKER_DIR"

# Both halves
docker compose pull svc engine frontend
docker compose up -d --no-deps svc engine frontend

# Backend only:  docker compose pull svc engine   && docker compose up -d --no-deps svc engine
# Frontend only: docker compose pull frontend     && docker compose up -d --no-deps frontend
```

If `STRIKER_TAG` is pinned to a `sha-` tag, `pull` keeps that version: ask the user which tag to
move to, change `STRIKER_TAG` in `.env` only with their agreement, then pull again.

### 1b. Build from the local checkout instead

```bash
cd "$STRIKER_DIR"

# Rebuild from scratch
docker compose build --no-cache svc engine frontend
docker compose up -d --no-deps svc engine frontend

# Backend only:  docker compose build --no-cache svc engine && docker compose up -d --no-deps svc engine
# Frontend only: docker compose build --no-cache frontend   && docker compose up -d --no-deps frontend
```

A local build tags the images with the published names, so a later `docker compose pull` is what
switches back to the published ones.

### 2. Verify

```bash
cd "$STRIKER_DIR"
docker compose ps

# Confirm which images run (CREATED should be recent after a build or a pull of a newer version)
docker images --filter "reference=ghcr.io/scruz84/striker-*"

# Quick health look at the backend
docker compose logs --tail=30 svc engine
```

Report to the user: which images were updated or rebuilt, their new IDs/creation times, and
whether the containers are `running`. If a container is restarting, show its logs.

## Safety rules

- Do **not** run `docker compose down -v`, `docker volume rm`, `docker system prune`, or
  `docker image prune -a`. They can destroy data or other projects' images.
- Do **not** run the `init` service (`docker compose run --rm init`). It wipes and recreates all
  game data. Only do it if the user explicitly asks for a database reset and confirms.
- Do not touch `postgres`/`redis` during an update (`--no-deps` keeps them out of it).
- `--no-cache` makes a local build slow (Go module download, `npm ci`). That is expected; do not
  drop the flag unless the user asks for a quick/cached build.
- Old images become dangling after an update. Offer `docker image prune -f` (dangling only) to
  reclaim disk space, but only run it if the user agrees.
- Don't pass secrets (`SECRET_JWT`, `DB_PASSWORD`, `GOOGLE_OAUTH_CLIENT_ID`, ...) on the command
  line or echo them back in the chat; they live in `.env`.

## Quick reference (all from `$STRIKER_DIR`)

| Goal | Command |
|---|---|
| Update everything to the published images | `docker compose pull svc engine frontend && docker compose up -d --no-deps svc engine frontend` |
| Update only the backend | `docker compose pull svc engine && docker compose up -d --no-deps svc engine` |
| Update only the frontend | `docker compose pull frontend && docker compose up -d --no-deps frontend` |
| Rebuild everything from the checkout | `docker compose build --no-cache svc engine frontend && docker compose up -d --no-deps svc engine frontend` |
| Rebuild only the backend | `docker compose build --no-cache svc engine` |
| Rebuild only the frontend | `docker compose build --no-cache frontend` |
| Status | `docker compose ps` |
