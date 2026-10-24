# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Default result after repeated failures
A league match, friendly or cup tie that has already used 3 simulation attempts without being played MUST NOT be simulated again. The next recovery pass, run once kickoff has passed, SHALL settle it as a 0-0 default result instead (see match-simulation for what that result books).

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
