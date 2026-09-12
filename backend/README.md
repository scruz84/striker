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
`resources/sql/init.sql`). Spain's pool is a real ~29k-name
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
user and their team), `PUT users/:id/admin` (grant/revoke another user's admin access - a user
can't revoke their own), `POST simulate` (force a manual simulation pass instead of waiting for
the scheduler's next tick), `POST advance-season` (force a season-completion check),
`POST teams/:id/budget` (a manual budget correction, logged to that team's ledger like any
other transaction), `PUT transfer-window` (open/close the transfer and loan markets
globally - see "Transfer window" below), `PUT match-schedule` (set the recurring
weekday/time-of-day fixtures are generated on - see "Match schedule" below), and the world
management endpoints (world structure, recreating leagues, adding a country, per-league
schedules - see "World management" below).

The old unauthenticated `GET /v1/initData` reset endpoint now requires an authenticated admin
**once any user is registered** - before that (a genuinely fresh deployment, nobody to protect
yet) it still works with no login, so bootstrapping isn't broken. This only affects that HTTP
route; `docker compose --profile init run --rm init` (above) calls the same reset logic
directly via the CLI and was never affected either way.

### Live matches

Watching a not-yet-played match's page opens a WebSocket (`GET /v1/teams/match/:id/live`,
`.../cups/ties/:id/live`) that streams its result out minute by minute once the engine actually
simulates it (`engine.BroadcastMatchLive`) - the simulation itself is instant, computed and
persisted to Postgres in one go; the ~22-second "live" pacing is purely a cosmetic replay of an
already-known result for anyone watching. That replay is published over Redis pub/sub
(`common.LiveMatchHub`), not an in-process channel - the API server and the simulation engine are
separate processes (separate containers in `docker-compose.yml`, and still separate
`DatabaseContext`/hub instances even when both run together via `--svc --engine` in one process,
per "Local development" below), so only a hub both of them actually connect to can deliver a
message from one to the other. The frontend's `useMatchDetails` also polls every 5 seconds while
a match isn't played yet, as a backstop in case a socket message is ever missed (a dropped
connection, a missed reconnect) - it stops as soon as the match shows played, from whichever
source noticed first.

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

A single league can override that global default with its own weekday/time via
`PUT /v1/admin/leagues/:id/schedule` (same body shape as above) and go back to the default via
`DELETE /v1/admin/leagues/:id/schedule`. `GET /v1/admin/leagues` lists every league with its
effective schedule (the override if it has one, otherwise the global default) - see
`common.GetLeagueScheduleOverrides`, consulted by `engine.GenerateMatches` for every league's
fixtures independently.

### Promotion and relegation

How many teams move between each pair of adjacent tiers at season rollover
(`engine.ComputePromotionRelegation`) is admin-configurable via `PUT /v1/admin/promotion-relegation`:

```json
{"promotionSlotsPerGroup": 1, "relegationSlotsPerGroup": 2}
```

`promotionSlotsPerGroup` is how many of each lower-tier group's top teams are promotion
candidates; `relegationSlotsPerGroup` is how many of each upper-tier league's bottom teams drop.
Defaults (1 promoted, 2 relegated per group) match the values this was hardcoded to before it
became configurable, so an existing world sees no behavior change until an admin touches this.
Either can be set to 0 to turn that direction off entirely; both are capped at 10 (a league only
ever has a handful of teams). The current values come back on `GET /v1/admin/overview` as
`promotionSlotsPerGroup`/`relegationSlotsPerGroup`. It's still a straight swap with no play-off
model - when a tier has more relegation slots open than promotion candidates available (fewer
groups below than above), the extra bottom teams simply stay put. Only affects a season rollover
that hasn't happened yet; one already applied doesn't get undone.

### World management

Three more admin surfaces let a commissioner reshape the game world without a redeploy:

**World structure** (`GET`/`PUT /v1/admin/world-config`) is the tier/group/teams-per-league
shape used the next time leagues are (re)created for a country - see `common.TierConfig` and
`common.GetWorldConfig`/`SetWorldConfig`. It's just saved settings; on its own it doesn't touch
any league or team that already exists.

```json
{"tiers": [{"tier": 1, "name": "I", "groups": 1, "teamsPerLeague": 10},
           {"tier": 2, "name": "II", "groups": 4, "teamsPerLeague": 10}]}
```

`teamsPerLeague` must be even (`engine.randomWeekMatches2`'s round-robin fixture algorithm has
no bye slot for an odd team out) - see `common.ValidateWorldConfig` for the full bounds.

**Recreating leagues** (`POST /v1/admin/recreate-leagues`) applies the saved structure right
now: it wipes and regenerates every league, team, player and match for whichever countries are
currently active, the same way a fresh `--initialize` seeds the world, but **keeps every
registered login** - each existing user is handed a brand new team afterward
(`common.AssignTeam`), in their existing country. Team history, trophies and squads are still
lost; only the account itself (and its admin flag, and the app-wide match schedule) survives -
see `handlers.RecreateLeaguesKeepingUsers`. Creating hundreds of teams synchronously takes a
noticeable amount of time (tens of seconds for the default structure across every active
country) - the request blocks until it's done rather than returning early.

**Countries** (`GET /v1/admin/countries`, `POST /v1/admin/countries {"code": "it"}`) lists every
country the game engine knows how to generate a world for (`common.KnownCountries` - a
`names_<code>`/`lastnames_<code>` table pair plus team-naming data in `teamnames.go`), marking
which ones are already active, and activates one of the inactive ones: creates its leagues and
teams per the current world structure, generates this season's fixtures for just those new
leagues (`engine.GenerateMatchesForLeagues` - `engine.GenerateMatches` itself only fires when
*no* league anywhere has upcoming matches, which is never true once the world already exists),
and starts its cup (`engine.StartCupForCountry`) - all without touching any other country's
data. `es`/`en`/`fr`/`de` are active from a fresh `--initialize`; `it`/`pt` ship with their name
pools already loaded (see `resources/sql/init.sql`) but stay inactive until an admin adds one.

### League directory

`GET /v1/teams/leagues` lists every league in the game world, grouped by country - see
`common.GetLeagueDirectory`. It's how the frontend's Leagues page lets any signed-in manager
browse into a league they don't play in (via the existing `GET /v1/teams/division/:id/`, unchanged)
and see its standings and squads, not just their own division. Purely public information, same as
real football league tables - unlike a team's finances or a player's wage/contract, which stay
owner-only (see "Player contracts & free agency" below).

### Schema changes on an already-running database

`resources/sql/init.sql` is the full, consolidated schema for a brand-new database - it auto-
applies on a **fresh** Postgres data volume (that's how the official Postgres image's
`docker-entrypoint-initdb.d` works), and `--initialize` (see `handlers.
InitializeDatabase`/`common.EnsureSchema`) also applies it itself the first time it's pointed at
a database with no schema yet - so seeding a brand new, genuinely empty database needs nothing
beyond running `--initialize` (or the `init` profile below); this doesn't require Docker's own
auto-init to have run first. Either way, this only ever happens **once**: `init.sql` isn't
written to be safely re-run (its `CREATE TABLE` statements have no `IF NOT EXISTS`, and the
name-pool inserts have no uniqueness constraint stopping a second run from duplicating every
row), so both mechanisms only apply it when the schema doesn't already exist.

Every schema change that shipped before this file was consolidated is preserved, unchanged,
under `resources/sql/history/` (`init2_league_tier.sql` through
`init29_player_position_minutes.sql` at the time of writing) - kept for the record and for
upgrading a database that predates one of them, but never auto-applied (`EnsureSchema` only
lists the top-level `resources/sql/` directory, not `history/`).

If you already have a running, already-initialized database and pull in a change that adds a new
`resources/sql/initN_*.sql` file (numbered to continue after the highest one in `history/`),
apply just that new file manually once - `EnsureSchema` won't do this for you, since the schema
already exists as far as it's concerned:

```bash
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/initN_whatever.sql
```

If instead your database predates one of the migrations already folded into `init.sql`, apply the
matching file from `resources/sql/history/` the same way, e.g.:

```bash
docker compose exec -T postgres psql -U ${DB_USER:-striker} -d ${DB_DATABASE:-striker} < resources/sql/history/init2_league_tier.sql
```

**Note:** the multi-country migration only adds the name-pool tables for the new countries
(en/fr/de/it/pt) - it doesn't retroactively add their leagues/teams/players to a world that's
already been seeded. Adding those requires a full reseed (`docker compose --profile init run --rm
init`, destructive - see above).

### Team names

New teams get a randomly generated, realistic-sounding club name (`common.RandomTeamName`,
e.g. "Ashford United", "Real Villanueva") rather than a plain "<player name> Team" placeholder.
An owner can rename their own team exactly once, ever, via `PUT /v1/user/team/name`
(`{"name": "..."}`) - `common.RenameTeam` flips `teams.name_changed` atomically on the same
update, so a second attempt (or two racing attempts) fails with a 400. There's no admin reset
for that flag; it's a one-shot deal by design.

### Player contracts & free agency

Every player has a contract (`players.contract_years_remaining`) that counts down by one every
season (`common.AdvancePlayerContract`, run from `engine.ProgressPlayers` alongside aging). A
contract reaching zero releases the player to free agency - `team_id` goes to `NULL` - rather
than keeping them with their club forever; this is skipped for a player currently out on loan, so
the parent club's contract keeps ticking but doesn't lapse mid-loan. No contract, initial or
renewed, ever runs longer than `common.MaxContractYearsRemaining` (3 seasons). Any team can sign
a free agent for free (`GET /v1/free-agents`, `POST /v1/free-agents/:id/sign`) - no transfer fee,
just their wage, freshly negotiated from their current skills - and gets a new randomly-lengthed
contract (2-3 seasons, `common.RandomContractYears`).

An owner renews one of their own players' contracts by making them an offer
(`POST /v1/user/team/players/:id/renew-contract`, `{"wage": ..., "years": ...}`) rather than a
guaranteed instant action - the player evaluates it on the spot
(`common.MakeContractRenewalOffer`) against two expectations, both driven purely by their own
attributes, nothing else:

- **Wage**: at least their market rate (`common.PlayerWageFromSkills` - the same formula used to
  price a free agent's fresh contract), i.e. what a similarly-skilled player earns.
- **Duration**: at most `common.MaxAcceptableRenewalYears(age)` - the full 3-season cap under
  30, 2 seasons from 30 up to (not including) 33, 1 season at 33+ (mirroring
  `engine.RetirementEligibleAge`) - a player closer to retirement won't commit years in advance.

The response (`common.ContractRenewalOffer`) is a normal 200 either way and always reports the
player's actual `expectedWage`/`maxAcceptableYears`, whether the offer was accepted or not - a
decline changes nothing in the database, so the team can immediately submit a better offer
informed by those numbers. That retry loop is the "negotiation"; there's no separate
counter-offer step modeled server-side, no fee for making an offer, and no other condition
beyond the player's own acceptance. A Manager-tier AI team (see "AI teams" below) always offers
exactly what its own player is guaranteed to accept, so it never fails to retain someone it's
trying to keep.

`POST /v1/user/team/players/:id/release` terminates a contract early for a severance payment (ten
times the player's wage, debited from the team's budget and logged to the ledger). Both the
renewal offer and release endpoints reject a player currently out on loan - the borrowing team
holds `team_id` for the loan's duration, but the contract belongs to the parent club, not them.

A player's wage, contract length, and financial transfer history (`GET /v1/teams/player/:id/`'s
`history`) are only ever visible in full to the team that currently owns them -
`common.Player.RedactFinancials` zeroes them out (and the response's `history` comes back empty)
for anyone else, the same way `GET /v1/teams/team/:id/` only shows a squad's real wages/contracts
to its own owner. A free agent's wage stays visible to everyone, same as today, since it's exactly
the figure a prospective signer needs to see.

### Youth academy (cantera)

Every team also holds a small pool of unsigned young prospects (`common.CanteraSize`, currently
3), flagged `players.in_academy = true` and generated by `common.createCanteraProspects` with the
same skill/age ranges and trainer bonus as the automatic squad backfill below - but a prospect
draws no wage and has no contract, and is excluded from every senior-squad-facing query
(`GET /v1/teams/team/:id/`'s squad list, the wage bill, and the squad-size count that drives the
backfill) until an owner actually promotes them. The pool tops itself back up to `CanteraSize`
lazily, on every read and again right after a promotion (`common.topUpCantera`), so a team created
before this feature shipped gets one automatically the first time anyone asks for it - no backfill
migration needed.

`GET /v1/user/team/academy` lists the caller's own pool; `POST /v1/user/team/academy/:id/promote`
promotes one into the senior squad, granting them a fresh market wage and a randomly-lengthed
contract exactly as `SignFreeAgent` does for a free agent (`common.PromoteFromAcademy`) - mechanically,
promotion is their first professional signing. This is capped at once per real calendar week per
team (`common.CanPromoteFromAcademy`, using the same `StartOfWeek` helper as the friendly-match
weekly cap) via `teams.last_academy_promotion_at`; a second attempt the same week returns 409.

This is deliberately just the promotion mechanic, not a separate reserve/youth team - there's no
second squad, fixture list, or league for cantera prospects, only a holding pool a manager picks
from.

### Automatic squad backfill

Separately, `engine.topUpAcademy` (run from `engine.ProgressPlayers` at season end) keeps a team's
senior squad from falling below `common.SquadSize` by inserting new players directly onto the
squad via `common.CreateAcademyPlayers` - immediately fieldable, with a real wage and contract, no
manual step involved. This safety-net mechanism is unrelated to the cantera pool above and
unaffected by it.

### Player birth country

`players.birth_country` records the country whose name pool generated the player (see
`common.doCreatePlayers`), independent of `team_id`'s current country - the transfer market has no
country restriction, so a player can end up playing anywhere regardless of where they were
"born". Set once at creation, never touched again by a transfer, loan, or release. Nullable, since
it isn't knowable for anyone created before `resources/sql/init30_player_birth_country.sql` (those
rows were backfilled with their then-current team's country as the best available guess).

### OpenTelemetry metrics

Both `svc` and `engine` can export runtime metrics via OpenTelemetry - **off by default**, so
nothing changes for anyone who doesn't set it up. Turn it on with two environment variables:

```bash
OTEL_METRICS_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318   # your OTel Collector, Grafana Agent, etc.
```

See `common.InitMetrics` for the full mechanism. A few notes:

- The exporter is OTLP/HTTP and otherwise follows the [standard OpenTelemetry environment
  variables](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/) -
  `OTEL_EXPORTER_OTLP_HEADERS` for auth, `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` to target metrics
  at a different endpoint than traces/logs, etc. There's no code to change for any of that; the
  exporter library reads them itself.
- `OTEL_SERVICE_NAME` overrides the `service.name` resource attribute this process reports
  (default: `striker-svc`, `striker-engine`, or `striker-combined` for `--svc --engine` in one
  process - see `cmd/striker.go`). Set it if you're running `svc` and `engine` as separate
  containers (the normal Docker Compose setup) exporting to the same collector, so their metrics
  don't get attributed to the same series.
- Metrics export on a periodic timer (the SDK default is every 60 seconds) and flush once more
  on a clean shutdown (SIGINT/SIGTERM).
- Every instrument is a package-level var in `common/telemetry.go` (`common.Meter`,
  `common.HTTPRequestsCounter`, etc.) - safe to call from anywhere in the codebase whether or not
  export is actually enabled, since otel's default global MeterProvider is a documented no-op
  until `InitMetrics` installs a real one.

What's actually instrumented, chosen for being the handful of numbers most worth watching in
production rather than exhaustive coverage of every code path:

- **HTTP requests** (`striker.http.requests`, `striker.http.request.duration`) - every request
  `svc` handles, labeled by method, matched route (not the raw path, so `/v1/teams/player/:id/`
  stays one series regardless of which id), and status code. See the `metricsMiddleware` in
  `service/fmservice.go`.
- **Matches simulated** (`striker.matches.simulated`) and **goals scored**
  (`striker.goals.scored`) - every league match, cup tie, and friendly the engine finishes
  simulating, labeled by `type`. The core signal that the simulated world is actually moving.
- **Season rollovers** (`striker.season.rollovers`) - a rare, high-significance event (aging,
  promotion/relegation, new fixtures) worth seeing happen in a dashboard, not just in logs.
- **Transfers** (`striker.transfers.completed`, `striker.transfers.volume`) and **free agent
  signings** (`striker.free_agents.signed`) - the in-game economy's activity and the money moving
  through it.
- **Logins** (`striker.logins`, labeled by `provider`) and **new user registrations**
  (`striker.users.registered`) - the simplest signal of how much the game is actually being
  played.
- **Notifications sent** (`striker.notifications.sent`, labeled by `type`) - a proxy for how much
  is happening across every user's game at once.

## Local development (without Docker)

```bash
cp app.env.sample app.env   # fill in real values
go run ./cmd --svc          # or --engine / --initialize / --svc --engine
```

`--svc` and `--engine` can be combined to run the API server and the match-simulation/season-
rollover scheduler in one process - handy for local development so you don't need two terminals.
Docker Compose still runs them as separate containers (see `docker-compose.yml`), which works
exactly the same way.
