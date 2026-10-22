# Spec Delta

## ADDED Requirements

### Requirement: Set-piece chances
Each attacking chance SHALL first be classified: a penalty with probability 2.2%, a direct free kick with probability 6.5%, and otherwise an open-play chance. The set piece goes to the side that won the chance, and it MUST NOT add to or remove from the number of chances a match produces.

#### Scenario: Chance classification
- **WHEN** a chance is generated
- **THEN** it is a penalty if a uniform roll r < 0.022, a direct free kick if 0.022 ≤ r < 0.087, and open play otherwise

#### Scenario: Expected volume in an even match
- **WHEN** two teams with equal midfields play (about 12.6 chances per match at 14% per minute)
- **THEN** the match averages about 0.28 penalties and 0.82 direct free-kick attempts
- **AND** these match the real-world 0.24–0.41 penalties and ~0.85 direct free-kick shots per match

#### Scenario: Dominant side wins more penalties
- **WHEN** one side gets 68% of the chances
- **THEN** it also gets on average 68% of the penalties and direct free kicks, because they are a fixed share of its chances

#### Scenario: Total goals unchanged
- **WHEN** a large sample of even matches is simulated before and after this change
- **THEN** the average goals per match differ by no more than 0.05
- **AND** that holds because each set piece replaces an open-play chance worth ~0.24 goals with a penalty worth 0.78 or a free kick worth 0.06

### Requirement: Penalty resolution
A penalty SHALL be taken by the attacking side's on-pitch outfield player with the highest fatigue-adjusted Shot, ties going to the lowest player id. It is scored with probability `0.78 + 0.30 × (taker Shot − keeper GoalKeeper) / 99`, bounded to 60%–92%. It uses neither the open-play clear-cut roll nor the flank logic.

#### Scenario: Even taker and keeper
- **WHEN** the taker's Shot equals the opposing keeper's GoalKeeper
- **THEN** the penalty is scored with probability 78%

#### Scenario: Elite taker against a weak keeper
- **WHEN** the taker has Shot 80 and the keeper has GoalKeeper 30
- **THEN** the probability is 0.78 + 0.30 × 50/99 ≈ 0.93, bounded to 92%

#### Scenario: Weak taker against an elite keeper
- **WHEN** the taker has Shot 25 and the keeper has GoalKeeper 85
- **THEN** the probability is 0.78 − 0.30 × 60/99 ≈ 0.598, bounded to 60%

#### Scenario: Best finisher already sent off
- **WHEN** the side's highest-Shot player has been sent off, injured off or substituted
- **THEN** the best remaining on-pitch outfield player takes the penalty

#### Scenario: No outfield player left
- **WHEN** the attacking side has no outfield player on the pitch
- **THEN** the penalty is discarded, the same as an open-play chance

### Requirement: Direct free-kick resolution
A direct free kick SHALL be taken by the attacking side's on-pitch outfield player with the highest `0.6 × Shot + 0.4 × Pass` (fatigue-adjusted), ties going to the lowest player id. It is scored with probability `0.06 + 0.10 × (taker value − keeper GoalKeeper) / 99`, bounded to 2%–15%.

#### Scenario: Even taker and keeper
- **WHEN** the taker's value equals the keeper's GoalKeeper
- **THEN** the free kick is scored with probability 6%, matching the real-world ~5–6% conversion

#### Scenario: Specialist taker
- **WHEN** the taker has Shot 75 and Pass 70 (value 73) and the keeper has GoalKeeper 40
- **THEN** the probability is 0.06 + 0.10 × 33/99 ≈ 0.093

#### Scenario: Bounds
- **WHEN** the taker value exceeds the keeper's GoalKeeper by 99, or trails it by 99
- **THEN** the probability is 15% and 2% respectively

### Requirement: Set-piece event recording
A set-piece outcome SHALL be recorded as the usual goal or missed-chance event, carrying a set-piece kind (0 open play, 1 penalty, 2 direct free kick) and naming the defending goalkeeper as the other team's player. Every goal count (score, top scorers, awards, player stats, match ratings) MUST include set-piece goals.

#### Scenario: Penalty scored
- **WHEN** a home penalty is converted
- **THEN** a goal event with set-piece kind 1 is recorded for the taker, the home score goes up by one, and the away goalkeeper is listed as the other team's player

#### Scenario: Free kick missed
- **WHEN** a direct free kick is not converted
- **THEN** a missed-chance event with set-piece kind 2 is recorded and the score is unchanged

#### Scenario: Historic events
- **WHEN** an event stored before this change is read
- **THEN** its set-piece kind is 0 (open play)

#### Scenario: Match report shows the kind
- **WHEN** a match report or live replay lists a set-piece goal
- **THEN** it is marked as a penalty or a free kick next to the scorer's name in the viewer's language

## MODIFIED Requirements

### Requirement: Chance resolution
An open-play chance SHALL be central 50%, left 25% or right 25%, taken by a striker 70%, midfielder 20% or defender 10% (falling back to any remaining line), and converted with a logistic probability bounded to 5%–65% and centered on 28% for an even matchup. Set-piece chances are resolved separately (see "Set-piece chances").

#### Scenario: Flank chance
- **WHEN** a chance goes down the left
- **THEN** the attack value is the left attack plus 0.2 × (own left midfield − opponent's right midfield)
- **AND** it is compared with the opponent's right defense, because the two teams attack in opposite directions

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
