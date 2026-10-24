# engine-replicas Specification

## Purpose

The engine (`--engine`) must scale horizontally: production runs at least two replicas, and every
replica fires the same cron triggers at the same instant. This spec states how every engine step
that generates data (fixtures, season rollovers, cup rounds, national-team fixtures, simulations,
call-ups, periodic batches) stays correct when several replicas run it at once, so two nodes never
generate the same data twice or overwrite each other's work.

## Requirements

### Requirement: Generation is safe under concurrent replicas
Every engine step that reads state and then writes data derived from it MUST give the same result when N replicas run it at the same instant as when one does. It SHALL use one of: an advisory lock with the state re-read inside the lock, a single-statement atomic claim, or a per-slot job claim. Skipping on a held lock is allowed only when the next tick redoes the work.

#### Scenario: Two replicas tick together
- **WHEN** two replicas fire the same job in the same second
- **THEN** each piece of data (fixture, cup round, tie result, notification) is written once

#### Scenario: New engine job
- **WHEN** a new engine job is added
- **THEN** it ships with an integration test that runs it from several goroutines at once against Postgres

### Requirement: Season rollover once per country
A country's rollover (aging, loans, promotion/relegation, prizes, cup start, classification reset, season bump, fixtures) SHALL run under a session advisory lock keyed `hashtext(country)`, re-reading the current season and its match counts from the database inside the lock. New fixtures MUST be built from the leagues as they stand after promotion/relegation.

#### Scenario: Three replicas see the season complete
- **WHEN** three replicas run the rollover for the same finished country at once
- **THEN** the season advances by exactly 1, one cup is created and each league gets one schedule

#### Scenario: Relegated team
- **WHEN** a team is relegated from division D1 at rollover
- **THEN** no new-season D1 fixture includes it

### Requirement: League fixtures generated once
A league's fixtures SHALL be inserted in one transaction holding `pg_advisory_xact_lock(leagueId)`, which waits rather than skips, and re-checks inside the lock whether the league already has upcoming matches for the season.

#### Scenario: Lock key collision
- **WHEN** an unrelated cup or lineup lock with the same numeric id is held
- **THEN** fixture generation waits for it and still generates the league's fixtures

### Requirement: Cup rounds and ties
Advancing a cup (next round or completion) SHALL hold a session lock keyed `cupId`. Each due cup tie SHALL be simulated under a session lock on its own namespaced key, re-reading `played` inside the lock, so a tie is simulated once; a crash releases the lock and the next tick retries.

#### Scenario: Two replicas pick up the same tie
- **WHEN** two replicas' cup jobs find the same due tie
- **THEN** it is simulated once and its round prize is paid once

### Requirement: International cups and World Cup start
The international cup round SHALL be claimed by one statement that checks the 60-day cadence and increments the counter together. Starting a World Cup MUST hold the national-team scheduling lock (waiting for it), so no friendlies pass can pair a country while its group fixtures are created.

#### Scenario: Daily job on two replicas
- **WHEN** the 03:00 UTC job fires on two replicas with a round due
- **THEN** exactly one claims it and one set of international cups starts

### Requirement: World Cup knockout start once
Moving a World Cup from groups to knockout SHALL run under a session lock keyed to that World Cup, re-reading its stage inside the lock.

#### Scenario: Last group match played
- **WHEN** three replicas' cup jobs see the group stage complete
- **THEN** one knockout cup is created

### Requirement: National-team fixtures and call-ups
Friendly pairing SHALL run under one global session lock (also taken by the World Cup start). The call-up pass SHALL run under its own global lock, with its "side has no lineup yet" query inside it, so each side is called up and notified once.

#### Scenario: Two replicas pair friendlies
- **WHEN** two replicas run the pairing pass at once
- **THEN** no national team has two friendlies on the same date

#### Scenario: Two replicas run call-ups
- **WHEN** two replicas run the call-up pass at once
- **THEN** each called-up player's club is notified once

### Requirement: Simulation claims and stuck recovery
League matches and friendlies SHALL be claimed (`in_simulation = true`, `in_simulation_since = now()`) in the same transaction that selects them `FOR UPDATE`. Stuck recovery MUST only take rows claimed at least 10 minutes ago (or never stamped) and re-claim them in one statement.

#### Scenario: Recovery fires as a pass claims
- **WHEN** the 15-minute recovery job runs the same second a live pass claims a match
- **THEN** that match is not simulated a second time

#### Scenario: Two replicas recover
- **WHEN** two replicas run stuck recovery at once
- **THEN** each abandoned match is re-simulated by one of them

### Requirement: Periodic jobs once per slot
A job meant to run once per period SHALL first claim its slot (fire time truncated to the period, UTC) in `engine_job_runs`; only the winner runs. A database error counts as not claimed.

#### Scenario: Weekly free agent injection
- **WHEN** Sunday 00:00 UTC fires on two replicas
- **THEN** one batch of free agents is injected, not two

| Job | Slot |
| --- | --- |
| `free_agent_progression` | 24 h |
| `free_agent_injection` | 24 h (fires weekly) |
| `birthdays` | 24 h |
| `offer_response` | 12 h |
| `ai_activity` | 5 min |

### Requirement: Advisory lock keys
Existing locks keep their bare keys (`leagueId`, `cupId`, `hashtext(country)`, constants 8817224/8817225) so a replica on the previous build still contends during a rolling deploy. New locks MUST use `NamespacedLockKey(namespace, id)` (namespace in the top 16 bits) so they never collide with a bare id.

#### Scenario: New lock
- **WHEN** a new engine lock is added
- **THEN** it gets a new `LockNamespace*` constant

## Incident history

- Before the per-country rollover lock and the fixture-generation lock (PRs #241 and #243,
  2026-10-16), two replicas could roll the same country over at once. Damage seen in production:
  teams that played several cup ties against different opponents in the same round, and, in a
  season generated then, a league fixture against a team that had already been relegated from
  that division. Such fixtures are most likely data left over from a rollover that ran before the
  fix; the concurrent-rollover integration test now checks that every new-season fixture matches
  the post-relegation leagues.
- National-team friendlies double-booked a country until the pairing pass got its lock (PR #253,
  2026-10-18).
- An audit on 2026-10-24 found races the earlier fixes did not cover, all fixed together: cup
  ties simulated once per replica (no claim at all); a World Cup knockout cup created once per
  replica; the international cup round claimed by a separate check and increment; a World Cup
  start racing the friendlies pass; call-up notifications sent once per replica; weekly and daily
  batches (free agent injection and progression, birthdays, offer sweep, AI activity) running once
  per replica; stuck-match recovery re-simulating rows a live pass had just claimed; and fixture
  generation silently skipping a league when its lock key collided with an unrelated lock.

## Code map

- Helpers: `backend/common/enginelocks.go` (`NamespacedLockKey`, `LockNamespace*`,
  `TryWithSessionAdvisoryLock`, `WithSessionAdvisoryLock`, `ClaimEngineJobSlot`); schema
  `resources/sql/init84_engine_horizontal_scaling.sql` (`engine_job_runs`,
  `in_simulation_since`).
- Rollover: `advanceCountrySeasonIfComplete` (`engine/season.go`). Fixtures:
  `generateMatchesForOneLeague` (`engine/engine.go`).
- Cups: `advanceCup`, `simulateCupTieOnce`, `StartInternationalCupsIfDue` (`engine/cups.go`),
  `common.ClaimInternationalCupSeason` (`common/database.go`), `common.IsCupTiePlayed`.
- World Cup: `StartWorldCupForNewSeason`, `advanceWorldCup` (`engine/worldcup.go`).
- National teams: `ScheduleNationalTeamFriendlies`, `WithNationalTeamSchedulingLock`,
  `CallUpNationalTeamSquads` (`common/nationalteam.go`).
- Simulation claims: `GetSimulableMatches`, `GetStuckMatches`, `StuckSimulationAge`
  (`common/database.go`); `GetSimulableFriendlies`, `GetStuckFriendlies` (`common/friendlies.go`).
- Periodic jobs: `claimRun` and the `Execute` methods in `engine/tasks.go`.
  `striker.engine.job.claims{engine_job,outcome}` (`common/telemetry.go`) shows how the claims are
  shared: each replica records `won` or `lost` for every claim.
- Tests: `engine/engine_replicas_integration_test.go`, `common/enginelocks_integration_test.go`,
  plus `engine/season_advance_lock_integration_test.go`, `engine/cup_advance_lock_integration_test.go`,
  `engine/fixture_generation_lock_integration_test.go`,
  `common/national_team_friendly_lock_integration_test.go` (all need `STRIKER_TEST_DB_DSN`).
