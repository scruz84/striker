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
- **THEN** each central defender also adds 0.2 of his midfield value to the central midfield, each central midfielder 0.2 of his Defense to the central defense and 0.2 of his attacking value to the central attack, and each central striker 0.2 of his midfield value to the central midfield

#### Scenario: Wide forwards and wide midfielders
- **WHEN** a central rating is computed
- **THEN** each lateral striker adds 0.35 of his attacking value to the central attack and each lateral midfielder 0.5 of his midfield value to the central midfield

#### Scenario: Identical 4-4-2 and 4-3-3
- **WHEN** a 4-4-2 and a 4-3-3 are rated with every skill at 50
- **THEN** the 4-4-2's central defense, midfield and attack are 120, 190 and 120
- **AND** the 4-3-3's are 130, 180 and 115

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
No formation SHALL be better or worse than another by construction. With identical squads, every formation's average points per match against all six formations MUST stay within 0.06 of the six-formation mean, and with random squads its mean projected goal difference against the same squad's other formations MUST stay within ±0.06.

#### Scenario: Identical squads round robin
- **WHEN** squads with every skill at 50 play 3,000 neutral, Balanced matches per formation pairing
- **THEN** each formation's average points per match lies within 0.06 of the mean of all six

#### Scenario: Random squads
- **WHEN** 400 squads with skills drawn from 30 to 60 are each projected in every formation against themselves in every formation
- **THEN** each formation's mean expected goal difference lies between −0.06 and +0.06

#### Scenario: Goal volume
- **WHEN** two identical 4-4-2 squads with every skill at 50 play
- **THEN** they score 3.0 to 3.4 goals per match between them (3.19 before the rebalance, 3.15 in the prototype)
