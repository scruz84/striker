# Spec Delta

## MODIFIED Requirements

### Requirement: Simulation claims and stuck recovery
League matches and friendlies SHALL be claimed (`in_simulation = true`, `in_simulation_since = now()`) in the same transaction that selects them `FOR UPDATE`. Stuck recovery MUST only take rows claimed at least 10 minutes ago (or never stamped) and re-claim them in one statement.

#### Scenario: Recovery fires as a pass claims
- **WHEN** the 15-minute recovery job runs the same second a live pass claims a match
- **THEN** that match is not simulated a second time

#### Scenario: Two replicas recover
- **WHEN** two replicas run stuck recovery at once
- **THEN** each abandoned match is re-simulated by one of them

#### Scenario: Result written twice
- **WHEN** two simulations of the same fixture both reach the result save
- **THEN** only the first save applies, and the second changes neither the result, events, standings nor any budget

## ADDED Requirements

### Requirement: Attempt counting and default result under concurrency
Each claim (live pass, recovery, or cup-tie lock) SHALL increment the fixture's attempt counter in the claiming statement itself. A fixture with 3 counted attempts SHALL be settled 0-0 by one statement that only matches an unplayed row, so several replicas settle it once.

#### Scenario: Two replicas reach the limit together
- **WHEN** two replicas' recovery passes both find the same fixture at 3 attempts
- **THEN** exactly one settles it 0-0, and standings, wages and the round prize are booked once

#### Scenario: Cup tie retried by two replicas
- **WHEN** two replicas run the cup tick while a tie's last failed attempt is 10 minutes old
- **THEN** only the replica holding the tie's lock counts and makes the next attempt

### Requirement: One-off standings repair
The engine SHALL run, once per deployment of this change via a job slot claim, a pass that rebuilds every current-season classification row whose `matches` count differs from the number of played league matches of that team in that league and season. A rebuilt row MUST equal the sum of those played matches.

#### Scenario: Standings double-counted by an earlier failed save
- **WHEN** a team's classification shows 12 matches but it has 11 played league matches this season
- **THEN** its matches, points, wins, draws, losses and goals are recomputed from those 11 matches

#### Scenario: Two replicas start together
- **WHEN** two replicas start the new build at the same instant
- **THEN** only the replica that wins the slot runs the repair

#### Scenario: Consistent standings
- **WHEN** every classification row already matches its played matches
- **THEN** the pass changes nothing
