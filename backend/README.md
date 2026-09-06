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

The seeded world spans four countries (`seededCountries` in `handlers/initializer.go`): Spain
(`es`), England (`en`), France (`fr`), and Germany (`de`) - each with its own three-tier league
structure and its own player/team name pool (`names_<code>`/`lastnames_<code>`, see
`resources/sql/init.sql` and `init19_multi_country_names.sql`). Spain's pool is a real ~29k-name
dump; the other three are curated common-name lists of about 60 first/last names each - real
names, just a much smaller pool, so expect more repetition across a large league than Spain's
teams see. A new Google sign-in is assigned to one of the four at random
(`randomSeededCountry`) - there's no sign-up step for a new user to pick one themselves.

Override any setting via a `.env` file in this directory (see `docker-compose.yml` for the full
list: `DB_*`, `SECRET_JWT`, `SESSION_SECRET`, `REALM`, `SESSION_TIMEOUT`, `SVC_PORT`). The
defaults are dev-only placeholders - set real secrets for anything beyond local development.

Set `ADMIN_EMAILS` (comma-separated) to grant commissioner/admin access - see "Admin tooling"
below - to those Google accounts. It's checked on every login, so adding an email takes effect
the next time that person logs in; no manual DB edit needed.

### CORS

The API only accepts cross-origin requests (including the OAuth login call) from an allowlisted
set of origins - `common.CorsAllowedOrigins`, always including the two deployed frontend domains
plus `http://localhost:3000` (the Docker/nginx-served frontend) and `http://localhost:5173`
(Vite's default `npm run dev` port). Running the frontend on a different port, or from a
different host entirely, needs `CORS_ALLOWED_ORIGINS` (comma-separated) set to that origin - e.g.
`CORS_ALLOWED_ORIGINS=http://localhost:4000`. Without it, the browser's preflight `OPTIONS`
request gets rejected (visible in this service's logs as a `403` on `OPTIONS`), and every API
call - login included - fails with a CORS error before it ever reaches a handler.

### Dev login

Signing in normally requires a real Google OAuth app (client ID/secret, a verified redirect
URL) - unavailable in most local/CI setups. Set `DEV_LOGIN_ENABLED=true` to open a shortcut:
`GET /v1/dev-login/users` lists a fixed roster (`common.DevUsers` - one admin plus a few plain
users) and `POST /v1/login` accepts `{"devEmail": "<one of those emails>", "provider": "dev"}`
in place of a Google `jwt`, signing in with **no credential check at all**. Each account is
auto-registered and assigned a team on first use, exactly like a first-time Google sign-in.
The frontend's login page shows a "Dev login" button per roster entry whenever the list comes
back non-empty, so it needs no separate frontend flag.

**Never set `DEV_LOGIN_ENABLED` on a real deployment** - anyone who can reach the API could sign
in as any of `common.DevUsers`, including the admin account. It's off by default and meant only
for local development and automated testing where a Google OAuth setup isn't available.

### Admin tooling

Anyone whose email is in `ADMIN_EMAILS` gets a set of commissioner endpoints under
`/v1/admin/*`: `GET overview` (season/user/team/match counts), `GET users` (every registered
user and their team), `POST simulate` (force a manual simulation pass instead of waiting for
the scheduler's next tick), `POST advance-season` (force a season-completion check),
`POST teams/:id/budget` (a manual budget correction, logged to that team's ledger like any
other transaction), `PUT transfer-window` (open/close the transfer and loan markets
globally - see "Transfer window" below), and `PUT match-schedule` (set the recurring
weekday/time-of-day fixtures are generated on - see "Match schedule" below).

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

### Match schedule

New league fixtures (a full season's worth, generated in one batch at world init or season
rollover - see `engine.GenerateMatches`) are dated according to one global recurring schedule:
a set of weekdays plus a time of day, e.g. "Saturdays at 14:00" or "every day at 16:00". An
admin sets it with `PUT /v1/admin/match-schedule`:

```json
{"weekdaysMask": 64, "hour": 14, "minute": 0}
```

`weekdaysMask` is a bitmask over the seven weekdays, bit 0 = Sunday through bit 6 = Saturday
(so Saturday alone is `1<<6 = 64`; every day is `127`); at least one bit must be set. The
current value comes back on `GET /v1/admin/overview` as `matchScheduleWeekdaysMask`/
`matchScheduleHour`/`matchScheduleMinute`. A single selected weekday gives one match round per
week; selecting every weekday gives one round per day. Changing the schedule only affects
fixtures generated **after** the change - it never reschedules matches that already have a
date (see `engine.ComputeFixtureDates`).

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
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init15_season_history.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init16_groups.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init17_notification_prefs.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init18_match_schedule.sql
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/init19_multi_country_names.sql
```

**Note:** init19 only adds the name-pool tables for the new countries (en/fr/de) - it doesn't
retroactively add their leagues/teams/players to a world that's already been seeded. Adding those
requires a full reseed (`docker compose --profile init run --rm init`, destructive - see above).

## Local development (without Docker)

```bash
cp app.env.sample app.env   # fill in real values
go run ./cmd --svc          # or --engine / --initialize
```
