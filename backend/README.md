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

The seeded world spans four countries by default (`common.DefaultSeededCountries`): Spain
(`es`), England (`en`), France (`fr`), and Germany (`de`) - each with its own five-tier league
pyramid (1/2/4/8/16 groups of 10, see "Promotion and relegation") and its own player/team name pool (`names_<code>`/`lastnames_<code>`, see
`resources/sql/init.sql`). Spain's pool is a real ~29k-name
dump; every other country's pool has at least 300 first names and 500 surnames, written as text
sources under `utils/namepools/<code>/{first,last}.txt` and turned into idempotent SQL
(`resources/sql/namepools/init91_name_pool_<code>.sql`) by `utils/gen_name_pools.py`. A new Google sign-in is assigned to one of the seeded countries at random
(`randomSeededCountry`) - there's no sign-up step for a new user to pick one themselves.

Override which countries get seeded with the comma-separated `SEEDED_COUNTRIES` environment
variable (e.g. `SEEDED_COUNTRIES=es,en,it,pt`) - each code must already be a known country (see
`common.KnownCountries`), i.e. have its own `names_<code>`/`lastnames_<code>` table pair and
team/stadium naming data; an unknown code is skipped with a logged error rather than failing the
whole list. Unset or entirely invalid falls back to the four defaults above.

Override any setting via a `.env` file in this directory (see `docker-compose.yml` for the full
list: `DB_*`, `SECRET_JWT`, `SESSION_SECRET`, `REALM`, `SESSION_TIMEOUT`, `SVC_PORT`). The
defaults are dev-only placeholders - `--svc` **refuses to start** with them (or any `SECRET_JWT` /
`SESSION_SECRET` shorter than 32 characters) and without `GOOGLE_OAUTH_CLIENT_ID`, unless
`DEV_LOGIN_ENABLED=true` (see "Dev login"), in which case it starts and logs each problem at ERROR
(`common.CheckDeploymentSecurity`). Generate secrets with `openssl rand -base64 48`.

### Security

- **Identity comes only from the verified session.** `handlers.GetUserIdFromCookie` reads the
  claims the JWT middleware already verified (`ginjwt.ExtractClaims`); on a route without that
  middleware, or with a forged/unsigned/expired cookie, there is no user.
- **Google sign-in** validates the ID token with `idtoken.Validate`: signature, expiry, issuer and
  `aud` = `GOOGLE_OAUTH_CLIENT_ID` (the same client id the frontend is built with), and requires
  `email_verified`. Anything else is a 403 and creates no account. Accounts only come from a
  successful sign-in - there is no register endpoint.
- **CSRF**: every `POST`/`PUT`/`PATCH`/`DELETE` except `POST /v1/login` must carry the header
  `X-Striker-Client: web` (403 otherwise; `service.RequireClientHeader`). The bundled frontend
  (`src/api/client.ts`) always sends it; a script calling the API with curl has to add it too.
  The header is in the CORS allowed headers, so only the allow-listed origins can send it
  cross-site.
- When dev login is on, startup logs an ERROR and the frontend shows a warning strip.

See `HOWTO_RUN.md`'s "Production checklist" for what a public deployment must set.

### Account lifecycle

Schema `resources/sql/init92_user_account_lifecycle.sql` (applied on startup like every `initN_*`;
existing users start with `last_seen_at = now()` and no accepted terms).

- **Activity**: any authenticated request or sign-in updates `users.last_seen_at` (at most hourly
  per replica, `handlers.ActivityMiddleware`). A valid token of a deleted user gets 401.
- **Engine jobs** (`engine/accountlifecycle.go`): daily 03:17 UTC `account_lifecycle_daily` - warn
  owners idle 46/53 days (in-app + push), release the club of an owner idle 60 days, warn at 351 and
  erase at 365 days; every 15 minutes `reset_released_teams` - reset each released club (not while
  a match is in simulation or between 2 h ago and 30 min ahead). Thresholds are constants in
  `common/accounts.go`.
- **Endpoints**: `GET /v1/user/account`; `DELETE /v1/user/account` with `{"confirm": "<club name>"}`
  (or `DELETE`/`BORRAR` without a club) answers 204 and clears the cookie;
  `GET /v1/user/account/export` downloads `striker-my-data.json`; `POST /v1/auth/accept-terms` with
  `{"version": n, "ageConfirmed": true}` (`/auth/me` carries `termsAccepted`, `termsVersion`,
  `releasedTeamName`; onboarding answers 400 until the terms are accepted);
  `DELETE /v1/admin/users/:id` (admins; 403 for one's own id). Team responses no longer include an
  owner name.
- **Metric**: `striker.teams.reset{reason}` (inactivity, self_deletion, admin_deletion, erasure).
- **Legal identity** (`LEGAL_*`) is a *frontend* runtime property, see `HOWTO_RUN.md`.

Set `ADMIN_EMAILS` (comma-separated) to grant commissioner/admin access - see "Admin tooling"
below - to those Google accounts. It's checked on every login, so adding an email takes effect
the next time that person logs in; no manual DB edit needed.

### CORS

The API only accepts cross-origin requests (including the OAuth login call) from an allowlisted
set of origins - `common.CorsAllowedOrigins`, always including `http://localhost:3000` (the
Docker/nginx-served frontend) and `http://localhost:5173` (Vite's default `npm run dev` port).
A deployment, or a frontend on a different port, needs `CORS_ALLOWED_ORIGINS` (comma-separated)
set to its origin(s) - e.g. `CORS_ALLOWED_ORIGINS=https://striker.example.com` or
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

`POST /v1/admin/initialize` wipes and reseeds the whole game (`handlers.Initialize`) and is the
only HTTP route that does - behind the JWT and admin middleware like the rest of this group. The
old unauthenticated `GET /v1/initData` no longer exists (404). An empty database is bootstrapped
with the CLI instead: `docker compose --profile init run --rm init` (or `go run ./cmd
--initialize`).

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

#### Live replay recovery

The effects that must not show before a match has nominally finished - the availability countdown
and new injuries/suspensions, skill growth and the result notification - are stored as a pending
record (`pending_post_match_effects`, payload `common.PostMatchPayload`) in the same transaction as
the result (`common.SaveLeagueMatchResultTx` and its friendly/cup counterparts). The replay's
completion claims the record with `DELETE ... RETURNING` and applies it
(`engine.completePostMatchEffects`); `LiveBroadcastRecoveryJob` (every 5 minutes) applies any record
still pending once kickoff + the 2-hour live-broadcast window has passed
(`engine.ApplyDuePostMatchEffects`), so an engine restart during the replay (up to 108 minutes) no
longer loses them. Whoever deletes the row applies it, so the effects are applied at most once.

#### Added time

Each half gets 1-5 added minutes, worked out from the time its regulation minutes lost to goals,
substitutions, injuries, cards and penalties (`engine/stoppage_time.go`, openspec/specs/stoppage-time)
and played with the normal per-minute stages (`engine.playMatchMinutes`). A minute in added time is
stored as its clock minute (45 or 90) plus `added_minute`, an `ADDED_TIME` event (type 9) records the
board, and each fixture keeps `first_half_added_time` / `second_half_added_time`
(`resources/sql/init107_stoppage_time.sql`). The replay plays the added minutes at one per real minute;
the half-time break absorbs the first half's, so minute 46 still starts 59 minutes after kickoff and
the final whistle comes at most 108 minutes after kickoff - `engine.TestLongestBroadcastFitsLiveWindow`
keeps that at least 5 minutes inside the 2-hour live-broadcast window every time-based spoiler gate
relies on.

A simulation that fails (an error or a panic) counts an attempt, releases its claim and stores the
reason (`simulation_attempts`, `last_simulation_error`); it is retried no sooner than 10 minutes
later, and after 3 attempts `FinishStuckSimulationsJob` settles it 0-0 (`engine.SettleDefaultResults`).
The admin overview lists the failing matches and the "finish stuck matches" action runs the same
default settlement and re-simulation on demand - and, without waiting out the 10 minutes, gives
each failed match whose kickoff has passed one last attempt (`engine.RetryFailingFixturesNow`), so
one that fails again is settled 0-0 in the same press.

### Transfer window

The transfer and loan markets share one global open/closed flag (`season.transfer_window_open`,
defaults to open) - listing, buying, and accepting a loan are all blocked while it's closed;
cancelling your own listing never is. There's no automatic schedule behind it (fixtures run on
independent per-league clocks, so there's no single "week of the season" to derive it from) -
an admin toggles it with `PUT /v1/admin/transfer-window` (`{"open": true|false}`), and
`GET /v1/transfers/window` reports the current state to anyone.

A player's listings and offers never outlive his time at the club they name. Whenever he leaves -
sold, released (`ReleasePlayer`), out of contract (`AdvancePlayerContract`), signed on an accepted
offer, a paid release clause or a pre-contract - his transfer listing, loan listing and pending
offers go in the same transaction (`common.clearPlayerMarketTx` and its equivalents). As a second
line of defence, `ExecuteTransfer` and `AcceptLoan` re-read the player's club under lock and,
when it is no longer the seller's or lender's, delete the stale listing and answer
`ErrListingNotFound` / `ErrLoanListingNotFound` without moving money or the player.
`init113_drop_orphan_market_rows.sql` removed the stale rows written before this rule.

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

A country's leagues form a pyramid with a fixed rule (openspec/specs/league-pyramid): tier 1 has
one group and every tier below it exactly twice the groups of the one above, all with the same
teams per group (default 1/2/4/8/16 groups of 10, `common.DefaultWorldTiers`). The zones of every
group follow from it (`common.PyramidZones`): below tier 1 the winner goes up and the runner-up
plays the promotion playoff; above the bottom tier the last two go down and the two above them
play the playoff. There is no admin setting for the slot counts any more.

- **Playoff.** Once a country's last league broadcast ends, `advanceCountrySeasonIfComplete`
  creates the season's playoff (`engine.startPlayoff`): one `cups` row of `kind = 'playoff'`
  (`cups_playoff_once` makes it once per country and season) with a single-match tie per pairing,
  the upper-tier team at home (`engine.ComputePlayoffPairings`). The ties are dated the day after
  the last league matchday at the upper tier's league kickoff (`engine.playoffKickoff`) and are
  simulated, broadcast and lineup-edited exactly like cup ties, but `AdvanceCups` never advances
  them and they pay no cup round prize. The season rolls over only when every tie is played and
  broadcast; then `ComputeSeasonMoves` applies the direct moves plus each tie's outcome (a swap when
  the lower team won) and `season_history.via_playoff` records the playoff moves.
- **Legacy pyramids.** A country whose active leagues break the rule (e.g. the old 1/4/16) plays
  no playoff and keeps the old one-for-one swap for that season (`computeLegacyMoves`).
- **Restructure.** At rollover, a country whose active leagues differ from the saved world
  structure is reshaped into it (`engine.restructureIfNeeded`): every team is ranked down the
  pyramid, tiers are filled top-down in snake order (teams that do not fit drop a tier), the bottom
  tier is topped up with new CPU teams (`common.CreateTeam`), and leagues the structure no longer
  has are archived (`leagues.archived_season`), never deleted.
- **Resuming.** The rollover runs as numbered steps (aging and contracts, loans, promotion/relegation
  with prizes, restructure, cup, standings reset, season bump, fixtures), each recorded in
  `countries.rollover_season`/`rollover_step`/`rollover_state` as it completes
  (`init116_resumable_season_rollover.sql`). A run that stops halfway - a restart, an error between
  steps - is resumed at the next step on the following tick, with the standings and moves stored
  in the marker; the season bump records itself in the same `UPDATE`, and the fixtures clear the
  marker (`engine.runSeasonRollover`, openspec/specs/engine-replicas "Resumable season rollover").

### League kickoffs

Each country has a league kickoff on its own clock (`countries.league_kickoff_minute`, default
14:00): tier 1 kicks off then and every lower tier 15 minutes later (`common.EffectiveLeagueSchedule`),
dated in the country's time zone so the local time holds across daylight-saving changes. The
app-wide match schedule only sets the match days now. A league can still get its own override
(`PUT /v1/admin/leagues/:id/schedule`), and `PUT /v1/admin/countries/:code/kickoff` changes a
country's kickoff and re-dates its unplayed matches.

### World management

Three more admin surfaces let a commissioner reshape the game world without a redeploy:

**World structure** (`GET`/`PUT /v1/admin/world-config`) is the tier/group/teams-per-league
shape new countries are created with, and the one every active country is restructured into at
its own next season rollover (see "Promotion and relegation") - see `common.TierConfig` and
`common.GetWorldConfig`/`SetWorldConfig`. Saving it changes nothing until then.

```json
{"tiers": [{"tier": 1, "name": "I", "groups": 1, "teamsPerLeague": 10},
           {"tier": 2, "name": "II", "groups": 2, "teamsPerLeague": 10}]}
```

It must follow the pyramid rule (tier 1 one group, every tier twice the groups of the one above,
the same even `teamsPerLeague` of at least 8 everywhere - `engine.randomWeekMatches2`'s round-robin
has no bye slot for an odd team out) and hold at least as many teams as every active country
already has - see `common.ValidateWorldConfig` / `ValidateWorldConfigFitsTeams`.

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
`names_<code>`/`lastnames_<code>` table pair plus team-naming data in `teamnames.go`; the 211 FIFA
members), marking
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
`init58_stadium_capacity_default.sql` at the time of writing) - kept for the record and for
upgrading a database that predates one of them, but never auto-applied (`EnsureSchema` only
lists the top-level `resources/sql/` directory, not `history/`).

If you already have a running, already-initialized database and pull in a change that adds a new
`resources/sql/initN_*.sql` file (numbered to continue after the highest one in `history/`), you
don't need to do anything: both `--svc` and `--engine` call `EnsureSchema` on every startup, and
against a database that already has a schema it re-applies just the numbered `initN_*.sql`
migrations (skipping `init.sql` itself and the name-pool dumps, which are only safe to run once -
see the doc comment on `EnsureSchema`). Every migration file is written to be safely re-run
(`CREATE TABLE`/`ADD COLUMN`/`CREATE INDEX ... IF NOT EXISTS`, `UPDATE ... WHERE x IS NULL`), so
restarting the containers after pulling a branch update (`docker compose up -d --build`) is
enough to pick up a new one - no manual `psql` step required. If you'd rather apply it yourself
without restarting anything, that still works too:

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
- **No pay cut**: never below their current wage while 2 or more seasons are left, and in the
  last season 75% (Naive) to 95% (Mastermind) of it by negotiation tier
  (`common.PersonalTermsExpectation`, `LastSeasonFloorShare`). A transfer offer's personal terms follow
  the same rule with the whole current wage as the floor; a free agent has no floor.
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

**Negotiation intelligence** (`common/playerintelligence.go`): every player has a hidden tier,
Naive, Casual, Aware, Shrewd or Mastermind. It scales how much of his skill trend (the average of
his last 5 progression runs, `common.PlayerSkillTrend`) he turns into a raise or a longer term, and
sets his last-season wage floor, his pre-contract premium and his clause discount (see above).
The tier is private: no API response gives it to a manager, not even after negotiating with him
(the old `negotiation_insights` table is dropped by `init109_drop_negotiation_insights.sql`).

**Pre-contracts** (`common/precontracts.go`, `resources/sql/init104_pre_contracts.sql`): once a
player's league has played half its season, any other club may offer a player in the last season
of his contract a deal for next season (`POST /v1/pre-contracts` `{playerId, wage, years}`, no fee).
He only considers offers of at least his move premium x his expected wage (1.05 for a Naive
player up to 1.20 for a Mastermind, `common.PreContractPremium`), and decides 72 hours after the first
one (`common.DecidePreContracts`, run from `OfferResponseJob`), signing the best. Until then his
club is told each offer's terms and keeps him by renewing at the best offer / his premium (the renewal
expectation is raised to that bar); a renewal, sale or loan cancels the offers. A signed player
can't be renewed, released, listed or offered for, and joins the buyer at his country's next
rollover (`AdvancePlayerContract`, transfer kind `PLAYER_TRANSFER_PRE_CONTRACT`). Another club's
player in his last season carries the public `contractExpiring` flag. CPU Managers and Directors
make pre-contract offers and counter the ones they receive (see `engine/aitransfers.go`
`preContractCandidates`, `engine/aiteams.go` `answerPreContractOffers`).

**Release clauses** (`common/releaseclauses.go`, `resources/sql/init105_release_clauses.sql`): a
renewal, a free-agent signing, a transfer offer's personal terms and a pre-contract may carry an
optional, public release clause (`releaseClause` in the request body) of at least 1.5 x the
player's market value on a 3-season contract. The player asks 5% (Naive) to 15% (Mastermind)
less wage for a clause of at most 2 x that value, half of that up to 4 x (`ClauseDiscountFor`). Any other solvent club may pay it 14 days or more after
the contract was agreed (`POST /v1/release-clauses` `{playerId, wage, years}`) and sign him at
once without his club's consent, on personal terms he accepts (`common.PayReleaseClause`,
transfer kind `PLAYER_TRANSFER_RELEASE_CLAUSE`). CPU Managers set clauses on bench renewals,
Directors on non-key players, and Directors pay clauses up to 1.5 x their valuation, at most one
per pass and never twice on the same human club within 7 days.

`POST /v1/user/team/players/:id/release` terminates a contract early for a severance payment (ten
times the player's wage, debited from the team's budget and logged to the ledger). Both the
renewal offer and release endpoints reject a player currently out on loan - the borrowing team
holds `team_id` for the loan's duration, but the contract belongs to the parent club, not them.

A player's contract length and financial transfer history (`GET /v1/teams/player/:id/`'s
`history`) are only ever visible in full to the team that currently owns them -
`common.Player.RedactFinancials` zeroes them out (and the response's `history` comes back empty)
for anyone else, the same way `GET /v1/teams/team/:id/` only shows a squad's real contract terms
to its own owner. Wage is the one exception, on both endpoints: it's restored right after
`RedactFinancials` zeroes it, so it stays visible on someone else's rostered player too, giving a
visitor a rough sense of what signing them (or fielding a rival's whole squad) would cost - a free
agent's wage was already visible to everyone, since it's exactly the figure a prospective signer
needs to see. `GET /v1/teams/team/:id/` makes the same exception for each player's `averageSkill`
(a single 0-99 aggregate, unlike the seven individual skill fields `RedactSkills` zeroes) and also
reports the team's real classification position and public stadium summary (name, capacity, most
recent home match with attendance) regardless of who's asking.

A player's transfer history is a separate, public thing: `GET /v1/teams/player/:id/`'s `transfers`
is read from the `player_transfers` table (`init78_player_transfers.sql`), never redacted. One
structured row per move - `from_team_id` (NULL = came from free agency), `to_team_id` (NULL =
released), `fee`, `kind` (`common.PLAYER_TRANSFER_*`: bought listing, accepted offer, free agent,
loan, buy option, release) - written by `common.RecordPlayerTransfer` inside the same transaction
as the ledger rows of `ExecuteTransfer`, `AcceptOffer`, `SignFreeAgent` (the premium paid to the
last club, if any, is the fee and that club the origin), `AcceptLoan`, `ExerciseLoanBuyOption`,
`ReleasePlayer` and `AdvancePlayerContract` (contract expiry). The migration backfills the older
history from `team_transactions` and `player_club_history`; the old ledger can't tell a bought
listing from an accepted offer, so those come back as the generic `PLAYER_TRANSFER_PERMANENT`.

### Player aging and decline

A rostered player's core skills drift match-by-match, not once a season: right after every
competitive match (league or cup) they feature in, only whichever attributes are relevant to
whichever position they played the most minutes at *in that match* have a chance to move
(`engine.driftSkillsForPosition`), by an amount that trends with age (`engine.ageTrendFor`):
upward below 24, flat in their prime (24-29), downward from 30 on. A hired trainer's growth bonus
(`engine.trainerBonus`) only ever applies while that trend is still flat or rising - once a
player is past 30, a great coach can no longer reverse the decline, only fail to slow it down, so
a veteran's skills merely hold steady at best rather than climbing indefinitely no matter how
good their coaching is (`engine.driftSkill`). Each relevant skill's chance of actually drifting
that match scales with minutes played in it, up to `engine.MatchSkillGrowthMinutesReference` (900
- the same 10-full-matches budget a season-end drift used to spend all at once, just spread out
match by match now, so a single game rarely moves more than a skill or two). See
`openspec/specs/player-development` for the full mechanism. On top of that,
`engine.ShouldRetire` forces every player out of the game eventually regardless of skill: never
below 33, a rising chance each year after that, and always by 40 - that part is still checked
once a season, in `engine.ProgressPlayers`.

### Free agency has a time limit

`engine.ProgressPlayers` only iterates players who belong to a team, so a free agent (an
unowned player - see "Player contracts & free agency" above) would otherwise never age or retire
at all. `engine.ProgressFreeAgents`, run at the same point in the season rollover, closes that
gap: every free agent still ages a year and can retire on the same age-based odds as anyone
else, and separately, `players.free_agent_seasons` counts how many consecutive seasons they've
now gone without a club - reset to 0 the moment they're signed (see `SignFreeAgent`) - forcing
retirement once it reaches `engine.FreeAgencyRetirementSeasons` (2) even for a player who's
otherwise still young. Free agents don't have their skills drift at all while unsigned - no club
means no matches and no coach to develop or decline them, only the clock ticking.

Either way, a free agent forced out by this doesn't just vanish: `common.RetireFreeAgentAsTrainer`
converts them into a fresh, unemployed trainer candidate on the open market first, keeping their
own name - their playing career becomes a coaching one instead. The candidate's skill is the
average of their seven core attributes (`trainerSkillFromPlayerSkills`), clamped to the normal
`TrainerSkillMin`/`TrainerSkillMax` range.

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
"born". Set once at creation, never touched again by a transfer, loan, or release.

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
- Every metric also carries a `service.instance.id` resource attribute set to the process's
  hostname - unique per replica with no extra configuration on every deployment shape that
  actually scales `svc`/`engine` to several nodes (Docker Compose's `--scale`, a Kubernetes
  Deployment, each container/pod already gets its own hostname). Against a backend that ingests
  OTLP natively (Prometheus >=2.47's OTLP receiver, Mimir, Grafana Cloud), this becomes the usual
  `instance` label on every series, so you can check whether work is actually spread evenly
  across replicas - see the "Per node" row in the Grafana dashboard below.
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
  `service/fmservice.go`. A websocket upgrade (the live match viewer) is counted but records **no
  duration**: the connection stays open for the whole match and would bury the latency of real
  requests, so `striker.http.request.duration` never has a `/live` route (the dashboard also
  filters `route!~".*/live"` for series exported before that change).
- **Matches simulated** (`striker.matches.simulated`) and **goals scored**
  (`striker.goals.scored`) - every league match, cup tie, and friendly the engine finishes
  simulating, labeled by `type`. The core signal that the simulated world is actually moving.
- **Season rollovers** (`striker.season.rollovers`) - a rare, high-significance event (aging,
  promotion/relegation, new fixtures) worth seeing happen in a dashboard, not just in logs.
- **Transfers** (`striker.transfers.completed`, `striker.transfers.volume`) and **free agent
  signings** (`striker.free_agents.signed`) - the in-game economy's activity and the money moving
  through it. Counted for both ways a transfer can complete: a listed player bought outright
  (`ExecuteTransfer`) or a negotiated offer accepted (`AcceptOffer`) - see
  `common.RecordTransferCompleted`.
- **Transfer market size** (`striker.transfer_market.listed_players`) and **loan market size**
  (`striker.loan_market.listed_players`) - observable gauges reporting how many players are
  listed for transfer/loan right now, plus **active loans** (`striker.loans.active`) - how many
  players are currently out on loan. Unlike the counters above, these are point-in-time snapshots
  (re-queried on every export tick), not cumulative totals - see `common.RegisterMarketGauges`.
- **Total money in the economy** (`striker.economy.total_budget`) - every team's budget summed
  together, right now. Also an observable gauge: a transfer or accepted offer just moves money
  from one team's budget to another's (`striker.transfers.volume` tracks that flow), so this total
  only actually moves when money is created or destroyed - season prize payouts, sponsor deals,
  wage/stadium upkeep. Can go negative in aggregate if enough teams are in debt (see
  `ErrTeamInsolvent`).
- **Transfer offers** (`striker.offers.made`, and `striker.offers.resolved` labeled by `outcome`:
  `accepted`/`rejected`/`withdrawn`, plus the counter-offer round's
  `counter_accepted`/`counter_declined`/`counter_expired`) - the negotiated-offer side of the
  transfer market, separate from the fixed-price listings above.
- **Loans** (`striker.loans.listed`, `striker.loans.accepted`) - players newly offered out on
  loan, and loan listings a borrowing team took up.
- **Academy promotions** (`striker.academy.promotions`) - youth academy prospects promoted to the
  senior squad (`common.PromoteFromAcademy`).
- **CPU market actions** (`striker.ai.market_actions`, labeled by `action`: `signed_free_agent`,
  `bought_listed`, `offer_made`, `offer_expired`, `listed_surplus`, `listed_to_fund`,
  `listed_in_crisis`, `released_veteran`) - what
  CPU-managed teams do on the transfer market, so a day's activity can be read off this series.
- **CPU levels** (`striker.ai.teams`, `striker.ai.average_budget`, `striker.ai.insolvent_teams`,
  `striker.ai.points_per_match`, labeled by `ai_level`: `human`, `caretaker`, `manager`,
  `director`) - how many clubs each kind of manager runs, their average budget, how many are
  insolvent and the league points per match they take this season, from one grouped query per
  export tick (`common.GetAiLevelStats`, also shown in the admin overview). Engine-only, read
  with max(). `AI_LEVEL_SIM=1 go test ./engine -run TestAiLevelCalibration -v` plays each level
  against a Caretaker with identical squads to check the levels are apart.
- **Logins** (`striker.logins`, labeled by `provider`) and **new user registrations**
  (`striker.users.registered`) - the simplest signal of how much the game is actually being
  played.
- **Notifications sent** (`striker.notifications.sent`, labeled by `type`) - a proxy for how much
  is happening across every user's game at once.
- **Per-process health** (`striker.process.goroutines`, `striker.process.memory.heap_alloc`,
  `striker.process.cpu_time`, `striker.process.uptime`) - each replica's own goroutine count, heap
  memory in use, cumulative CPU time, and seconds since it started, registered unconditionally for
  every process (unlike the market/economy gauges above, there's no DB-wide value here to
  double-count - see `common.registerProcessHealthGauges`). CPU time is Linux-only (reads
  `/proc/self/stat`); the other three work everywhere. Paired with `service.instance.id`, these
  are what the "Per node" dashboard row's goroutines/memory/CPU/uptime panels graph to compare
  replicas side by side.

- **Economy flow** (`striker.economy.flow`, labeled by `reason` and `direction`) - the money
  credited (`in`) and debited (`out`) through the ledger, `|amount|` per committed
  `team_transactions` row, `reason` being the snake_case name of its `TRANSACTION_*` constant
  (`wages`, `sponsorship`, `ticket_revenue`, `prize_money`, `transfer_in`, ...; anything unmapped is
  `other`, and a test keeps that from happening). It is recorded only after the transaction commits
  (`common.LedgerFlows`, written through `insertLedgerRow`), so a rolled-back write is never
  counted. Where `striker.economy.total_budget` shows the total, this shows where it comes from and
  goes. Every process that writes the ledger exports it: read it with `sum`.
- **Engine jobs** (`striker.engine.job.runs`, `striker.engine.job.duration`,
  `striker.engine.job.last_run`, labeled by `engine_job`, and `striker.engine.job.claims`, labeled by
  `engine_job` and `outcome` = `won`/`lost`) - every scheduled run of each engine job on each
  replica, its duration in seconds, the unix time it last finished, and who won each once-per-period
  claim (`claimRun`). The label is `engine_job`, not `job`, because an OTLP backend turns the
  resource's `service.name` into `job` and would overwrite a data-point label of that name. A run
  that loses its claim still counts as a run (the trigger fired there). Registered by the engine
  only; `sum` the counters, and read `last_run` with `max` (`time() - max by (engine_job)(...)` is
  the job's freshness). See `engine.scheduleJob`.
- **Simulation backlog** (`striker.matches.overdue`, `striker.matches.in_simulation`, labeled by
  `type` = `league`/`cup`/`friendly`) - observable gauges, DB-wide like the market gauges, so
  registered by the engine only and read with `max`, never `sum`. Overdue counts unplayed matches
  with both teams set whose kickoff passed more than 5 minutes ago (a cup bye never counts); in
  simulation counts the ones an engine replica has claimed and not finished. One query per export
  tick, over the partial indexes of `resources/sql/init85_observability_indexes.sql`. A line that
  stays above 0 means the engine is not keeping up, or is down.
- **Live viewers** (`striker.live.connections`) - live-match websockets currently open on each
  `svc` instance (an up/down counter: +1 after a successful upgrade, -1 when it closes). `sum` it
  across instances.
- **Database pool** (`striker.db.connections` labeled by `state` = `in_use`/`idle`,
  `striker.db.connection_waits` and `striker.db.connection_wait_time`, in seconds) - each process's
  own `database/sql` pool (capped at 50 connections), registered by both `svc` and `engine`. Waits
  growing means requests are queueing for a connection.
- **Web push** (`striker.push.sent`, labeled by `outcome`: `sent`, `expired` for a 404/410 whose
  subscription is deleted, `failed` for a transport error or any other status) - nothing is counted
  while VAPID keys are not configured. Failed deliveries are only counted, not retried.

A ready-made Grafana dashboard covering every metric above lives at
`backend/resources/grafana/striker-dashboard.json`. Import it into Grafana (Dashboards > New >
Import > Upload JSON file) and pick the Prometheus-compatible datasource that receives striker's
OTLP metrics (Grafana Mimir/Cloud's native OTLP ingestion, or a Prometheus scraping an OTel
Collector in front of it) when prompted. Its queries assume the OTel-to-Prometheus naming
convention (dots become underscores, the unit is appended - `s` -> `_seconds`, `ms` ->
`_milliseconds`, `By` -> `_bytes` - monotonic counters get `_total`, an up/down counter is a plain
gauge, and histograms get `_bucket`/`_sum`/`_count`) - adjust the panels' PromQL if your pipeline
names metrics differently. `common/telemetry_dashboard_test.go` pins those names: it fails when an
instrument in `telemetry.go` appears in no panel, when panel ids repeat or overlap, or when the rows
change order.

The dashboard's rows, in order: **Service health** (request rate, 5xx and 4xx ratios, the ten routes
with the most 5xx, p50/p95 latency without the websockets), **Engine health** (backlog, seconds since
each job last finished with a 300 s line, job p95, claims won and runs per replica), **Simulated
world** (matches, goals, goals per match, rollovers, academy), **Economy**, **Economy flows**
(credited and debited per day by reason, net money created, total-budget trend), **Transfer & loan
market** (including CPU market actions), **Community**, **Live & delivery** (live viewers, push
outcomes, DB pool) and **Per node**, which the `job` / `instance` template variables filter. Every
season rollover is drawn as an annotation on every time series.

To try it locally without Grafana's OTel plumbing, start a Prometheus (>= 2.47) with
`--web.enable-otlp-receiver` and point the processes at it:

```bash
OTEL_METRICS_ENABLED=true OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:9090/api/v1/otlp \
  OTEL_METRIC_EXPORT_INTERVAL=5000 go run ./cmd --svc --engine
```

## Local development (without Docker)

```bash
cp app.env.sample app.env   # fill in real values
go run ./cmd --svc          # or --engine / --initialize / --svc --engine
```

`--svc` and `--engine` can be combined to run the API server and the match-simulation/season-
rollover scheduler in one process - handy for local development so you don't need two terminals.
Docker Compose still runs them as separate containers (see `docker-compose.yml`), which works
exactly the same way.

## CI

`.github/workflows/ci.yml` runs on every push to `main` and every pull request: `go build`/`go
vet`/`go test` for the backend, and `tsc --noEmit`/`oxlint`/`vitest run`/`npm run build` for the
frontend - the same checks expected before opening or merging a PR, just automated. Neither job
needs Postgres/Redis running, since backend tests only ever cover pure formulas, never DB
round-trips (no DB is available in CI to round-trip against).
