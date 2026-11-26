# Design

## Context

See proposal.md (Why) for the findings. The current state that constrains the approach:

- `calculateQua` (`engine/initial_simulation.go`) builds three central totals: CB Defense, CM
  Midfield, and Shot over ST plus lateral strikers. It also builds three flank values from single
  players: the full-back for defense and midfield, and the lateral striker or lateral midfielder
  for attack. `blendLine` then mixes each line, using an integer `centerCount`, and its spillover
  to the flanks uses the center's per-player average, so the head count never reaches the flanks.
- Chance resolution compares attack and defense as ratios: the conversion logistic takes the
  difference over the pair's mean, and the possession share takes the midfield difference over the
  total. Two consequences shape this design:
  - Scaling a whole line by a factor changes nothing relative to another formation, so a squad
    whose every player is better at one skill does not prefer a different shape.
  - The defense value (central defense plus the goalkeeper) is already large next to the attack
    value, so extra defense has diminishing returns.
- `newSideDuelists`/`playerContributions` mirror `calculateQua` player by player for the viewer's
  zone duel. `ProjectZoneDuel` and `ProjectExpectedGoals` run `calculateQua` itself, so the
  opponent preview and the Director's projection follow the engine automatically, except for
  `shooterMix`, which copies `getPlayerForChance` by hand. The projection leaves out set pieces,
  fatigue and cards, which slightly undervalues midfield.
- `PositionValue` (`common/lineup.go`, via `common/fmteam.go`) is hand-written per slot and has
  hand-kept copies: `positionSkillSQL` (`common/aimarket.go`), `driftSkillsForPosition`
  (`engine/progression.go`) and the client's `midfieldValue`/`strikerValue`
  (`frontend/src/utils/playerRating.ts`).
- `SelectBestFormationAndLineup` lives in `common` and is called from `engine/aiteams.go`
  (`refreshAiTeamLineup`), `common/ailineup.go` (`PlanAiMatchLineupWithOptions`) and
  `common/nationalteam.go`. The projection lives in `engine`, which `common` cannot import.
- Every number below comes from prototyping in a throwaway worktree, with the tunables as package
  variables and the study harness plus specialist-squad blocks. The production code uses named
  constants.

## Goals / Non-Goals

**Goals:**

- A zone model in which a formation only moves strength between lines and flanks, and pays off
  when the squad has the players that formation fields more of.
- Slot values that rank players the way the engine rewards them.
- The targets in the "Formation balance" and "Squad fit" requirements are checked by the harness,
  the same way `HOME_ADV_SIM` and `AI_LEVEL_SIM` check theirs.

**Non-Goals:**

- No new formations and no per-formation tuning constants.
- No change to chance rate, possession share, conversion curve, set pieces, fatigue, home
  advantage or mentality constants.
- Not removing the ratio invariance or defense's diminishing returns (see proposal, "Known
  limits"). Both would need a change to the conversion and possession formulas, which reach every
  calibration in the game.

## Decisions

### 1. How the study measured "no formation loses out"

Balance alone is not enough. A model can be even on average and still make the squad irrelevant:
before the rebalance, 4-3-3 was the best shape for 100% of squads whatever players they had.

Four measurements are used together:

1. **Even squads:** a round robin with every skill at 50, which is the "Formation balance"
   requirement.
2. **Generalist random squads:** each formation's mean projected goal difference against the same
   squad's 4-4-2 should be close to 0. Random squads are generalists, so the gap between formations
   barely varies from squad to squad, by about ±0.02 xG. A structural offset of only 0.03 is then
   enough to swing a formation between 0% and 80% "best for".
3. **Specialist squads:** random squads plus two or three pure specialists (75 in their role
   skills, 25 elsewhere), or with one line capped at 30. This is the "Squad fit" requirement.
   - Each formation should become the best projected shape for a fair share of its profile's
     squads.
   - Its edge over 4-4-2 should be larger with specialists at 75 than at 40.

4. **Realistic squads:** random squads plus 2 to 4 partial specialists of random types. They are
   used twice:
   - In full matches, every formation should score like the 4-4-2 mirror. In the prototype all
     six are within 1.34–1.40 against a 1.38 mirror (10,000 matches each, ±0.013).
   - In the projection, no formation should be the natural choice for most squads. The prototype
     still has 4-3-3 as the best shape for 38% of them, because it profits from more kinds of
     specialists.

Projection and full matches have to be read together. A formation can be "best" most often with a
mean edge of zero. In the reverse direction, small Monte Carlo samples are noisy (±0.03 to 0.04
points per match at 2,000 matches), so every points-per-match bound uses 10,000 matches.

Uniform "every player +12 in Defense" profiles were tried first. They move nothing, because of the
ratio invariance, so the requirement is written in terms of specialists. In football terms, a
5-3-2 pays off when you have three good centre-backs, not when all your players defend a bit
better.

### 2. Light, symmetric line cover

Each central player weighs 1 in his own line and `LineCoverWeight = 0.1` in each adjacent central
line: CB → midfield, CM → defense and attack, ST → midfield. Central totals become weighted sums,
and `blendLine` takes a float weight sum instead of `centerCount`.

The cover weight was measured at 0, 0.1, 0.2 and 0.4:

| Cover weight | Even squads | Specialist fit |
|---|---|---|
| 0 | 5-3-2 stays dominated | — |
| 0.2 | Best even-squad balance (spread 0.07 xG) | 5-3-2 and 3-4-3 are best for 0% of squads, even with their specialists |
| 0.4 | — | A pure centre-back's Midfield of 25 pulls a 5-3-2's midfield down, so three great centre-backs helped a 4-4-2 (+0.12 points per match) more than a 5-3-2 (+0.02) |
| **0.1** | Keeps even squads balanced | Keeps specialists useful |

0.1 combined with decisions 3 to 5 was chosen. Alternatives rejected:

- Hard-coded per-formation multipliers: opaque, and they must be retuned on every engine change.
- Dropping only the lateral-striker fold: 4-3-3 becomes structurally the same as 4-5-1, and 5-3-2
  stays dominated.

### 3. Defense head count reaches the flanks

After `blendLine`, each flank defense gains `ZoneSpillover × centralAverage × (W ÷ 2 − 1)`, where
W is the central defense's weight sum. A 4-4-2 is close to unchanged, a back five also shuts its
flanks, and a back three leaves them a little more exposed.

This was the lever that let three good centre-backs make a 5-3-2 worth playing. Without it the
squads with centre-back specialists picked 5-3-2 only 9% of the time; with it, 26%. Making central
chances more frequent (60% instead of 50%) was also measured. It made 4-5-1 never best and changed
the chance model's calibration, so it was rejected.

### 4. Wide players and back-three wing-backs

- `LateralStrikerCentralWeight = 0.5` replaces the full fold of the lateral strikers into the
  central attack.
- `LateralMidfieldCentralWeight = 0.65`: a wide midfielder tucks in.
- `LateralMidfieldFlankShare = 0.5`: a flank's midfield is the mean of the full-back's and the
  wide midfielder's flank-midfield values. It is the full-back's value alone where the formation
  has no wide midfielder (4-3-3).
- `BackThreeWingBackCentralWeight = 0.15`: with a single DEFENSE slot, each full-back adds that
  share of his Defense to the central defense. At 0.3, 3-4-3 and 3-5-2 move ahead of the other
  shapes by about 0.08 xG, so 0.15 was kept.
- Lateral strikers, lateral midfielders and full-backs join the striker, midfielder and defender
  shooter pools in `getPlayerForChance`, and `shooterMix` is changed the same way.

### 5. Pass and Dribbling enter the line values

- `MidfieldPassWeight = 0.3`: the midfield value is 0.7 × Midfield + 0.3 × Pass.
- `AttackDribblingWeight = 0.25`: the attacking value is 0.75 × Shot + 0.25 × Dribbling, used for
  the central attack and in the wide attack's 0.5 × Lateral + 0.5 × attacking value.
- `getPlayerScorerValue` (the finisher's own value) is unchanged.

### 6. One slot-weight table for `PositionValue` and its copies

A new `common/slotweights.go` holds `SlotWeights` (a fixed array of seven integer percentages
per position) plus `PositionValue = Σ skill × weight / 100`, rounded. The weights are the
rounded shares of the harness's marginal xG table, measured on the final prototype: +10 in one
skill for one player at that slot, in a 4-4-2, or a 4-3-3 for the wide striker. They are listed
in the player-development delta's "Slot weights" scenario. The goalkeeper stays GoalKeeper only.

The copies are derived from the table rather than kept by hand where possible:

- `positionSkillSQL` is generated from `SlotWeights` at package init.
- `driftSkillsForPosition` drifts every skill whose weight is at least `DriftSkillMinWeight = 15`.
- `GetMidfieldValue`, `GetStrikerValue` and the `GetLateral*Value` helpers become thin wrappers,
  or are removed where `PositionValue` covers the caller.
- The client's `playerRating.ts` keeps a hand-written copy of the three core outfield rows
  (DEFENSE, MIDFIELD, STRIKER) for `bestCorePosition`. A Vitest case pins it to the same numbers.

Because lineup selection uses `PositionValue`, the squad-fit figures depend on these weights.
Task 5 re-measures them once the final table is in place.

Alternative considered: deriving the weights analytically from the engine constants. Rejected,
because a slot's effective weight depends on the formation, the spillover and the conversion
curve.

### 7. CPU formation ranking by projected expected goals

`SelectBestFormationAndLineup` ranks formations by a score supplied by the engine. Its score is
the expected goal difference from `ProjectExpectedGoals`, rating each formation's best XI against
the same squad's best 4-4-2 XI on neutral ground. Ties go first to `lineupStrength`, then to
`AllFormations` order.

- Because `common` cannot import `engine`, `common` gets a registration hook:
  `SetFormationScorer(func(squad []*Player, lineup *Lineup, reference *Lineup) float64)`.
  `engine` registers its scorer from an `init()`.
- When no scorer is registered, as in common's own unit tests, the ranking falls back to
  `lineupStrength`.
- Threading a function argument through the three call chains instead was rejected: it touches
  many signatures for one value.
- Moving the projection into `common` was rejected: it depends on the engine's simulation types.
- National teams call at Director level, so they also pick their shape by projection.

Ranking by summed slot values would undo the squad-fit work: it picks 5-3-2 for about two thirds
of random squads whatever their players.

### 8. Best XI by optimal assignment

`SelectBestLineupForFormation` fills slots in `positionFillOrder` (goalkeeper, back line,
midfield, attack), each time with the best remaining player for that slot. A player whose best
slot comes later can be taken by an earlier one. In the study, wide-midfield specialists (Lateral,
Midfield and Pass 75) always ended up at full-back, so the "2 pure LM" profile was really
measuring full-backs.

It becomes an optimal assignment: the eleven players and slots that maximize the total
`PositionValue`. Ties go to the lower player id, so the result stays deterministic.

- A squad is at most a few dozen players for 11 slots. An exact Hungarian assignment over that
  matrix costs microseconds, so there is no need for a heuristic.
- The match-day backfill (`BackfillEmptyPositions`) keeps filling single holes greedily. It
  replaces one or two unavailable starters, where the greedy choice is the optimal one.
- Every caller gets the new XI: CPU lineups, the friendly fallback, a new team's default lineup
  and national teams. A human's saved lineup is never re-picked.

### 9. Harness becomes the balance and fit calibration

`engine/tactics_study_test.go` gains:

- assertions for "Formation balance" in its `rr`, `xg` and goal-volume blocks;
- a `fit` block with the specialist profiles, asserting "Squad fit";
- a `fitmc` block with the good-versus-poor centre-back Monte Carlo;
- a `real` block with the realistic squads, asserting the full-match bound and the share bounds
  (no formation above 30%, none below 8%).

All of them still run only with `TACTICS_SIM=1`, like the other calibrations.

## Risks / Trade-offs

- [Every club's strength shifts mid-season, and CPU Directors lose most of their edge: 1.695 →
  1.421 points per match against a Caretaker, level with a Manager's 1.423, since both now rank
  formations by projection] → The owner accepted the tie. `TestAiLevelCalibration` checks that a
  Director is never more than 0.05 points per match behind a Manager. Compensating the AI is out of
  scope.
- [Squad fit is modest. A good set of specialists is worth about 0.03 to 0.14 points per match
  for the right formation] → This reflects how much any single player moves the engine today. The
  requirement guards the direction and a minimum size; making players matter more is a separate
  design question.
- [The prototype misses the realistic-squad share bounds: 4-3-3 38%, 5-3-2 5%, 4-5-1 6%] → The
  best-XI assignment changes who plays where, so it lands first. Task 5.5 then narrows the gap
  between formations' fit gains (4-3-3 +0.13 xG with two wingers, 5-3-2 +0.02 with three
  centre-backs) using the constants of decisions 2 to 4. If a bound still cannot be met without
  breaking "Formation balance", the outcome is recorded and the owner decides whether to relax it.
- [4-5-1 has the weakest fit, 18% best with no strikers] → Part of the same tuning in task 5.5.
- [Players whose value rested on Pass, for example central midfielders, lose some slot value;
  wide midfielders gain from Shot] → Wages and market value come from the seven skills
  (`PlayerWageFromSkills` via `MarketValue`), not from `PositionValue`, so no money moves. Only
  CPU lineup, transfer and sale judgements change.
- [The client's `bestCorePosition` copy can drift from `SlotWeights`] → The Vitest case pins it,
  and the Code map names both.
- [The zone duel's per-player contributions show weighted figures, so a full-back now appears in
  two duels and a wide midfielder in up to three] → `newSideDuelists` keeps "own value in the
  duel / rival per-player value", with the per-player value being the weighted total over the
  weight sum. Existing viewer tests are updated with the new figures.
- [Saved lineups were built under the old weights] → They stay valid; only their ratings change.

## Migration Plan

- No schema change, no migration file, no `TRANSACTION_*` reason, no notification key.
- The change only affects matches simulated after deploy. Matches already simulated keep their
  stored instants and replay unchanged.
- Rollback is a code revert; no data has to be undone.

## Open Questions

- Whether `LineCoverWeight` should later differ per direction to give mentality-style
  trade-offs. Deferrable: it changes only constants, not the approach.
