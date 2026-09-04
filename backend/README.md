# Striker backend

Go/Gin API service, background simulation engine, and DB seeder for Striker Football Manager.
See `cmd/striker.go` for the three run modes: `--svc`, `--engine`, `--initialize`.

## Docker

```bash
docker compose up --build
```

Starts Postgres, Redis, the API service (`svc`, published on `http://localhost:1984`) and the
background engine (`engine`, which simulates matches and rolls over seasons on a schedule).
Postgres is seeded from `resources/sql/*.sql` on first start.

The database itself is created empty by those `*.sql` files (schema + name-generation data) but
has no countries/leagues/teams/players yet. Seed (or reset) the game world explicitly - this is
**destructive**, it wipes and recreates all game data:

```bash
docker compose --profile init run --rm init
```

Override any setting via a `.env` file in this directory (see `docker-compose.yml` for the full
list: `DB_*`, `SECRET_JWT`, `SESSION_SECRET`, `REALM`, `SESSION_TIMEOUT`, `SVC_PORT`). The
defaults are dev-only placeholders - set real secrets for anything beyond local development.

Set `ADMIN_EMAILS` (comma-separated) to grant commissioner/admin access - see "Admin tooling"
below - to those Google accounts. It's checked on every login, so adding an email takes effect
the next time that person logs in; no manual DB edit needed.

### Admin tooling

Anyone whose email is in `ADMIN_EMAILS` gets a set of commissioner endpoints under
`/v1/admin/*`: `GET overview` (season/user/team/match counts), `GET users` (every registered
user and their team), `POST simulate` (force a manual simulation pass instead of waiting for
the scheduler's next tick), `POST advance-season` (force a season-completion check),
`POST teams/:id/budget` (a manual budget correction, logged to that team's ledger like any
other transaction), and `PUT transfer-window` (open/close the transfer and loan markets
globally - see "Transfer window" below).

The old unauthenticated `GET /v1/initData` reset endpoint now requires an authenticated admin
**once any user is registered** - before that (a genuinely fresh deployment, nobody to protect
yet) it still works with no login, so bootstrapping isn't broken. This only affects that HTTP
route; `docker compose --profile init run --rm init` (above) calls the same reset logic
directly via the CLI and was never affected either way.

### Transfer window

The transfer and loan markets share one global open/closed flag (`season.transfer_window_open`,
defaults to open) - listing, buying, and accepting a loan are all blocked while it's closed;
cancelling your own listing never is. There's no automatic schedule behind it (fixtures run on
independent per-league clocks, so there's no single "week of the season" to derive it from) -
an admin toggles it with `PUT /v1/admin/transfer-window` (`{"open": true|false}`), and
`GET /v1/transfers/window` reports the current state to anyone.

### Schema changes on an already-running database

`resources/sql/*.sql` only auto-applies on a **fresh** Postgres data volume (that's how the
official Postgres image's `docker-entrypoint-initdb.d` works). If you already have a running
database and pull in a change that adds a new `resources/sql/init*.sql` file (e.g.
`init2_league_tier.sql`, adding promotion/relegation), apply it manually once:

```bash
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init2_league_tier.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init3_player_unavailability.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init4_finances.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init5_transfers.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init6_player_age.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init7_cups.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init8_notifications.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init9_admin.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init10_stadium.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init11_loans.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init12_transfer_window.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init13_transfer_offers.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init14_formation.sql
```

## Local development (without Docker)

```bash
cp app.env.sample app.env   # fill in real values
go run ./cmd --svc          # or --engine / --initialize
```
