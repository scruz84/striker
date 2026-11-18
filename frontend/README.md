# Striker frontend

React + TypeScript + Vite frontend for Striker Football Manager, talking to the Go backend
in `../backend`.

Stack: Vite, React 19, MUI, TanStack Query for server state, React Router.

## Local development

```bash
npm install
cp .env.development.local.sample .env.development.local   # set VITE_GOOGLE_OAUTH_CLIENT_ID
npm run dev
```

Requires the backend running locally (see `../backend/README.md`, or `docker compose up -d svc engine
postgres redis` from the repository root) on `http://localhost:1984` by default (configurable via
`VITE_API_ENDPOINT`).

## Checks

```bash
npm run lint
npx tsc -b
npm test
npm run build
```

## Docker

The production build is served by nginx from the `frontend` service of the root
`docker-compose.yml` (see `../HOWTO_RUN.md`), on `http://localhost:3000` (override with
`FRONTEND_PORT`). Its image is published as `ghcr.io/scruz84/striker-frontend` on every merge to
`main`; `docker compose up --build -d frontend` from the repository root builds it from this
directory's `Dockerfile` instead.

`API_URI`, `GOOGLE_OAUTH_CLIENT_ID`, `VAPID_PUBLIC_KEY` and the `LEGAL_*` settings are written into
`config.js` when the container starts (see `set-env.sh`), so one image fits every deployment - the
browser calls `API_URI` directly, so point it at wherever the backend is actually reachable from
your browser, not from inside this container.
