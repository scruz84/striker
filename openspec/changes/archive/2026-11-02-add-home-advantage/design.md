# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- `runMatchSimulation` (`engine/simulation.go`) builds `MatchSimulation` and runs the per-minute
  pipeline; `InitialSimulation` (`engine/initial_simulation.go`, `calculateQua` + `blendLine`)
  recomputes `HomeQua` / `AwayQua` every minute, and every later stage reads them. The instants
  (`MatchInstants`) are built from those ratings, so the zone duel shows whatever they hold.
- The weather and the crowd are drawn **after** the result is saved: `simulateMatch` calls
  `common.RandomWeather()` then `UpdateTeamFinancesAfterMatch` → `creditTicketRevenue` →
  `stadiumAttendanceAndRevenue` (`engine/stadium.go`), which both draws the crowd and books the
  gate. Cup ties (`engine/cups.go` `simulateCupTie`) and friendlies (`engine/friendlies.go`
  `simulateFriendly`, `creditFriendlyTicketRevenue`) do the same.
- Money must only move after `SaveLeagueMatchResultTx` / `SaveCupTieResultTx` /
  `SaveFriendlyResultTx` commit (match-simulation "Post-match processing").

## Goals / Non-Goals

**Goals:**
- One crowd number per match, drawn once, used for the home advantage, the gate and the report.
- Keep the "nothing is booked if the save fails" guarantee.

**Non-Goals:**
- Changing the attendance model itself (socios "Attendance model" / "Attendance factors").

## Decisions

### D1. Split "draw the crowd" from "book the gate"
`stadiumAttendanceAndRevenue` becomes `drawMatchdayCrowd(dbCtx, homeTeamId, mc) matchdayCrowd`
(`{Attendance, Capacity, Revenue}`, pure apart from its reads) and `creditTicketRevenue` /
`creditFriendlyTicketRevenue` take the drawn `matchdayCrowd` instead of drawing it. The three
`simulate*` functions draw weather and crowd **before** `runMatchSimulation`, pass the crowd in,
and after the save commits credit exactly that crowd and store it in the report flavor.
`UpdateTeamFinancesAfterMatch` takes the crowd instead of the weather.
*Alternative:* compute an "expected" attendance without noise for the advantage and keep drawing
the real crowd afterwards. Rejected: the report would show a crowd that is not the one the
simulation used, and the code would compute attendance twice.

### D2. Multiplier applied after the blend, every minute
`MatchSimulation` gains `HomeAdvantage float32` (1 for World Cup fixtures and when no crowd
could be drawn). `InitialSimulation.Apply` multiplies the nine home fields of `HomeQua` after
`blendLine`, before `calculateDiffs`, so chance volume, chance share and conversion all see it.
Constants in `engine/initial_simulation.go`: `HomeAdvantageBase = 0.08`,
`HomeAdvantageCrowdWeight = 0.14` (retuned from 0.04 / 0.06 by the calibration, task 2.2). `homeAdvantageMultiplier(attendance, capacity)` clamps fill to
[0, 1].
*Alternative:* a flat bonus to the home conversion probability. Rejected: it would not touch the
midfield battle, so the home side would not have more of the ball, and set pieces and open play
would diverge.

### D3. World Cup fixtures are neutral
`simulateFriendly` already knows a friendly is a World Cup group match (its World Cup link) and
`simulateCupTie` knows a tie belongs to a World Cup bracket; both pass `HomeAdvantage = 1`. They
still draw a crowd for the gate and the report, as today.

### D4. Failed saves
A crowd drawn for an attempt whose save fails is discarded with it; nothing is booked, and the
retry draws again. That matches the existing retry semantics (`engine/simulationfailures.go`).

## Risks / Trade-offs

- [Too strong or too weak an effect] → task 2.2 simulates 20,000 matches between identical squads
  and asserts the spec's ranges; constants are retuned (and the spec updated) if they miss.
- [Results shift → prize money, sponsorship bonuses and CPU budgets shift] → economy simulation
  run before/after (task 3.1); the shift is symmetric over a season (every team plays half its
  league matches at home).
- [Big clubs with big stadiums get stronger] → the effect depends on fill, not size, so a small
  club with a full ground gets the same +22%.

## Migration Plan

No schema change. Deploy the engine; matches already simulated keep their results. Rollback is a
redeploy of the previous engine.
