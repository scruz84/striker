# Proposal

## Why

No CPU routine looks at the league table. A Director fighting relegation shops exactly like one
cruising in mid-table, and one with nothing left to play for in the last rounds keeps its
youngsters on the bench although competitive minutes are what makes them grow (player-development
"Match-triggered skill drift"). Human managers react to their season; CPU clubs should too.

## What Changes

- **Season objective**: from the league table, the club's remaining league matches and the
  promotion/relegation slots, every CPU club has an objective once a third of its league season is
  played: `relegation_fight` (in the relegation zone or within 3 points of it), `promotion_push`
  (in the promotion places - the title in a top division - or within 3 points of them),
  `nothing_at_stake` (it can no longer reach those places and can no longer be caught by the
  relegation zone) or `open` (anything else, and always before a third of the season).
- **Market urgency**: a Manager or Director in a relegation fight lowers its minimum score by 2
  (not below 1) and halves its budget reserve; a Director in a promotion push lowers its minimum
  score by 1.
- **Youth minutes**: a Director with nothing at stake starts, at each league kickoff, up to 2 of
  its fit players aged 21 or under from the bench in place of its lowest-valued outfield starters
  at the same slot, when each is worth at least 80% of the starter he replaces.

Out of scope:
- Cup-versus-league priorities and resting players before a cup tie.
- Changing mentality from the table (the kickoff mentality already weighs both XIs).
- Objectives for Caretakers (they stay reactive; they have no market to speed up).

Open questions for the owner:
- Should the objective be shown to players (for example on the opponent preview)? Not in this
  change.
- 3 points as the "within reach" margin and a third of the season as the start are guesses to be
  tuned with the calibration and the CPU level gauges.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cpu-team-management`: new "CPU season objective" and "Youth minutes when nothing is at stake"
  requirements.
- `cpu-transfer-market`: new "Market urgency" requirement.

## Impact

- Backend: new `common/seasonobjective.go` (pure objective, situation reader), `common/ailineup.go`
  (`AiPlanOptions`, youth starts), `engine/aikickoff.go` and `engine/aiteams.go` /
  `engine/aitransfers.go` (objective passed to the kickoff plan and the transfer pass).
- No schema change.
