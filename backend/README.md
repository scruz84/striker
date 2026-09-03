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
```

## Local development (without Docker)

```bash
cp app.env.sample app.env   # fill in real values
go run ./cmd --svc          # or --engine / --initialize
```
