# striker

Striker Football Manager - a browser-based football (soccer) management game. Each user manages
one team: set a lineup, buy and sell players on a transfer market, watch matches simulate
automatically on a schedule, compete in a knockout cup, and get promoted or relegated at the end
of a season.

- `backend/` - Go/Gin API, background simulation engine, and DB seeder. See `backend/README.md`
  for setup, Docker, and the full list of database migrations.
- `frontend/` - React/TypeScript/Vite client. See `frontend/README.md` for setup and Docker.

Each half has its own `docker-compose.yml` and can be run independently; point the frontend's
`VITE_API_ENDPOINT` (or `API_URI` in its Docker build) at wherever the backend ends up running.

See `HOWTO_RUN.md` for concrete run instructions (Docker quick start, env vars, local dev without
Docker, and pointing at a Postgres/Redis that already run elsewhere), and `ARCHITECTURE.md` for
an orientation to how the codebase is put together.

## What's implemented

- Match simulation: minute-by-minute engine covering goals, yellow/red cards, and injuries, run
  automatically on a schedule (also watchable live over WebSocket as a match finishes - see
  "Live match streaming" below).
- Lineup management, with suspended/injured players enforced as unavailable.
- Leagues with promotion/relegation between tiers at each season rollover, and a per-country
  single-elimination cup competition alongside the league season.
- Team finances: a budget, per-match wage bills, and a transaction ledger; a transfer market to
  list and buy players.
- Player progression: aging, retirement, and skill drift each season, backfilled by an academy
  so squads don't shrink over time.
- A notification feed (match results, promotion/relegation, transfers, cup results) and a
  season-recap view built from it.
- Live match streaming: a match's already-simulated events are paced out over a WebSocket to any
  connected viewer, so watching one in the UI looks like it's happening in real time.
- Admin/commissioner tooling (`ADMIN_EMAILS`-gated): a game-state overview, a user list, manual
  simulation/season-advance triggers, and manual budget corrections.
