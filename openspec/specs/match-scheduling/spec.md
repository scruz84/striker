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
- **THEN** the season-rollover check, the regional cup check, the CPU daily activity, national-team friendly pairing, national-team call-ups and stale live-broadcast recovery each run every 5 minutes
- **AND** stuck-simulation recovery runs every 15 minutes
- **AND** player birthdays run daily at 01:00 UTC, free-agent progression daily at 02:00 UTC, the World Cup job daily at 03:00 UTC, pending-offer responses at 00:00 and 12:00 UTC, and free-agent injection weekly on Sunday at 00:00 UTC

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
The engine SHALL replay each simulated match over a WebSocket channel at one simulated minute per real minute, added minutes included, never publishing minute 1 before the advertised kickoff, so a viewer sees it unfold rather than a finished score. The half-time break SHALL last 15 minutes minus the first-half added time, so minute 46 is always published 59 minutes after kickoff.

#### Scenario: Viewer opens the match mid-way
- **WHEN** a viewer connects 30 minutes after kickoff
- **THEN** they see the state as of minute 30 and the remaining minutes as they are paced out
- **AND** the starting lineups are published once before the first minute

#### Scenario: Publish schedule with added time
- **WHEN** a match that kicks off at 14:00 has 3 first-half and 5 second-half added minutes
- **THEN** minute 45 is published at 14:44 and the added-time board for 3 minutes comes with it, 45+1 to 45+3 at 14:45 to 14:47, and half time at 14:47
- **AND** minute 46 is published at 14:59, minute 90 at 15:43 with the board for 5 minutes, 90+1 to 90+5 at 15:44 to 15:48, and the final whistle right after 90+5

#### Scenario: Shortest break
- **WHEN** the first half has the maximum 5 added minutes
- **THEN** the half-time break lasts 10 minutes and minute 46 is still published 59 minutes after kickoff

#### Scenario: Late simulation catches up
- **WHEN** a match is simulated 50 minutes after its kickoff
- **THEN** every minute up to the one matching 50 real minutes after kickoff, added minutes included, is published at once, and the rest follow the schedule above

### Requirement: Spoiler-free side effects
Everything a team owner could see outside the match page MUST wait until the live replay publishes its final whistle, after the last second-half added minute: the result notification, new injuries and suspensions, and post-match skill growth.

#### Scenario: Match simulated before kickoff
- **WHEN** a league match is simulated 15 minutes before kickoff
- **THEN** no result notification, injury/suspension or skill change becomes visible until its replay finishes

#### Scenario: Injury in added time
- **WHEN** a player is injured at 90+3 of a match with 5 added minutes
- **THEN** the injury becomes visible outside the match page only after the final whistle that follows 90+5

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

#### Scenario: Simulation returned an error or panicked
- **WHEN** simulating a league match, friendly or cup tie returns an error or panics
- **THEN** the attempt is counted, its error message is stored on the fixture, and the fixture is retried no sooner than 10 minutes later

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
A league's kickoff weekdays SHALL come from its own admin-set override, or else the app-wide default. Its kickoff time SHALL come from its override, or else the country's league kickoff time (local, default 14:00) plus 15 minutes for each tier below tier 1. Kickoffs are dated on the country's local clock. Fixtures are generated a season at a time on that cadence.

#### Scenario: Staggered tiers
- **WHEN** a 5-tier Spanish country with the default 14:00 kickoff generates its fixtures
- **THEN** tier 1 kicks off at 14:00, tier 2 at 14:15, tier 3 at 14:30, tier 4 at 14:45 and tier 5 at 15:00, Madrid time

#### Scenario: Daylight saving
- **WHEN** a season's matchdays span the end of summer time in Spain
- **THEN** tier 1 still kicks off at 14:00 on the Madrid clock before and after the change

#### Scenario: Override wins
- **WHEN** the admin gives league III.2 its own schedule at 18:00
- **THEN** III.2 kicks off at 18:00 and the other tier-3 groups keep 14:30

#### Scenario: League created by a restructure
- **WHEN** a restructure creates league IV.5 in a country whose kickoff is 14:00
- **THEN** IV.5 has no override and kicks off at 14:45 local time

### Requirement: Cup kickoff times
A domestic cup tie SHALL kick off at 17:00 in its country's local time, a regional (international) cup tie at 20:45 in the home side's country, each at the first such local time at least 12 hours after its round is created (one round a day). A World Cup knockout tie kicks off 24 hours after its round is created.

#### Scenario: Domestic cup round
- **WHEN** a Spanish cup round is created on Friday 13 Nov 2026 at 10:00 UTC
- **THEN** its ties kick off on Saturday 14 Nov at 17:00 Madrid time (16:00 UTC)

#### Scenario: Regional cup across timezones
- **WHEN** a Copa America round is created at 10:00 UTC with an Argentine and a Mexican home side
- **THEN** each tie kicks off at 20:45 local time in Buenos Aires and in Mexico City respectively

#### Scenario: Daylight saving
- **WHEN** a domestic cup round is dated across a daylight-saving change
- **THEN** it still kicks off at 17:00 on the local clock

### Requirement: Friendly kickoff slots
A friendly MUST kick off on the hour, at quarter past, half past or quarter to (:00, :15, :30, :45, zero seconds) and at least 5 minutes after it is accepted. A kickoff off that grid is rejected with `ErrInvalidFriendlyKickoff`; an auto-accepted friendly takes the first slot at least 5 minutes ahead.

#### Scenario: Accepted friendly challenge
- **WHEN** a CPU team accepts a friendly challenge at 18:07
- **THEN** the friendly is dated 18:15

#### Scenario: Off-grid kickoff
- **WHEN** an owner accepts a challenge with a kickoff at 18:50
- **THEN** the request is rejected and the pickers already flag it, offering only quarter-hour steps

### Requirement: Weekly friendly cap and clashes
No club SHALL play more than `max_friendlies_per_week` friendlies in one calendar week (Monday to Monday, UTC), whichever side sent the challenge, and no club SHALL have two friendlies at the same kickoff. Accepting a challenge MUST re-check both rules for both teams, for the week of the chosen kickoff, even under concurrent acceptances. A friendly in a later week does not use up the current week's cap.

#### Scenario: Several challenges sent below the cap
- **WHEN** a club with a cap of 1 and no friendly sends challenges to two human clubs and both accept for the same week
- **THEN** the first acceptance creates the friendly and the second is rejected with the weekly limit

#### Scenario: Same kickoff twice
- **WHEN** a human club accepts two challenges for the same kickoff
- **THEN** the second is rejected and the club keeps one friendly at that time

#### Scenario: Friendly next week
- **WHEN** a club with a cap of 1 has a friendly scheduled next week
- **THEN** it can still be challenged and play one this week

### Requirement: Rescheduling on schedule change
Changing a league's schedule override, clearing it, changing the app-wide default, or changing a country's league kickoff time MUST immediately re-date every unplayed match of the affected leagues onto the new schedule, keeping round order. A league with its own override is unaffected by a default or country change. A match that is played or in simulation MUST keep its date, even when other matches of its round move.

#### Scenario: Admin changes the default schedule
- **WHEN** the app-wide default schedule changes
- **THEN** every unplayed match in leagues without an override is re-dated onto it, round by round
- **AND** leagues with their own override keep their dates

#### Scenario: Round half played
- **WHEN** a league is rescheduled while one match of a round is played, one is in simulation and one has not started
- **THEN** only the match that has not started is re-dated; the other two keep their dates

#### Scenario: Admin changes a country's kickoff
- **WHEN** the admin moves Italy's league kickoff from 14:00 to 15:00
- **THEN** Italy's unplayed matches without an override are re-dated to 15:00 for tier 1, 15:15 for tier 2 and so on, and no other country's matches move

### Requirement: Default result after repeated failures
A league match, friendly or cup tie that has already used 3 simulation attempts without being played MUST NOT be simulated again for that kickoff. The next recovery pass, run once kickoff has passed, SHALL settle it as a 0-0 default result instead (see match-simulation for what that result books). If its kickoff was moved more than 15 minutes ahead, that pass SHALL instead give it its attempts back.

#### Scenario: Third attempt fails
- **WHEN** a league match fails at 0, +10 and +20 minutes after its first claim
- **THEN** the next recovery pass after the third failure records it as played 0-0 with a `MATCH_DEFAULT_RESULT` event
- **AND** its country's season rollover is no longer blocked by it

#### Scenario: Retry succeeds
- **WHEN** a match fails once and its second attempt succeeds
- **THEN** it keeps its simulated result and no default is applied

#### Scenario: Database unreachable
- **WHEN** a fixture's last stored error is a connection error (the database or connection pool was unreachable)
- **THEN** that attempt is not counted towards the limit of 3, so an outage cannot default a whole round to 0-0

#### Scenario: Already-stuck rows after deploy
- **WHEN** this change is deployed and a league match or friendly has been claimed for more than 1 hour without being played
- **THEN** it gets exactly one more simulation attempt, and is settled 0-0 if that attempt fails

#### Scenario: Admin finishes a failing fixture
- **WHEN** an admin runs the "finish stuck matches" action while a league match, friendly or cup tie that already failed (and kicked off) is waiting out its retry backoff with attempts left
- **THEN** it gets one last attempt at once, without waiting for the backoff
- **AND** if that attempt fails too (an error or a panic), it is settled 0-0 in the same run; if it succeeds it keeps its simulated result
- **AND** an attempt that fails only because the database was unreachable is not counted, so the fixture is not defaulted

#### Scenario: Exhausted fixture re-dated to a later kickoff
- **WHEN** a league match that used its 3 attempts is rescheduled to kick off in 2 days
- **THEN** the next recovery pass, or the admin "finish stuck matches" action, releases its claim and sets its attempts to 0, so it no longer counts as stuck or failing
- **AND** it is simulated normally at its new kickoff, and settled 0-0 only if it then fails 3 more times

#### Scenario: Exhausted fixture kicking off soon
- **WHEN** a league match that used its 3 attempts kicks off in 5 minutes
- **THEN** it keeps its attempts and is settled 0-0 by the first recovery pass after kickoff

### Requirement: Durable post-match effects
The spoiler-gated post-match effects (availability countdown and new injuries and suspensions, skill growth, result notification) SHALL be stored as pending in the same transaction as the result. They MUST be applied exactly once: when the replay completes, or by a 5-minute job once kickoff is more than 2 hours (the live-broadcast window) in the past.

#### Scenario: Engine restarts during the replay
- **WHEN** the engine is redeployed at minute 50 of a match's live replay
- **THEN** the match's injuries, suspension countdown, skill growth and result notification are applied by the recovery job within 5 minutes of kickoff + 2 hours

#### Scenario: Replay completes normally
- **WHEN** a replay reaches its "complete" message
- **THEN** its pending effects are applied at that moment and the recovery job finds nothing left for that match

#### Scenario: Two replicas recover the same effects
- **WHEN** the replay's completion and two replicas' recovery jobs all try to apply the same match's pending effects
- **THEN** exactly one of them applies them

### Requirement: Country league kickoff setting
Each country SHALL have a league kickoff time on its local clock, from 00:00 to 22:45 on a quarter-hour. It defaults to 14:00, is set when the country is added, and the admin can change it later. It is the tier-1 time; lower tiers add 15 minutes per tier. A value that would push the deepest tier past 23:59 SHALL be rejected.

#### Scenario: Adding a country
- **WHEN** the admin adds Portugal with kickoff 13:00
- **THEN** Portugal's tier 1 kicks off at 13:00 Lisbon time and tier 5 at 14:00

#### Scenario: Too late for the deepest tier
- **WHEN** the admin sets a 5-tier country's kickoff to 23:30
- **THEN** the change is rejected, because tier 5 would kick off at 00:30 the next day

## Code map

- Jobs: `backend/engine/scheduler.go` (`Start*Scheduler`), wired in `engine/fmengine.go`.
- Eligibility queries: `common.GetSimulableMatchesStmtSql` (`common/database.go`), friendlies in
  `common/friendlies.go`, cup ties in `common/cups.go`.
- Concurrency: `maxConcurrentSimulations`, `engine/engine.go`.
- Live replay: `engine/livematch.go` (`BroadcastMatchLive`, `onComplete`); recovery in
  `engine/livematch_recovery.go` and `FinishStuckSimulations` (`engine/engine.go`), which only
  re-simulates rows claimed at least `common.StuckSimulationAge` ago.
- Admin "finish stuck matches" (`handlers.AdminFinishStuckMatchesHandler`): releases re-dated fixtures,
  re-simulates stuck ones, then `engine.RetryFailingFixturesNow` (`engine/defaultresults.go`,
  `common.GiveFailingFixturesLastAttempt`) gives failed fixtures past kickoff one last attempt, and
  `SettleDefaultResults` closes 0-0 whatever has used its attempts.
- Recovery from interrupted simulations: `engine.runClaimedSimulation` (`engine/simulationfailures.go`)
  records an error or panic through `common.RecordSimulationFailure` (`common/simulationfailures.go`:
  `MaxSimulationAttempts`, `SimulationRetryBackoff`, `IsConnectionError`); columns
  `simulation_attempts`, `last_simulation_error` and `last_simulation_attempt_at`
  (`resources/sql/init86_simulation_failures.sql`). Default 0-0: `engine.SettleDefaultResults`
  (`engine/defaultresults.go`), run first by `FinishStuckSimulationsJob`, after
  `engine.ReleaseRedatedFixtures` (`common.ReleaseRedatedFixtures`, `RedatedFixtureMargin` in
  `common/defaultresults.go`) gives back their attempts to exhausted fixtures re-dated to a later kickoff.
- Durable post-match effects: table `pending_post_match_effects` (same migration), written by
  `common.Save*ResultTx`, claimed with `DELETE ... RETURNING` by `engine.completePostMatchEffects`
  (the replay's `onComplete`) and drained by `engine.ApplyDuePostMatchEffects` from
  `LiveBroadcastRecoveryJob` (`common.ClaimDuePostMatchEffects`, `FOR UPDATE SKIP LOCKED`); payload
  `common.PostMatchPayload`. The backend README's live replay section describes the same flow.
- Multi-replica safety (simulation claims, cup tie locks, fixture generation lock): see the
  engine-replicas spec.
- Lineup lock: `common/lineuplock.go` (`LineupLockMinutes`, `IsLineupLocked`); handlers
  `SetMatchLineupHandler`/`SetCupTieLineupHandler`/`SetFriendlyLineupHandler`; frontend
  `useIsLineupLocked` (`frontend/src/api/useLiveMatch.ts`).
- Schedules: `common/leagueschedule.go` (`EffectiveLeagueSchedule`, `GetEffectiveLeagueSchedules`,
  `ListLeaguesForAdmin`), `common/leaguekickoff.go` (`countries.league_kickoff_minute`,
  `GetCountryLeagueKickoff`/`SetCountryLeagueKickoff`, `TierKickoffMinute`,
  `LeagueKickoffTierStagger`), `common/matchschedule.go` (`MatchSchedule.Location`),
  `engine/schedule.go` (`ComputeFixtureDates`/`NextScheduledOccurrence`, dated on the schedule's
  location), `engine/reschedule.go` (`RescheduleUnplayedMatchesForLeagues`/`...ForCountry`, each
  league on its effective schedule); admin `handlers.AdminSetCountryKickoffHandler`
  (`PUT /v1/admin/countries/:code/kickoff`); tests `TestEffectiveLeagueSchedule_*`,
  `TestComputeFixtureDates_LocalKickoffHoldsAcrossDST`, `TestPyramid_CountryKickoffChangeReschedules`, `TestRescheduleUnplayedMatches_LeavesStartedMatchesOfAHalfPlayedRound`; friendly delay `friendlyScheduleDelay` and
  quarter-hour grid `IsFriendlyKickoffSlot`/`nextFriendlyKickoffSlot` (`common/friendlies.go`),
  weekly cap and clash check `CountFriendliesInWeekOf`/`lockTeamFriendliesTx` in
  `acceptFriendlyChallenge` (`common/friendlies.go`, tests `common/friendly_cap_integration_test.go`),
  mirrored by `isQuarterHour`/`kickoffErrorKey` (`frontend/src/utils/friendlyKickoff.ts`).
- Cup kickoffs: `engine.cupTieKickoff` (`cupMinimumLead`, `cupRoundGap`, `engine/cups.go`) with
  `common.NextLocalKickoffAfter` and `NationalCupKickoffHour`/`InternationalCupKickoffHour`
  (`common/timezones.go`); tests `TestCupTieKickoff`, `TestNextLocalKickoffAfter`.
- Added time in the replay: `engine.minutePublishOffset(minute, addedMinute)` (45+N one minute per
  added minute after 45, 46 always at kickoff + 59 minutes, 90+N after 90), `engine.MaxBroadcastDuration`
  and `TestLongestBroadcastFitsLiveWindow` (`engine/livematch_test.go`), which keeps the longest
  replay at least 5 minutes inside `common.LiveBroadcastWindow`. `BroadcastMatchLive` keys events by
  `(minute, addedMinute)` and sends "halftime" after `45+FirstHalfAddedTime` and "complete" after the
  last instant, both carrying `LiveMatchMessage.Minute` / `AddedMinute`
  (`TestBroadcastMatchLive_AddedTime`).
