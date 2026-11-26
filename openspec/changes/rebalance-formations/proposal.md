# Proposal

## Why

The tactics study harness (`TACTICS_SIM=1`, `engine/tactics_study_test.go`) shows that the
formation a team picks decides more than its players.

- **4-3-3 dominates.** With identical squads it averages 1.73 points per match against the other
  formations, which all sit between 1.19 and 1.39. A Director picks it for 400 of 400 random
  squads, and it is the best projected formation for 100% of squads whatever players they have.
  Its two lateral strikers count both on their flank and in the central attack, so its nine zones
  add up to 700 where every other shape adds up to 600.
- **5-3-2 is dominated.** It averages 1.19 points per match because its single central midfielder
  loses the midfield battle outright. Yet a Manager ranks it first for half of all random squads.
- **The squad does not decide the shape.** Three great centre-backs do not make a 5-3-2 worth
  playing, and three great strikers do not make a 3-4-3 worth playing.
- **The CPU's player valuation disagrees with the engine.** `PositionValue` is the "value at
  slot" that CPU lineups, substitutions, transfers and skill drift all use. It weights Pass at
  50% for a central midfielder and Pass plus Dribbling at 50% for a striker, while the engine
  barely uses either. It also rates a wide midfielder on Midfield and Pass, while the engine
  rates him on Lateral and Shot only.
- **Wide players never take an open-play shot.** The lone striker of a 4-3-3 or 4-5-1 takes 70% of
  his side's chances.

## What Changes

The zone model is changed so that a formation trades strength between lines rather than gaining it
outright, and so that a formation pays off when the squad has the players it needs:

- **Line cover.** Each central defender, midfielder and striker lends 10% of himself to each
  adjacent central line: a defender to the midfield, a midfielder to the defense and the attack,
  and a striker to the midfield. It is kept small because more cover pulls specialists down in the
  lines they do not play.
- **Crowded back line.** The central defense's spillover to the flanks scales with how many
  players it holds, so a back five also shuts the flanks and a back three leaves them more exposed.
- **Back-three wing-backs.** With a single DEFENSE slot, each full-back adds 15% of his Defense to
  the central defense.
- **Wide forwards.** A lateral striker counts 50% (was 100%) toward the central attack.
- **Wide midfielders.** A lateral midfielder counts 65% toward the central midfield and supplies
  half of his flank's midfield rating; the full-back supplies the other half.
- **Pass and Dribbling.** A midfielder's value is 70% Midfield and 30% Pass. An attacker's value is
  75% Shot and 25% Dribbling, which also enters the wide attack.
- **Shot takers.** Lateral strikers join the striker pool, lateral midfielders the midfield pool
  and full-backs the defense pool when an open-play shooter is drawn. The 70/20/10 split stays.
- **`PositionValue` alignment.** `PositionValue` becomes a weighted average of all seven skills
  per slot, matching what each skill is worth to the engine at that slot. Skill drift,
  `BestCorePosition`, the market SQL pre-selection and the client's best-position helper follow
  it.
- **Manager formation.** A Manager ranks the six formations by the projected expected goal
  difference of each best XI against the same squad's 4-4-2 XI, instead of by summed slot values.
  A Caretaker keeps the third or fourth of that same ranking.
- **New "Formation balance" and "Squad fit" requirements**, checked by the harness.

### Balance with even squads

Identical squads, 3,000 matches per pairing:

| Formation | Points per match now | After |
|---|---|---|
| 4-4-2 | 1.39 | 1.41 |
| 5-3-2 | 1.19 | 1.36 |
| 3-4-3 | 1.32 | 1.39 |
| 4-3-3 | 1.73 | 1.36 |
| 4-5-1 | 1.39 | 1.39 |
| 3-5-2 | 1.39 | 1.42 |

With random squads (skills U[30,60]), the Monte Carlo result of each formation against the same
squad in 4-4-2 stays between 1.34 and 1.40 points per match. Goals per match barely move: 3.19 to
3.16 with every skill at 50.

### Squad fit

Random squads were given specialists at skill 75: a "pure" specialist is 75 in his role skills and
25 in the rest. The table shows how often each formation becomes the best projected shape for the
squad, against the same squad's 4-4-2.

| Formation | Squad profile | Best for, now | Best for, after |
|---|---|---|---|
| 5-3-2 | 3 pure centre-backs | 0% | 26% |
| 3-4-3 | 3 pure strikers | 0% | 75% |
| 4-3-3 | 2 pure wingers | 100% (as for every squad) | 90% |
| 4-5-1 | no striker above 30 | 0% | 18% |
| 3-5-2 | 2 pure full-backs | 0% | 50% |

In full matches (Monte Carlo), each line compares the formation with the same squad in 4-4-2, both
against the unmodified squad's 4-4-2:

| Squad | Its formation | Same squad in 4-4-2 |
|---|---|---|
| 3 good centre-backs (75), in a 5-3-2 | 1.47 | 1.33 |
| The same 3 centre-backs at 40, in a 5-3-2 | 1.27 | 1.28 |
| 3 good strikers, in a 3-4-3 | 1.55 | 1.52 |
| 2 good full-backs, in a 3-5-2 | 1.52 | 1.48 |

**BREAKING (balance):** saved lineups keep working, but every team's ratings change. Most of the
Director's edge came from the 4-3-3 exploit. Against its Caretaker it goes from 1.695 vs 1.079 to
1.494 vs 1.270 points per match; a Manager goes from 1.499 vs 1.277 to 1.446 vs 1.313.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-simulation`: "Team qualification recomputed every minute" gains line cover, the crowded
  back line, wing-back tuck-in and new weights for wide players and for Pass and Dribbling.
  "Chance resolution" adds wide players to the shooter pools. New "Formation balance" and "Squad
  fit" requirements are added.
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

Known limits of the model, which this change measures but does not remove:

- Raising one skill by the same amount for every player does not change which formation suits a
  squad. The engine compares lines as ratios, so a uniform boost scales every formation alike.
  What decides the shape is which players the squad has, for example a third good centre-back.
- Defense has diminishing returns. A centre-back is only compared with the opponent's attack on
  central chances, and on top of the goalkeeper's skill, so a back line that is already strong
  gains little from getting stronger.
- 4-5-1 is the weakest fit: it is the best shape for only 18% of squads that lack strikers.

Open questions for the owner:

- Is the Director's remaining edge over a Caretaker (≈0.22 points per match with identical
  squads) enough, or should it get more back some other way, for example through mentality or
  substitutions?
- Should the goalkeeper's weight and defense's diminishing returns be reviewed in a follow-up
  change?
