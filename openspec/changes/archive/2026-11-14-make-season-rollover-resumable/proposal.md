# Proposal

## Why

`engine.advanceCountrySeasonIfComplete` closes a country's season as a long run of steps with no
common transaction, and only moves `countries.current_season` near the end (issue #401). The
country lock stops two replicas running it at once, but not a run that stops halfway: a restart
during a long rollover (a deploy during a restructure), or `GetLeagueIdsForCountry` /
`SetCurrentSeasonForCountry` failing, leaves the season where it was with every match played, and
the next tick runs the whole rollover again. Players then age twice and lose two contract years,
league prizes and season income are paid twice, a second cup is created and promotion is
recomputed on a table that was already reset. If the stop comes after the season bump but before
the fixtures, the new season has no matches and never starts.

## What Changes

- **Progress marker per country.** `countries` gets `rollover_season`, `rollover_step` and
  `rollover_state` (`init116_resumable_season_rollover.sql`). Each step of the rollover records
  itself as done, under the country lock, right after it completes.
- **Resume, don't repeat.** A rollover that finds a marker for the current season skips the steps
  already done and goes on from the next one. The final standings and moves that the later steps
  need are kept in `rollover_state`, so they are not recomputed from a table that has moved on.
- **The season bump records the last step.** The `UPDATE` that moves `current_season` also marks
  the fixtures as the only step left; generating them clears the marker. A rollover stopped
  between the two generates the fixtures on the next tick.
- Spec `engine-replicas`: the rollover requirement names the marker, and a new requirement covers
  resuming.

## Out of scope

- A step that stops halfway through its own work (for example in the middle of `ProgressPlayers`)
  is run again from its start on resume. Making each step transactional on its own is a larger
  change; the marker removes the repeats between steps, which is where every error return and most
  of the elapsed time of a rollover are.
- Paying back or undoing a rollover that already ran twice before this change.
