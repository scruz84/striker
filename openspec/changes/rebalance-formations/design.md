# Design

## Context

See proposal.md (Why) for the findings. The current state that constrains the approach:

- `calculateQua` (`engine/initial_simulation.go`) builds three central totals: CB Defense, CM
  Midfield, and Shot over ST plus lateral strikers. It also builds three flank values from single
  players: the full-back for defense and midfield, and the lateral striker or lateral midfielder
  for attack. `blendLine` then mixes each line, using an integer `centerCount`.
- `newSideDuelists`/`playerContributions` mirror `calculateQua` player by player for the viewer's
  zone duel. `ProjectZoneDuel` and `ProjectExpectedGoals` run `calculateQua` itself, so the
  opponent preview and the Director's projection follow the engine automatically, except for
  `shooterMix`, which copies `getPlayerForChance` by hand.
- `PositionValue` (`common/lineup.go`, via `common/fmteam.go`) is hand-written per slot and has
  hand-kept copies: `positionSkillSQL` (`common/aimarket.go`), `driftSkillsForPosition`
  (`engine/progression.go`) and the client's `midfieldValue`/`strikerValue`
  (`frontend/src/utils/playerRating.ts`).
- `SelectBestFormationAndLineup` lives in `common` and is called from `engine/aiteams.go`
  (`refreshAiTeamLineup`), `common/ailineup.go` (`PlanAiMatchLineupWithOptions`) and
  `common/nationalteam.go`. The projection lives in `engine`, which `common` cannot import.
- The prototype for every number below is the study harness in a throwaway worktree, with the
  tunables as package variables. The production code uses named constants.

## Goals / Non-Goals

**Goals:**

- One zone model in which a formation only moves strength between lines and flanks.
- Slot values that rank players the way the engine rewards them.
- The balance targets in the "Formation balance" requirement are checked by the harness, the same
  way `HOME_ADV_SIM` and `AI_LEVEL_SIM` check theirs.

**Non-Goals:**

- No new formations and no per-formation tuning constants.
- No change to chance rate, possession share, conversion curve, set pieces, fatigue, home
  advantage or mentality constants.

## Decisions

### 1. Symmetric line cover rather than per-formation multipliers

Each central player contributes weight 1 to his own line and `LineCoverWeight = 0.2` to each
adjacent central line: CB → midfield, CM → defense and attack, ST → midfield. Central totals
become weighted sums; `blendLine` takes a float weight sum instead of `centerCount`, both for
averaging and for scaling back up.

Alternatives considered, all measured with the harness (spread = max − min of the six formations'
mean xG difference, with identical and random squads):

| Option | Spread | Goals (xG, 4-4-2 mirror) |
|---|---|---|
| Today | ≈1.2 | 2.71 |
| Drop the lateral-striker fold only | 4-3-3 becomes structurally identical to 4-5-1; 5-3-2 still dominated | — |
| Cover only in the defensive direction (CM→D, CB→M, ST→M) | 0.05 | 2.48 (−8% goals) |
| **Symmetric cover K = 0.2** | **0.07** | **2.71 (unchanged)** |
| Hard-coded multipliers per formation | — (rejected: opaque, must be retuned on every engine change) | — |

The symmetric option was chosen. It keeps goal volume without retuning the conversion constants,
and it uses one constant instead of four. Cover is also what fixes 5-3-2: its three centre-backs
and two strikers feed its lone midfielder's line.

### 2. Wide players

- `LateralStrikerCentralWeight = 0.35` replaces the full fold into the central attack. The study
  tried 0.25, 0.35 and 0.5; at 0.35 the 4-3-3's mean xG difference is 0.00 with random squads,
  where 0.25 gives −0.03 and 0.5 gives +0.05.
- `LateralMidfieldCentralWeight = 0.5`: a wide midfielder tucks in. Values of 0.4 to 0.6 behave
  alike and 0.5 is the round one.
- `LateralMidfieldFlankShare = 0.5`: a flank's midfield is the mean of the full-back's and the
  wide midfielder's flank-midfield values. It is the full-back's value alone where the formation
  has no wide midfielder (4-3-3). With uniform squads this changes no formation's ratings; it only
  makes a wide midfielder's Midfield and Lateral count.
- Lateral strikers, lateral midfielders and full-backs join the striker, midfielder and defender
  shooter pools in `getPlayerForChance`, and `shooterMix` is changed the same way.

### 3. Pass and Dribbling enter the line values

- `MidfieldPassWeight = 0.3`: the midfield value is 0.7 × Midfield + 0.3 × Pass.
- `AttackDribblingWeight = 0.25`: the attacking value is 0.75 × Shot + 0.25 × Dribbling, used for
  the central attack and in the wide attack's 0.5 × Lateral + 0.5 × attacking value.
- `getPlayerScorerValue` (the finisher's own value) is unchanged.

### 4. One slot-weight table for `PositionValue` and its copies

A new `common/slotweights.go` holds `SlotWeights` (a fixed array of seven integer percentages
per position) plus `PositionValue = Σ skill × weight / 100`, rounded. The weights are the
rounded shares of the harness's marginal xG table: +10 in one skill for one player at that slot,
in a 4-4-2, or a 4-3-3 for the wide striker. They are listed in the player-development delta's
"Slot weights" scenario. The goalkeeper stays GoalKeeper only.

The copies are derived from the table rather than kept by hand where possible:

- `positionSkillSQL` is generated from `SlotWeights` at package init.
- `driftSkillsForPosition` drifts every skill whose weight is at least `DriftSkillMinWeight = 15`.
- `GetMidfieldValue`, `GetStrikerValue` and the `GetLateral*Value` helpers become thin wrappers,
  or are removed where `PositionValue` covers the caller.
- The client's `playerRating.ts` keeps a hand-written copy of the three core outfield rows
  (DEFENSE, MIDFIELD, STRIKER) for `bestCorePosition`. A Vitest case pins it to the same numbers.

Alternative considered: deriving the weights analytically from the engine constants. Rejected,
because a slot's effective weight depends on the formation, the spillover and the conversion
curve. The harness's `weights` block prints the measured table next to `SlotWeights`, so a later
engine change shows how far the table has drifted.

### 5. CPU formation ranking by projected expected goals

`SelectBestFormationAndLineup` ranks formations by a score supplied by the engine. Its score is
the expected goal difference from `ProjectExpectedGoals`, rating each formation's best XI against
the same squad's best 4-4-2 XI on neutral ground. Ties go first to `lineupStrength`, then to
`AllFormations` order.

- Because `common` cannot import `engine`, `common` gets a registration hook:
  `SetFormationScorer(func(squad []*Player, lineup *Lineup, reference *Lineup) float64)`.
  `engine` registers its scorer from an `init()`.
- When no scorer is registered, as in common's own unit tests, the ranking falls back to
  `lineupStrength`, so those tests keep their current behaviour.
- Threading a function argument through the three call chains instead was rejected: they cross
  `PlanAiMatchLineupWithOptions` and the national-team call-up, so it touches many signatures for
  one value.
- Moving the projection into `common` was rejected: it depends on the engine's simulation types.
- National teams call at Director level, so they also pick their shape by projection.

Measured with random squads, a Manager's first pick becomes 4-4-2 51%, 4-3-3 29%, 4-5-1 15% and
3-5-2 4%. With summed slot values it was 5-3-2 66%.

### 6. Harness becomes the balance calibration

`engine/tactics_study_test.go` gains assertions for the "Formation balance" requirement in its
`rr`, `xg` and goal-volume blocks. They still run only with `TACTICS_SIM=1`, like the other
calibrations.

## Risks / Trade-offs

- [Every club's strength shifts mid-season, and CPU Directors lose most of their edge (proposal:
  1.695 → 1.438 points per match against a Caretaker)] → This is the owner's open question;
  compensating the AI is out of scope.
- [Players whose value rested on Pass, for example central midfielders, lose some slot value;
  wide midfielders gain from Shot] → Wages and market value come from the seven skills
  (`PlayerWageFromSkills`), not from `PositionValue`, so no money moves. Only CPU lineup, transfer
  and sale judgements change.
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

- Whether `LineCoverWeight` should later differ per direction (for example a stronger CM →
  defense) to give mentality-style trade-offs. Deferrable: it changes only constants, not the
  approach.
