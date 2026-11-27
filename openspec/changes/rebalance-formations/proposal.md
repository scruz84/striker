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
- **Crowded back line.** Each flank defense gains 0.225 × the central defense's per-player value
  for every half-player of central weight over two, so a back five also shuts the flanks and a back
  three leaves them more exposed.
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
- **Best-XI assignment.** A formation's best XI is the assignment of players to its eleven slots
  with the highest total slot value. Today slots are filled greedily, back line first, which puts
  a wide-midfield specialist at full-back because his Lateral fills that slot before the midfield
  is reached.
- **Manager formation.** A Manager ranks the six formations by the projected expected goal
  difference of each best XI against the same squad's 4-4-2 XI, instead of by summed slot values.
  A Caretaker keeps the third or fourth of that same ranking.
- **Recent signings are not for sale.** A CPU team refuses every offer for a player it signed
  within the 14-day hold that already keeps him off its transfer list. Without it, the new slot
  values let two Directors trade the same player back and forth.
- **New "Formation balance" and "Squad fit" requirements**, checked by the harness.

Figures below are from the implementation (`TACTICS_SIM`), after tuning; the prototype's earlier
figures are replaced.

### Balance with even squads

Identical squads, 3,000 matches per pairing:

| Formation | Points per match now | After |
|---|---|---|
| 4-4-2 | 1.39 | 1.39 |
| 5-3-2 | 1.19 | 1.38 |
| 3-4-3 | 1.32 | 1.43 |
| 4-3-3 | 1.73 | 1.39 |
| 4-5-1 | 1.39 | 1.36 |
| 3-5-2 | 1.39 | 1.42 |

With generalist random squads (skills U[30,60]), every formation's mean projected goal difference
against the same squad's 4-4-2 lies between −0.042 and 0.000. Goals per match barely move: 3.19 to
3.13 with every skill at 50.

### Realistic squads

Realistic squads are random squads plus 2–4 partial specialists of random types: a role skill
drawn from 60 to 80, and the other skills 15 lower. Each formation played the same squad in
4-4-2, 10,000 matches per formation (±0.013 points per match):

| Formation | Points per match now | After |
|---|---|---|
| 4-4-2 (mirror) | 1.37 | 1.34 |
| 5-3-2 | 1.18 | 1.36 |
| 3-4-3 | 1.33 | 1.38 |
| 4-3-3 | 1.78 | 1.37 |
| 4-5-1 | 1.40 | 1.37 |
| 3-5-2 | 1.42 | 1.38 |

In real matches no formation is stronger. Each one is also the best projected choice for a fair
share of realistic squads, so the CPU varies its shape by squad:

| | 4-4-2 | 5-3-2 | 3-4-3 | 4-3-3 | 4-5-1 | 3-5-2 |
|---|---|---|---|---|---|---|
| Before tuning (prototype) | 19% | 5% | 13% | 38% | 6% | 16% |
| After | 22% | 14% | 12% | 23% | 13% | 16% |

The optimal best-XI assignment and a stronger crowded back line (0.225 instead of 0.15) closed
the gap: 4-3-3 had been chosen most often only because it got the most out of specialists, and
from more kinds of them.

### Squad fit

Random squads were given one kind of specialist at skill 75: a "pure" specialist is 75 in his role
skills and 25 in the rest. The table shows how often each formation becomes the best projected
shape for squads with **that** profile, against the same squad's 4-4-2. It measures fit, not
overall strength: 4-3-3's 99% applies only to squads with two pure wingers.

| Formation | Squad profile | Best for, now | Best for, after |
|---|---|---|---|
| 5-3-2 | 3 pure centre-backs | 0% | 61% |
| 3-4-3 | 3 pure strikers | 0% | 69% |
| 4-3-3 | 2 pure wingers | 100% (as for every squad) | 99% |
| 4-5-1 | no striker above 30 | 0% | 89% |
| 3-5-2 | 2 pure full-backs | 0% | 65% |

Every formation's projected edge over 4-4-2 grows as its specialists get better (at 75 against at
40). For a 5-3-2 with three pure centre-backs it is +0.003 xG at 75 against −0.016 at 40. In full
matches that edge is small: about +0.03 points per match between good and poor centre-backs over
40,000 matches per case, with a good back five level with the same squad's 4-4-2. That is as large
as the noise of a normal calibration run, so by the owner's decision the requirement is checked on
the projection, and full-match figures are only reported.

**BREAKING (balance):** saved lineups keep working, but every team's ratings change. Most of the
Director's edge came from the 4-3-3 exploit. Against its Caretaker it goes from 1.695 vs 1.079 to
1.421 vs 1.339 points per match; a Manager goes from 1.499 vs 1.277 to 1.423 vs 1.329 (20,000
matches each). Because a Manager now also ranks formations by projected expected goals, a Director
no longer gains a measurable edge over it in the calibration, where both sides field the same
squad. The owner accepted that tie: the calibration only checks that a Director is not clearly
worse than a Manager.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-simulation`: "Team qualification recomputed every minute" gains line cover, the crowded
  back line, wing-back tuck-in and new weights for wide players and for Pass and Dribbling.
  "Chance resolution" adds wide players to the shooter pools. New "Formation balance" and "Squad
  fit" requirements are added.
- `cpu-team-management`: "Lineup and formation" ranks formations by projected expected goals for a
  Manager and a Caretaker, and picks each formation's best XI by optimal slot assignment.
  "Answering transfer offers" refuses offers for a player signed within the last 14 days.
- `player-development`: "Match-triggered skill drift" now follows the new per-slot skill weights.
- `match-ratings`: "Zone contribution per minute" uses the new value formulas, lets a wide
  midfielder feed his flank's midfield too, and compares each player with the rival line's weighted
  per-player value.

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
- Squad fit is small in full matches. The right specialists in the right formation are worth a
  few hundredths of a point per match, about what one player moves the engine today.

Open questions for the owner:

- Should a Director get an edge over a Manager back some other way, for example through
  mentality or substitutions? (Decided for now: a tie with a Manager is accepted.)
- Should the goalkeeper's weight and defense's diminishing returns be reviewed in a follow-up
  change?
