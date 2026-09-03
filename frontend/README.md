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

Requires the backend running locally (see `../backend/README` / `../backend/docker-compose.yml`)
on `http://localhost:1984` by default (configurable via `VITE_API_ENDPOINT`).

## Checks

```bash
npm run lint
npx tsc -b
npm test
npm run build
```

## Docker

```bash
docker compose up --build
```

Serves the production build via nginx on `http://localhost:3000` (override with `FRONTEND_PORT`).
`API_URI` and `GOOGLE_OAUTH_CLIENT_ID` are injected into `config.js` at container start (see
`set-env.sh`) - the browser calls `API_URI` directly, so point it at wherever the backend is
actually reachable from your browser, not from inside this container.
