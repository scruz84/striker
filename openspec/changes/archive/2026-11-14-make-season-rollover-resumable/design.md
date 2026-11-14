# Design

## Steps

The rollover keeps its order. Each step is numbered and recorded once it has completed:

| Step | Work | Recorded as |
|------|------|-------------|
| 1 | `ProgressPlayers` (age, contracts, retirements) | `rolloverStepPlayersProgressed` |
| 2 | `ReturnExpiredLoans` | `rolloverStepLoansReturned` |
| 3 | `applyPromotionRelegation` (prizes, season income, awards, moves, history) | `rolloverStepPromotionApplied` |
| 4 | `restructureIfNeeded` | `rolloverStepRestructured` |
| 5 | `StartCupForCountry` | `rolloverStepCupStarted` |
| 6 | classification reset and starting order (`writeSeasonSeeds`) | `rolloverStepStandingsReset` |
| 7 | `current_season` + 1 | `rolloverStepSeasonAdvanced` |
| 8 | `GenerateMatchesForLeagues` | marker cleared |

Step 6 and step 8 are already safe to repeat (the reset and the seeds rewrite the same rows; a
league with upcoming fixtures is skipped), so they are recorded only to keep the flow uniform.

## Marker

`countries.rollover_season` is the season being closed, `rollover_step` the last step completed
(0 = none) and `rollover_state` (jsonb) the final standings and the moves of steps 3 and 4.

- Every write happens inside the country's session advisory lock (`hashtext(country)`), so
  replicas see one consistent marker.
- A marker counts only for the season it names: `rollover_season = current_season` while the
  rollover is in steps 1 to 6, `rollover_season = current_season - 1` with step 7 once the season
  moved. Anything else is stale and ignored.
- Step 7 is one `UPDATE countries SET current_season = $2, rollover_step = 7` so the season and the
  marker move together.
- A resumed rollover skips the completion checks (matches played, broadcasts over, playoff done):
  they held when the rollover started, and step 3 completes the playoff.

## Failure handling

A step that fails to record itself, or an error between steps that used to `return` with the
work already applied (`GetLeagueIdsForCountry`, `SetCurrentSeasonForCountry`), now stops the
rollover. The next tick resumes at the first step not recorded. Fixture generation that fails
keeps the marker, so the next tick retries it.

## Testing

`advanceCountrySeasonIfComplete` gets a test-only hook (`rolloverInterruptAfter`) that stops the
run right after a given step is recorded. Integration tests stop after step 3 and after step 7,
run the rollover again and check that ages, contract years, prize money and cups changed once and
that the new season has its fixtures.

## Rollback

Reverting the code leaves the three columns unused; a stale marker is ignored by the old code.
