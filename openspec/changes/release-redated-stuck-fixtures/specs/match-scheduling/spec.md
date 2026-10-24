# Spec Delta

## MODIFIED Requirements

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

#### Scenario: Exhausted fixture re-dated to a later kickoff
- **WHEN** a league match that used its 3 attempts is rescheduled to kick off in 2 days
- **THEN** the next recovery pass, or the admin "finish stuck matches" action, releases its claim and sets its attempts to 0, so it no longer counts as stuck or failing
- **AND** it is simulated normally at its new kickoff, and settled 0-0 only if it then fails 3 more times

#### Scenario: Exhausted fixture kicking off soon
- **WHEN** a league match that used its 3 attempts kicks off in 5 minutes
- **THEN** it keeps its attempts and is settled 0-0 by the first recovery pass after kickoff
