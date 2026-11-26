# Spec Delta

## MODIFIED Requirements

### Requirement: Team qualification recomputed every minute
The simulator SHALL recompute each team's attack, midfield and defense ratings (center, left and right) every minute from whoever is on the pitch, so red cards, injuries and substitutions take effect immediately.

#### Scenario: Core ratings
- **WHEN** a central rating is computed
- **THEN** it is the weighted sum of each contributing player's value for that line: central defense uses Defense, central midfield 0.7 × Midfield + 0.3 × Pass, central attack 0.75 × Shot + 0.25 × Dribbling
- **AND** every player in the line itself weighs 1

#### Scenario: Line cover
- **WHEN** a central rating is computed
- **THEN** each central defender also adds 0.1 of his midfield value to the central midfield, each central midfielder 0.1 of his Defense to the central defense and 0.1 of his attacking value to the central attack, and each central striker 0.1 of his midfield value to the central midfield

#### Scenario: Wide forwards and wide midfielders
- **WHEN** a central rating is computed
- **THEN** each lateral striker adds 0.5 of his attacking value to the central attack and each lateral midfielder 0.65 of his midfield value to the central midfield

#### Scenario: Back-three wing-backs
- **WHEN** a side fields a single DEFENSE slot
- **THEN** each of its full-backs also adds 0.15 of his Defense to the central defense

#### Scenario: Identical 4-4-2 and 4-3-3
- **WHEN** a 4-4-2 and a 4-3-3 are rated with every skill at 50
- **THEN** the 4-4-2's central defense, midfield and attack are 110, 185 and 110
- **AND** the 4-3-3's are 115, 165 and 115

#### Scenario: Flank ratings
- **WHEN** a flank rating is computed
- **THEN** the flank defense is the full-back's 0.5 × Lateral + 0.5 × Defense, and the flank attack the wide player's 0.5 × Lateral + 0.5 × attacking value (the lateral striker if the formation has one, otherwise the lateral midfielder)
- **AND** in a back three (one DEFENSE slot) the flank defenders weigh Lateral 35% and Defense 65%

#### Scenario: Flank midfield
- **WHEN** a flank's midfield rating is computed and the flank has a lateral midfielder
- **THEN** it is half the full-back's flank-midfield value (0.5 × Lateral + 0.5 × Midfield; 0.65 and 0.35 in a back three) plus half the lateral midfielder's 0.5 × Lateral + 0.5 × midfield value
- **AND** without a lateral midfielder it is the full-back's value alone

#### Scenario: Zone spillover
- **WHEN** a line's zones are blended
- **THEN** each zone takes 15% of its neighbors' per-player average strength, where the center's average is its weighted total over the sum of its contributors' weights
- **AND** the center's blended value is scaled back up by that sum of weights

#### Scenario: Crowded back line
- **WHEN** the defense line is blended and its central weight sum is W
- **THEN** each flank defense also gains 0.15 × the central per-player average × (W ÷ 2 − 1), which is negative below a weight of 2
- **AND** with every skill at 50 the flank defense is 54.1 in a 5-3-2, 50.8 in a 4-4-2 and 48.1 in a 3-4-3

### Requirement: Chance resolution
An open-play chance SHALL be central 50%, left 25% or right 25%, taken by a striker 70%, midfielder 20% or defender 10% (falling back to any remaining line), and converted with a logistic probability bounded to 5%–65% and centered on 28% for an even matchup. Set-piece chances are resolved separately (see "Set-piece chances").

#### Scenario: Flank chance
- **WHEN** a chance goes down the left
- **THEN** the attack value is the left attack plus 0.2 × (own left midfield − opponent's right midfield)
- **AND** it is compared with the opponent's right defense, because the two teams attack in opposite directions

#### Scenario: Wide players shoot
- **WHEN** the open-play shooter is drawn
- **THEN** the striker share is spread evenly over central and lateral strikers, the midfielder share over central and lateral midfielders, and the defender share over central defenders and full-backs

#### Scenario: Lone striker in a 4-3-3
- **WHEN** a 4-3-3 side takes an open-play chance
- **THEN** its central striker and each lateral striker take it with 23.3% probability each

#### Scenario: Finisher and goalkeeper
- **WHEN** a chance is resolved
- **THEN** the attack value includes the shooter's own scoring value and the defense value includes the goalkeeper's skill

#### Scenario: Clear-cut versus half-chance
- **WHEN** a chance is resolved
- **THEN** it is clear-cut (attack ×1.6) with probability 20% ± up to 15 points by attack-vs-defense edge, bounded to 5%–50%, and otherwise a half-chance (attack ×0.7)

#### Scenario: No outfield player left
- **WHEN** the attacking team has no outfield players left
- **THEN** the chance is discarded

#### Scenario: Set piece instead of open play
- **WHEN** a chance has been classified as a penalty or a direct free kick
- **THEN** none of the open-play rules above (flank, shooter weights, clear-cut roll, logistic curve) apply to it

## ADDED Requirements

### Requirement: Formation balance
No formation SHALL be better or worse than another by construction. With identical squads, every formation's average points per match against all six formations MUST stay within 0.06 of the six-formation mean, and with random squads of generalists its mean projected goal difference against the same squad's 4-4-2 MUST stay within ±0.05.

#### Scenario: Identical squads round robin
- **WHEN** squads with every skill at 50 play 3,000 neutral, Balanced matches per formation pairing
- **THEN** each formation's average points per match lies within 0.06 of the mean of all six

#### Scenario: Random squads
- **WHEN** 400 squads with every skill drawn from 30 to 60 are each projected in every formation against their own best 4-4-2 XI
- **THEN** each formation's mean expected goal difference lies between −0.05 and +0.05

#### Scenario: Goal volume
- **WHEN** two identical 4-4-2 squads with every skill at 50 play
- **THEN** they score 3.0 to 3.4 goals per match between them (3.19 before the rebalance)

### Requirement: Squad fit
A formation SHALL pay off when the squad has the players it needs. For each formation other than 4-4-2, a squad with that formation's specialists MUST find it the best projected shape for at least 15% of random squads, and its edge over the same squad's 4-4-2 MUST grow as those specialists get better.

#### Scenario: Specialist profiles
- **WHEN** random squads get pure specialists at 75 in their role skills and 25 in the rest (5-3-2: 3 centre-backs on Defense; 3-4-3: 3 strikers on Shot and Dribbling; 4-3-3: 2 wingers on Lateral, Shot and Dribbling; 3-5-2: 2 full-backs on Lateral and Defense), or every Shot and Dribbling is capped at 30 (4-5-1)
- **THEN** that formation is the best projected one against the same squad's 4-4-2 for at least 15% of 100 squads

#### Scenario: Better specialists, bigger edge
- **WHEN** the same profiles are built with specialists at 75 and again at 40, for 5-3-2, 3-4-3, 4-3-3 and 3-5-2
- **THEN** the formation's mean projected goal difference minus the same squad's 4-4-2 is higher at 75 than at 40

#### Scenario: Five at the back with good and poor centre-backs
- **WHEN** squads with three pure centre-backs play 150 × 12 neutral matches against their own unmodified squad in 4-4-2
- **THEN** with the centre-backs at 75 a 5-3-2 takes at least 0.08 points per match more than the same squad in 4-4-2
- **AND** with them at 40 it takes no more than 0.03 more
