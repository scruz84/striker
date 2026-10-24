# Proposal

## Why

Some matches stay unplayed and the app cannot clear them, either automatically or with the admin
"finish stuck matches" button. Stuck recovery only handles a process that crashed mid-simulation.
When a simulation returns an error or panics, the row stays claimed (`in_simulation = true`,
`played = false`). The 15-minute job then re-runs it forever, with no attempt limit and no
recorded reason. One such league match blocks its country's season rollover. One national-team
friendly blocks every national-team fixture.

The same review found failure modes that corrupt data instead of blocking it:

- `UpdatePlayedMatch` logs a failed `UPDATE matches` and returns. `simulateMatch` then still books
  standings, ticket revenue and wages, and broadcasts the match. Recovery later re-simulates it,
  so standings and money are counted twice (once more per further failure).
- Friendlies and cup ties set `played = true` before inserting events and instants, outside a
  transaction. A failed insert leaves a played match with no events, no replay and no
  post-match effects.
- The post-match effects (availability countdown, new injuries and suspensions, skill growth,
  result notification) run in the replay goroutine. An engine restart during the ~105-minute
  replay loses them for good. Stale-broadcast recovery closes the channel but never runs them.
- The admin overview counts every claimed unplayed row. The button only takes rows claimed at
  least 10 minutes ago, and every retry restamps that clock, so the button often does nothing.
  It also reports rows attempted, not rows resolved. Cup ties are neither counted nor covered.

## What Changes

- **Atomic result save.** A league match's result, events, instants and standings are written in
  one transaction that only applies when the row is still unplayed. Friendlies (with World Cup
  group standings) and cup ties get the same treatment. Money moves only after that commit
  succeeds, so a failed save can never be counted twice.
- **Bounded attempts.** League matches, friendlies and cup ties count their simulation attempts
  and keep the last error. A failed attempt is retried no sooner than 10 minutes later.
- **Default 0-0 result.** A match that has used its 3 attempts is settled as 0-0 instead of being
  retried again. It gets a new `MATCH_DEFAULT_RESULT` event (shown in the viewer). A league match
  counts as a draw in the standings and still charges matchday wages, with no gate or sponsorship
  income. A cup tie goes to the existing coin-flip shoot-out and pays the round prize. A World Cup
  group friendly counts as a draw. Both owners get the usual result notification.
- **Already-stuck rows are released.** The migration marks every row that has been claimed for
  more than an hour as having used 2 of its 3 attempts. After deploy each one gets one retry on
  the fixed code, then the default 0-0. No admin action is needed.
- **Standings repair.** A one-off engine pass finds classification rows whose `matches` count
  differs from the team's played league matches that season. It rebuilds those rows from the
  played matches, which fixes standings already double-counted by the bug above. The admin
  overview shows how many rows are still out of sync.
- **Durable post-match effects.** Saving a result also writes a pending post-match record in the
  same transaction. The replay's completion claims and applies it. A 5-minute job applies any
  record still pending 2 hours after kickoff (the live-broadcast window), so a restart during
  the replay no longer loses injuries, skill growth or notifications.
- **Admin overview and action.** They use the same rule and cover cup ties. The action reports
  how many rows were re-simulated, defaulted to 0-0 and still failing. The overview lists failing
  matches with their attempt count and last error.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-scheduling`: "Recovery from interrupted simulations" now covers failed (not only
  crashed) simulations, adds the attempt limit and the default 0-0 result, and recovers lost
  post-match effects. "Spoiler-free side effects" stays unchanged in behavior, but those effects
  become durable.
- `engine-replicas`: "Simulation claims and stuck recovery" adds attempt counting, cup ties and
  the default result under the same single-statement claim, plus the one-off standings repair.
- `match-simulation`: "Post-match processing" makes the result save atomic and only-once, and
  adds the default-result event to the stored match record.

## Impact

- Backend: `engine/engine.go`, `engine/simulation.go`, `engine/database.go`, `engine/cups.go`,
  `engine/friendlies.go`, `engine/livematch.go`, `engine/livematch_recovery.go`,
  `engine/tasks.go`, `common/database.go`, `common/friendlies.go`, `common/cups.go`,
  `common/fmconstants.go`, `handlers/adminHandlers.go`.
- Schema: new migration `resources/sql/init86_simulation_failures.sql`. It numbers after
  `init85_observability_indexes.sql`, which the in-flight `expand-observability-dashboard` change
  reserves. Renumber whichever change lands second.
- Frontend: the admin overview (failing-match list, new action result), the match viewer and
  commentary for event type 7, and the en/es locales.
- The observability change's `striker.matches.in_simulation` gauge keeps working. A failing row
  now spends most of its time unclaimed, so that gauge drops and the new admin list is the place
  to look.

## Out of scope

- Making `createRoundTies` transactional. A crash midway through creating a cup round leaves a
  partial bracket: that corrupts the cup but does not block it. It belongs in its own change.
- Root-causing any specific deterministic simulation error. This change makes such errors
  visible (last error in the admin list) and stops them blocking the game.
- Rebuilding a match's lost ticket revenue if the engine dies between the result commit and the
  budget update. Missing a payment is accepted in exchange for never paying twice.

## Open questions for the owner

- Decided by the owner: a match that keeps failing is settled 0-0.
- Proposed default, written into the specs, to be confirmed: a 0-0 default league match counts
  the matchday against suspended or injured players, so a suspension isn't extended by a match
  nobody played.
- Proposed default, written into the specs, to be confirmed: the limit is 3 attempts at least 10
  minutes apart, so the default lands about 30–50 minutes after the first attempt. Connection
  errors do not count towards the limit, so a database outage cannot default a whole round
  to 0-0.
