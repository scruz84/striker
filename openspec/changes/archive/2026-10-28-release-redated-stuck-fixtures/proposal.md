# Proposal

## Why

After `harden-stuck-match-recovery` was deployed, some fixtures still could not be finished: the
admin overview kept counting them as stuck or failing, the "finish stuck matches" action reported
0 re-simulated, 0 defaulted, and the engine log showed nothing.

These fixtures had already used their 3 simulation attempts and were then re-dated to a kickoff
still in the future. `RescheduleUnplayedMatches` re-dates every unplayed match, including one that
is stuck or failing. Every recovery path then skips such a fixture without a word:

- the per-minute selectors and stuck recovery only take fixtures with attempts left;
- the default 0-0 only takes fixtures whose kickoff has passed.

So the fixture sits there until its new kickoff (days, for a rescheduled round). It is then
settled 0-0 without ever being simulated on the code that fixed its original failure.
Reproduced with an integration test: four passes of `SettleDefaultResults` and
`FinishStuckSimulations` leave the fixture unplayed, and the overview keeps `stuck = 1, failing = 1`.

## What Changes

- **Re-dated fixtures get their attempts back.** A new step, `ReleaseRedatedFixtures`, runs before
  the default settlement in the 15-minute recovery job and in the admin action. It covers league
  matches, friendlies and cup ties. If a fixture is unplayed, has used its 3 attempts, and kicks
  off more than 15 minutes from now, the step releases its claim and resets its attempts, last
  error and last-attempt time. The fixture leaves the stuck and failing counts, is simulated
  normally at its new kickoff, and is settled 0-0 only if it fails 3 more times after that.
- **Admin action result.** The action's response gains `released` per kind, and its success
  message reports it (en/es).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-scheduling`: "Default result after repeated failures" adds the re-dated case.

## Impact

- Backend: `common/defaultresults.go`, `engine/defaultresults.go`, `engine/tasks.go`,
  `handlers/adminHandlers.go`.
- Frontend: `api/types.ts`, `features/admin/OverviewTab.tsx`, and
  `admin.actions.finishStuckMatchesSuccess` in `en.json` / `es.json`.
- No migration, no new `TRANSACTION_*` reason, no new notification key.

## Out of scope

- Making the admin action skip the 10-minute retry backoff. A league match or friendly that failed
  less than 10 minutes ago is still retried by the per-minute tick, not by the button.
- Changing `RescheduleUnplayedMatches` itself. The release step covers every way a fixture can be
  re-dated, including rows re-dated before this change.
- Root-causing the original simulation error of the affected fixtures. If it still fails at the new
  kickoff, the admin overview shows the error again.

## Open questions for the owner

- Proposed default, written into the spec: a re-dated fixture gets a fresh set of 3 attempts at its
  new kickoff instead of being settled 0-0 right away. Settling it now would reveal a result before
  its kickoff.
