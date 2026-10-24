# Design

## Context

A fixture whose attempts are used up (`simulation_attempts >= 3`) is excluded by every selector
(`simulationAttemptsLeftSql`). It is picked up only by the default claims in
`common/defaultresults.go`, which also require `match_date <= now()`. A fixture is first claimed at
most 15 minutes before kickoff, so on its own date it cannot use up its attempts before then.
If it has, it was re-dated after failing, typically by `engine.RescheduleUnplayedMatches`.

## Decisions

### 1. Release step, one statement per table

`common.ReleaseRedatedFixtures` runs one `UPDATE` per table (`matches`, `friendly_matches`,
`cup_ties`):

```sql
UPDATE matches SET in_simulation = false, simulation_attempts = 0,
  last_simulation_error = NULL, last_simulation_attempt_at = NULL
WHERE played = false AND simulation_attempts >= 3
  AND match_date > now() + interval '15 minutes'          -- common.RedatedFixtureMargin
  AND (in_simulation = false OR in_simulation_since IS NULL
       OR in_simulation_since <= now() - interval '10 minutes')  -- common.StuckSimulationAge
```

Cup ties have no claim, so their statement has no `in_simulation_since` guard. Each release is a
single statement, so replicas running it at the same instant are safe: each row is updated once,
and a second run finds nothing. `engine.ReleaseRedatedFixtures` logs the counts and runs before
`SettleDefaultResults` in `FinishStuckSimulationsJob` and `AdminFinishStuckMatchesHandler`.

*Alternative considered:* resetting attempts inside `RescheduleUnplayedMatches`. That misses rows
already re-dated before this change, and any other way a fixture's date can move.

*Alternative considered:* settling these fixtures 0-0 at once. That would publish a result days
before the fixture's kickoff, and a fixture whose failure was fixed in the meantime would never be
simulated.

### 2. Admin response

`finishStuckMatchesResponse` gains `released` (`common.FixtureKindCounts`). The frontend adds it to
the success message. The `finishStuckMatchesSuccess` key keeps its name and gains the
`{{released}}` placeholder.

## Risks / Trade-offs

- **[Risk] A fixture loops between failing and being released.** A release needs a kickoff more
  than 15 minutes away, and attempts are only counted from 15 minutes before kickoff. So the
  release only repeats if the fixture is re-dated again.
- **[Trade-off] The last error is cleared on release.** The fixture leaves the failing list until
  it fails again. The error was recorded against the old kickoff.
