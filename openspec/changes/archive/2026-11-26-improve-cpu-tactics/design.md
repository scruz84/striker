# Design

## Context

`common.SelectBestFormationAndLineup` builds the best XI for each formation and ranks them by
`lineupStrength` (sum of `PositionValue` at each slot); `formationRankForAiLevel` decides the rank
per level. `engine.ProjectZoneDuel` already runs `StaminaSimulation` and `InitialSimulation` on two
XIs for the opponent preview.

## Decisions

### Expected goals
`engine.ProjectExpectedGoals(own, opponent, homeAdvantage, ownIsHome)` reads the duel's zones and
goalkeepers and computes, per side, `90 × chanceProbability × (1 − set-piece share) × share ×
E[conversion]`. `E[conversion]` averages over the 50/25/25 zone split (with
`flankMidfieldSupport` on the flanks), the 70/20/10 shooter mix of `getPlayerForChance` (its
line fallbacks included, wide positions never shoot) and the clear-cut/half-chance mixture of
`chanceQualityMultiplier` feeding `chanceSuccessProbability`. All constants are the engine's.
A sanity test checks that over seeded simulated matches the side with the higher projected goal
difference does better at least 80% of the time.

### Director re-pick
`chooseFormationAgainst(available, opponent, homeAdvantage, ownIsHome)` is pure. In
`planAiKickoffs`, after both sides' first plans, each planned Director side of a league match or
cup tie calls `directorFormationAgainst` against the other side's first-pass XI and, when the
formation differs, is re-planned with `planAiKickoffWithFormation` (the plan's
`AiPlanOptions.Formation`), which re-runs the same validation. Both re-picks read the first-pass
XIs, so the order of the two sides doesn't matter. The projection is neutral (home advantage 1):
it multiplies every zone of one side alike and so barely changes which shape is best, and it
saves a crowd model read per kickoff.

### Manager
`formationRankForAiLevel` returns 0 for Managers; their remaining gap to Directors is the opponent
read, the 2-vs-3 substitutions and the cup orders.

### Calibration
`measureAiLevel` mirrors the re-pick for Directors.

## Risks / Trade-offs

- Six zone projections per Director kickoff: each is two pipeline stages for one minute, well
  below a millisecond.
- The projection ignores fatigue and mentality; the calibration shows the picks still pay off.

## Migration

None.
