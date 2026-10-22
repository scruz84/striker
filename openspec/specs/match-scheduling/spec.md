# match-scheduling Specification

## Purpose

When league matches, cup ties and friendlies get simulated, how a result is revealed to viewers
as if it were happening live, when a lineup can still be changed, and how kickoff dates are set
and re-set.

## Requirements

### Requirement: Periodic simulation jobs
The engine SHALL run each competition's simulation from its own cron job, each polling its own table for fixtures that are due: league matches, cup ties and friendlies every minute.

#### Scenario: Per-minute jobs
- **WHEN** the engine process is running
- **THEN** `SimulateMatches`, `SimulateCupTies` and `SimulateFriendlies` each run once a minute

#### Scenario: Slower housekeeping jobs
- **WHEN** the engine process is running
- **THEN** the season-rollover check, the CPU daily activity, national-team friendly pairing, national-team call-ups and stale live-broadcast recovery each run every 5 minutes
- **AND** stuck-simulation recovery runs every 15 minutes
- **AND** player birthdays run daily at 01:00 UTC, free-agent progression daily at 02:00 UTC, the international cup job daily at 03:00 UTC, pending-offer responses at 00:00 and 12:00 UTC, and free-agent injection weekly on Sunday at 00:00 UTC

### Requirement: Early simulation window
A league match or friendly MUST become eligible for simulation once its kickoff is less than 15 minutes away (or already past) and it is neither played nor in simulation. A cup tie MUST only become eligible once its kickoff has arrived.

#### Scenario: League match ten minutes before kickoff
- **WHEN** an unplayed league match kicks off in 10 minutes
- **THEN** the next per-minute tick simulates the whole 90 minutes at once

#### Scenario: Overdue match
- **WHEN** an unplayed match's kickoff is already in the past (e.g. the engine was down)
- **THEN** it is still picked up by the next tick

#### Scenario: Cup tie
- **WHEN** a cup tie's kickoff is 10 minutes away
- **THEN** it is not simulated until its kickoff time is reached

### Requirement: Live replay of a simulated result
The engine SHALL replay each simulated match over a WebSocket channel at one simulated minute per real minute, never publishing minute 1 before the advertised kickoff, so a viewer sees it unfold rather than a finished score.

#### Scenario: Viewer opens the match mid-way
- **WHEN** a viewer connects 30 minutes after kickoff
- **THEN** they see the state as of minute 30 and the remaining minutes as they are paced out
- **AND** the starting lineups are published once before the first minute

### Requirement: Spoiler-free side effects
Everything a team owner could see outside the match page MUST wait until the live replay completes: the result notification, new injuries and suspensions, and post-match skill growth.

#### Scenario: Match simulated before kickoff
- **WHEN** a league match is simulated 15 minutes before kickoff
- **THEN** no result notification, injury/suspension or skill change becomes visible until its replay finishes

### Requirement: Bounded concurrent simulation
The engine SHALL simulate at most 8 matches concurrently per tick, each isolated so one panic cannot abort the rest of the batch.

#### Scenario: One match panics
- **WHEN** one of a tick's matches panics during simulation
- **THEN** the other matches of that tick still complete

### Requirement: Recovery from interrupted simulations
The engine SHALL self-heal fixtures whose simulation or replay was interrupted (e.g. a crash or redeploy) without an admin.

#### Scenario: Simulation crashed mid-run
- **WHEN** a league match or friendly is left `in_simulation = true` and unplayed
- **THEN** the 15-minute recovery job re-simulates it

#### Scenario: Replay goroutine died
- **WHEN** a played match's live channel has gone silent past the stale timeout without completing
- **THEN** the 5-minute recovery job closes it once, for every viewer, with the known final score

### Requirement: Lineup lock before kickoff
Changing a specific match's lineup (league match, cup tie or friendly) MUST be rejected with `ErrLineupLocked` from 20 minutes before kickoff onwards. The team-wide default lineup is never locked.

#### Scenario: Edit 25 minutes before kickoff
- **WHEN** an owner saves a match lineup 25 minutes before kickoff
- **THEN** the change is accepted

#### Scenario: Edit 18 minutes before kickoff
- **WHEN** an owner saves a match lineup 18 minutes before kickoff
- **THEN** the request fails with `ErrLineupLocked`
- **AND** the frontend already shows a locked notice instead of the editor, so no save is attempted

#### Scenario: Default lineup
- **WHEN** an owner changes the team-wide default lineup at any time
- **THEN** the change is accepted

### Requirement: League kickoff schedule
Each league's kickoff weekdays and time SHALL come from its own admin-set schedule, or the app-wide default when it has none; fixtures are generated a season at a time on that cadence. Cup ties and friendlies kick off a short fixed delay after being scheduled.

#### Scenario: Accepted friendly challenge
- **WHEN** a friendly challenge is accepted
- **THEN** the friendly is dated 5 minutes later

### Requirement: Rescheduling on schedule change
Changing a league's schedule override, clearing it, or changing the app-wide default MUST immediately re-date every unplayed match of the affected leagues onto the new schedule, keeping round order. A league with its own override is unaffected by a default change.

#### Scenario: Admin changes the default schedule
- **WHEN** the app-wide default schedule changes
- **THEN** every unplayed match in leagues without an override is re-dated onto it, round by round
- **AND** leagues with their own override keep their dates

## Code map

- Jobs: `backend/engine/scheduler.go` (`Start*Scheduler`), wired in `engine/fmengine.go`.
- Eligibility queries: `common.GetSimulableMatchesStmtSql` (`common/database.go`), friendlies in
  `common/friendlies.go`, cup ties in `common/cups.go`.
- Concurrency: `maxConcurrentSimulations`, `engine/engine.go`.
- Live replay: `engine/livematch.go` (`BroadcastMatchLive`, `onComplete`); recovery in
  `engine/livematch_recovery.go` and `FinishStuckSimulations` (`engine/engine.go`).
- Lineup lock: `common/lineuplock.go` (`LineupLockMinutes`, `IsLineupLocked`); handlers
  `SetMatchLineupHandler`/`SetCupTieLineupHandler`/`SetFriendlyLineupHandler`; frontend
  `useIsLineupLocked` (`frontend/src/api/useLiveMatch.ts`).
- Schedules: `common/leagueschedule.go`, `common/matchschedule.go`, `engine/schedule.go`
  (`ComputeFixtureDates`), `engine/reschedule.go`; friendly delay `friendlyScheduleDelay`
  (`common/friendlies.go`).
