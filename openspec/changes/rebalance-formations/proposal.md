# Proposal

## Why

The tactics study harness (`TACTICS_SIM=1`, `engine/tactics_study_test.go`) shows that the
formation a team picks decides more than its players.

- **4-3-3 dominates.** With identical squads it averages 1.73 points per match against the other
  formations, which all sit between 1.19 and 1.39. A Director picks it for 400 of 400 random
  squads. Its two lateral strikers count both on their flank and in the central attack, so its
  nine zones add up to 700 where every other shape adds up to 600.
- **5-3-2 is dominated.** It averages 1.19 points per match because its single central midfielder
  loses the midfield battle outright. Yet a Manager ranks it first for half of all random squads.
- **The CPU's player valuation disagrees with the engine.** `PositionValue` is the "value at
  slot" that CPU lineups, substitutions, transfers and skill drift all use. It weights Pass at
  50% for a central midfielder and Pass plus Dribbling at 50% for a striker, while the engine
  barely uses either. It also rates a wide midfielder on Midfield and Pass, while the engine
  rates him on Lateral and Shot only.
- **Wide players never take an open-play shot.** The lone striker of a 4-3-3 or 4-5-1 takes 70% of
  his side's chances.

## What Changes

The zone model is changed so that every formation trades strength between lines rather than
gaining it outright:

- **Line cover.** Each central defender, midfielder and striker lends 20% of himself to each
  adjacent central line: a defender to the midfield, a midfielder to the defense and the attack,
  and a striker to the midfield.
- **Wide forwards.** A lateral striker counts 35% (was 100%) toward the central attack.
- **Wide midfielders.** A lateral midfielder counts 50% toward the central midfield and supplies
  half of his flank's midfield rating; the full-back supplies the other half.
- **Pass and Dribbling.** A central midfielder's value is 70% Midfield and 30% Pass. A striker's
  attacking value is 75% Shot and 25% Dribbling, which also enters the wide attack.
- **Shot takers.** Lateral strikers join the striker pool, lateral midfielders the midfield pool
  and full-backs the defense pool when an open-play shooter is drawn. The 70/20/10 split stays.
- **`PositionValue` alignment.** `PositionValue` becomes a weighted average of all seven skills
  per slot, matching what each skill is worth to the engine at that slot. Skill drift,
  `BestCorePosition`, the market SQL pre-selection and the client's best-position helper follow
  it.
- **Manager formation.** A Manager ranks the six formations by the projected expected goal
  difference of each best XI against the same squad's 4-4-2 XI, instead of by summed slot values.
  A Caretaker keeps the third or fourth of that same ranking.
- **New formation balance requirement**, checked by the harness.

Prototype results (identical squads, 3,000 matches per pairing):

| Formation | Points per match now | After |
|---|---|---|
| 4-4-2 | 1.39 | 1.39 |
| 5-3-2 | 1.19 | 1.36 |
| 3-4-3 | 1.32 | 1.38 |
| 4-3-3 | 1.73 | 1.36 |
| 4-5-1 | 1.39 | 1.41 |
| 3-5-2 | 1.39 | 1.43 |

With random squads (skills U[30,60]), every formation scores within ±0.06 xG of 4-4-2 on average,
and the Monte Carlo result against 4-4-2 stays between 1.33 and 1.39 points per match (the mirror
is 1.36). Goals per match barely move: 3.19 to 3.15 with every skill at 50, and 2.83 to 2.90
with random squads.

**BREAKING (balance):** saved lineups keep working, but every team's ratings change. Most of the
Director's edge came from the 4-3-3 exploit. Against its Caretaker it drops from 1.695 vs 1.079 to
1.438 vs 1.317 points per match; a Manager drops from 1.499 vs 1.277 to 1.417 vs 1.329.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-simulation`: "Team qualification recomputed every minute" gains line cover and new
  weights for wide players and for Pass and Dribbling. "Chance resolution" adds wide players to
  the shooter pools. A new "Formation balance" requirement is added.
- `cpu-team-management`: "Lineup and formation" ranks formations by projected expected goals for a
  Manager and a Caretaker.
- `player-development`: "Match-triggered skill drift" now follows the new per-slot skill weights.

## Impact

- Backend engine: `engine/initial_simulation.go` (`calculateQua`, `blendLine`,
  `lateralAttackValue`, `newSideDuelists`/`playerContributions`); `engine/scorer_simulation.go`
  (`getPlayerForChance`); `engine/expectedgoals.go` (`shooterMix`); `engine/progression.go`
  (`driftSkillsForPosition`); `engine/aiteams.go` (`refreshAiTeamLineup`).
- Backend common: a new `common/slotweights.go`; `common/lineup.go` (`PositionValue`,
  `SelectBestFormationAndLineup`); `common/fmteam.go`; `common/aimarket.go` (`positionSkillSQL`);
  `common/playerdetails.go` (`BestCorePosition`).
- Frontend: `src/utils/playerRating.ts` (best-position helper). The zone duel and opponent preview
  read backend values unchanged.
- No schema change, no migration, no new notifications or i18n keys.
- Calibrations to re-run: `TACTICS_SIM`, `AI_LEVEL_SIM`, `HOME_ADV_SIM`.
- Depends on `improve-cpu-tactics` being archived first, because both modify "Lineup and
  formation" in `cpu-team-management`.

Out of scope:

- Mentality. It is roughly zero-sum, moving goal volume more than results, and stays as is.
- Player traits (Header, Quick, Creative, Technical), which are still unused by the match engine.
- Goalkeeper weight: +10 GoalKeeper is worth about three times +10 Defense on one centre-back.
- Compensating the AI levels for the Director's lost edge.

Open questions for the owner:

- Should the Director get its edge back some other way, for example through mentality or
  substitutions, or is a 0.12 points-per-match gap over a Caretaker enough?
- Should the goalkeeper's weight be reviewed in a follow-up change?
