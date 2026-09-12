# Architecture

Orientation doc for a session picking up this codebase cold. `backend/README.md` is the
authoritative, exhaustive reference for backend behavior (every admin endpoint, every schema
migration file, transfer windows, match scheduling, world management, contracts...) - read it
before touching any of that. This file is the map: what lives where, and how the pieces talk to
each other.

## Repo layout

```
backend/    Go/Gin API + background simulation engine + DB seeder (one Go module)
frontend/   React/TypeScript/Vite client (current, actively developed)
```

There is no other frontend in this repo - a legacy `fmfrontend/` (an older Create-React-App
client) existed early in this project's history and has been removed; don't recreate it or
assume it still exists.

## Backend (`backend/`)

One Go binary (`cmd/striker.go`), three run modes selected by CLI flag, combinable except with
`--initialize`:

- `--svc` - the HTTP API (Gin), see `service/fmservice.go`.
- `--engine` - the background scheduler (match simulation, season rollover, cup progression,
  weekly wages, friendly-match generation), see `engine/fmengine.go`.
- `--initialize` - one-shot, **destructive** DB seed/reset (`handlers.InitializeDatabase`).

`--svc --engine` together run both in one process (handy for local dev); Docker Compose always
runs them as separate containers/processes regardless.

### Package structure

- `common/` - the shared core: DB access (raw `database/sql` + `pgx` driver, **no ORM** - every
  query is a `const ...StmtSql = "SELECT ..."` string plus a Go function that runs it and scans
  results), domain types (`fmstructs.go`), Redis cache + pub/sub (`cache.go`), JWT/session
  helpers, the multi-country/league-tier model, admin/world-management logic, contracts,
  trainers, notifications, friendlies, etc. This is the biggest package by far (60+ files) and is
  where most business logic actually lives, not `handlers/`.
- `handlers/` - Gin `gin.HandlerFunc` HTTP handlers. Thin: parse request, call into `common/` or
  `engine/`, write JSON response. Also owns the DB initializer/seeder (`initializer.go`) and the
  auth glue (`handlers.go`: `DoExternalLogin`, `DoDevLogin`, `GetUserIdFromCookie`).
- `engine/` - the simulation engine: minute-by-minute match simulation
  (`scorer_simulation.go`/`initial_simulation.go`/goals/cards/injuries/substitutions), fixture
  generation, season rollover, promotion/relegation, cup progression, player
  aging/progression/academy, live-match WebSocket broadcasting, and the cron-style schedulers
  that drive all of it from `StartEngine`.
- `service/` - Gin router wiring (`fmservice.go`): route groups, middleware (JWT auth, CORS,
  admin gate), one file.
- `resources/sql/` - `init.sql` is the full consolidated baseline schema, applied on a fresh
  database; `resources/sql/history/` holds every already-shipped `initN_*.sql` migration that
  was folded into it, kept only for the manual-upgrade path on an already-running database (see
  backend/README.md) - `common.EnsureSchema`'s directory listing is non-recursive, so files under
  `history/` are never auto-applied. See "Schema evolution" below.

### Data access pattern

No migration framework, no ORM. Every SQL string is a Go constant in `common/database.go` (or a
feature-specific file), named `<Verb><Noun>StmtSql`. Reading `common/database.go` top-to-bottom
is close to reading the schema itself. This is deliberate and consistent - follow the same
pattern (a named `const ...Sql` string + a small Go wrapper function) rather than introducing a
query builder or ORM for new work.

### Schema evolution

`resources/sql/init.sql` is the full baseline schema for a brand-new database - every table and
column, including everything that used to ship as a separate `initN_*.sql` migration. Those
superseded migration files still exist, unchanged, under `resources/sql/history/`, purely as a
record and for the manual-upgrade path described below; they are never auto-applied (see the
directory-skipping behavior noted above).

`init.sql` is applied by two independent mechanisms:

1. Postgres's own `docker-entrypoint-initdb.d` auto-init, when the container starts against a
   **fresh, empty** data volume (see `backend/docker-compose.yml`'s postgres service) - it globs
   `resources/sql/*.sql` non-recursively, so `history/` is skipped the same way.
2. `common.EnsureSchema`, called by `--initialize`, which applies every file in `resources/sql/`
   (again non-recursive) itself the first time it's pointed at a database with no schema yet, in
   `schemaFileSortKey` order: `init.sql` first, then any numbered `initN_*.sql` files in true
   numeric order, then anything else lexicographically (the `names_*_.../lastnames_*_...` data
   dumps).

Either path only ever runs **once** - these files have no `IF NOT EXISTS`/idempotency guards, so
re-running them against an already-seeded database duplicates data or errors outright.

**Going forward**, keep making schema changes as additive migrations on top of this baseline:
add one new `resources/sql/initN_whatever.sql` file (picking up numbering after the highest
existing `initN_*.sql` - check **both** `resources/sql/` itself, which accumulates new,
not-yet-folded migrations as the app evolves, and `resources/sql/history/` for older ones already
folded into the baseline; as of this writing the highest is `init33_free_agent_retirement.sql`,
sitting unfolded at the top level, not in `history/`), never edit `init.sql` itself. On an
**already-running** database (the common case once the app is live),
that new file needs applying by hand once - see backend/README.md's "Schema changes on an
already-running database" section for the exact `psql` command. Once enough of these accumulate,
they can be folded into `init.sql` and moved to `history/` again, exactly as this consolidation
did - but that's a deliberate, occasional cleanup, not something to do per-change.

### Auth

Google OAuth (verified server-side, see `DoExternalLogin`) or, when `DEV_LOGIN_ENABLED=true`, a
no-credential-check dev-login shortcut (`DoDevLogin`, a fixed `common.DevUsers` roster) - both
converge on the same `POST /v1/login` endpoint (`{"jwt": "..."}` vs `{"devEmail": "..."}`) via
`gin-jwt`'s `Authenticator` callback in `service/fmservice.go`. A successful login sets an
HTTP-only JWT cookie; `GetUserIdFromCookie` (in `handlers/handlers.go`) is how every other
authenticated handler recovers the caller's identity - there's no session store, the cookie *is*
the session.

**A brand-new sign-in doesn't get a team immediately.** It's registered with no country and no
team (`LoggedUser.pendingCountrySelection = true`), and the frontend's `Layout` component
redirects every route to `/onboarding` until they pick a country
(`POST /v1/auth/onboarding/country`), which is what actually triggers `common.AssignTeam`. See
"Country/league/team model" below and `frontend/src/features/onboarding/`.

Admin access is config-only (`ADMIN_EMAILS`, comma-separated), re-checked on every login - no
admin-managing-admins UI beyond one user promoting another via `PUT /v1/admin/users/:id/admin`
once they're already an admin themselves.

### Country/league/team model

- `countries` - one row per **active** country (has a generated world: leagues, teams, players).
  `common.KnownCountries` is the superset of countries the engine *knows how to* generate a
  world for (needs a `names_<code>`/`lastnames_<code>` table pair + team-naming data); not all of
  them are active. An admin activates an inactive one via `POST /v1/admin/countries`.
- `leagues` - one row per division-within-a-country-within-a-group, with a `tier` column
  (1 = top division, increasing = lower). Promotion/relegation moves teams between tiers at
  season rollover.
- `teams.owner_id` - `NULL` means unclaimed (AI-controlled or literally free). `AssignTeam`
  (`common/database.go`) hands an unclaimed team to a new user, picking by
  `GetFreeTeamStmtSql`: ordered by tier ascending then random - fills the top division first,
  only spilling into lower tiers once it's full of human owners.
- `teams.ai_level` - unclaimed teams still "play" a season; higher AI levels take more actions
  (lineup freshness, contract renewals, transfer shopping - see backend/README.md's "AI teams").
- Promotion/relegation slot counts (how many teams move between adjacent tiers each season) are
  admin-configurable per-world (`season.promotion_slots_per_group`/`relegation_slots_per_group`,
  `common.PromotionRelegationConfig`), not hardcoded - see backend/README.md's "Promotion and
  relegation".

### Player lifecycle

All of this runs from `engine.ProgressPlayers`/`engine.ProgressFreeAgents`, called once per
season from `engine.AdvanceSeasonIfComplete` - see backend/README.md's "Player aging and decline",
"Free agency has a time limit", and "Youth academy (cantera)" sections for the full detail; the
short version, since it's easy to get wrong when touching any one piece in isolation:

- **Skills drift with age**, not indefinitely: young players trend up, prime-age players hold
  flat, and past age 30 they trend down (`engine.ageTrendFor`). A hired trainer's growth bonus
  (`engine.trainerBonus`) only applies while that trend is still flat-or-rising - it cannot
  reverse a veteran's decline, only fail to slow it, so skill growth is never truly infinite.
- **Retirement** (`engine.ShouldRetire`) is age-based (rising odds from 33, forced at 40) and
  applies to rostered players and free agents alike.
- **A free agent's clock is separate from age**: `engine.ProgressFreeAgents` ages unowned players
  (who `ProgressPlayers` skips entirely, since it only iterates players belonging to a team) and
  forces retirement after `engine.FreeAgencyRetirementSeasons` (2) consecutive seasons unsigned,
  tracked by `players.free_agent_seasons` (reset to 0 on signing).
- **Retirement isn't always the end**: whether triggered by age or by the free-agency clock,
  `common.RetireFreeAgentAsTrainer` converts an unowned player into a fresh trainer candidate on
  the open market (skill = average of their playing attributes) instead of deleting them outright.
  A rostered player who retires by age alone (`common.RetirePlayer`) is not converted this way.
- **The youth academy ("cantera")** is a separate concept from all of the above: a small
  per-team pool of unsigned prospects (`players.in_academy`) a manager can promote from manually,
  at most one per real calendar week - unrelated to the automatic squad-backfill mechanism
  (`common.CreateAcademyPlayers`) that tops up a team's senior squad size directly.

### Redis usage

Two independent uses, both optional-but-expected in production, both configured via
`REDIS_HOST`/`REDIS_PORT`/`REDIS_PASSWORD`/`REDIS_DB`:

1. `common.RedisCache` (`cache.go`) - a short-TTL key/value cache for expensive read paths.
2. `common.LiveMatchHub` - pub/sub backing live match streaming. The API server (`svc`) and the
   simulation engine (`engine`) are **separate processes** (separate containers in Docker
   Compose), so a match simulated by `engine` can only reach a browser connected to `svc`'s
   WebSocket via a hub both processes actually connect to - hence Redis pub/sub rather than an
   in-process channel, even when both run combined via `--svc --engine` in one process for local
   dev (still separate `DatabaseContext`/hub instances internally).

### Observability

`common/telemetry.go` defines package-level OpenTelemetry metric instruments (`common.Meter`,
`common.HTTPRequestsCounter`, `common.MatchesSimulatedCounter`, ...) used from `service/`,
`engine/`, and `common/` alike - safe to call unconditionally anywhere, since otel's default
global MeterProvider is a no-op until `common.InitMetrics` installs a real one. Export is
**off by default**; set `OTEL_METRICS_ENABLED=true` (plus the standard `OTEL_EXPORTER_OTLP_*`
env vars) to send metrics to a collector - see backend/README.md's "OpenTelemetry metrics"
section for the full list of what's instrumented and how to configure it.

### Live match streaming

A match's simulation is instant and fully computed/persisted to Postgres in one go - there's no
real-time simulation loop. "Live" is a cosmetic replay: `engine.BroadcastMatchLive` paces
already-known events out over ~22 seconds via the Redis-backed hub above, to any browser with the
match's live WebSocket open (`GET /v1/teams/match/:id/live`). The frontend also polls every 5s
while a match isn't marked played yet, as a backstop against a missed/dropped socket message.

## Frontend (`frontend/`)

Vite + React 19 + TypeScript + MUI (Material 3, single seed color - see `theme.ts`) + TanStack
Query for all server state + React Router. No Redux/global client state store - server state
lives entirely in the Query cache, keyed by `queryKeys` in `api/hooks.ts`.

```
src/
  api/          client.ts (thin fetch wrapper), hooks.ts (every useQuery/useMutation), types.ts
                (hand-written types mirroring backend JSON - see its own header comment: field
                names follow the backend's json tags, not its Go field names)
  app/          App.tsx (routes), Layout.tsx (chrome + the onboarding redirect gate), Header,
                NavDrawer, Subheader, theme mode provider
  components/   shared presentational components (avatars, pitch/formation rendering, dialogs)
  features/     one folder per page/domain (dashboard, team, match, player, transfers, admin,
                onboarding, ...) - this is where almost all UI code lives
  i18n/         en/es locale JSON, i18next setup
  utils/        pure helpers (formatting, rating calculations)
```

`api/types.ts` is hand-maintained, not generated - when a backend JSON shape changes, update this
file by hand to match. `api/hooks.ts` centralizes every network call as a `useQuery`/
`useMutation`; feature components should call these hooks, not `api/client.ts` directly.

### Routing & the onboarding gate

`App.tsx` defines every route inside one `<Route element={<Layout />}>` wrapper - `Layout`
renders the Header/NavDrawer/Subheader chrome around whatever page is routed to. There is no
generic "protected route" wrapper; individual pages rely on their own data hooks 401ing
gracefully. The one exception is the onboarding gate: `Layout` checks `useMe()` and, if
`loggedUser.pendingCountrySelection` is true and the current route isn't already `/onboarding`,
renders a redirect instead of the normal chrome - see "Auth" above.

### i18n

Two locales (`en`, `es`) as flat-ish nested JSON under `i18n/locales/`. Every new user-facing
string needs a key added to **both** files (this is a real gap when it's missed, not just a
nice-to-have - there's no fallback-to-English-with-a-warning UX, missing keys just render the
key itself).

## Running it

See `HOWTO_RUN.md` for concrete Docker commands, including the case where Postgres/Redis already
run elsewhere and the app services need to be pointed at them instead of the bundled containers.

## Conventions worth preserving

- **No ORM, no query builder, backend.** Named SQL constant + small Go function, matching the
  existing `common/database.go` style.
- **Schema changes are additive migration files**, applied once, never edited after the fact
  once they've shipped (see "Schema evolution").
- **Frontend types are hand-written to match backend JSON tags exactly** (not Go field names) -
  keep `api/types.ts` in sync by hand when a backend struct's `json:"..."` tags change. A
  deliberately-preserved spelling quirk (`common.TeamQualification.RightAttach`'s JSON tag stays
  `rightAttach`, not the "corrected" `rightAttack`) exists precisely because already-persisted
  JSON in the DB used the old spelling - don't "fix" field-name typos in a JSON tag without
  checking whether historical data already used the old spelling.
- **i18n keys go in both `en.json` and `es.json` together**, same turn.
- **Every new user-facing capability that touches money, teams, or squads should also get a
  ledger/notification entry** where one of those already exists for similar actions (see how
  transfers, contract renewals, and budget corrections all log to the team transaction ledger) -
  consistency here is what makes the finances page and notification feed trustworthy.
