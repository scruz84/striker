# Spec Delta

## ADDED Requirements

### Requirement: Team mentality
Every lineup SHALL carry a mentality: Defensive (−1), Balanced (0, the default) or Attacking (+1). Every minute, after the zone blend and the home advantage, an Attacking side's three attack ratings MUST be multiplied by 1.08 and its three defense ratings by 0.92; a Defensive side's the other way round. Midfield ratings and a Balanced side are unchanged.

#### Scenario: Attacking side
- **WHEN** an Attacking side has a central attack of 200 and a central defense of 180 after the blend
- **THEN** they are 216 and 165.6 for that minute

#### Scenario: Two Balanced sides
- **WHEN** both sides are Balanced
- **THEN** the match is simulated exactly as without mentality

#### Scenario: Lineup saved before this change
- **WHEN** a lineup saved before this change is used
- **THEN** it plays Balanced

### Requirement: Mentality effect sizes
Between sides with identical squads, an Attacking side facing a Balanced one SHALL score 8%–20% more goals and concede 8%–20% more than a Balanced side does against a Balanced side; a Defensive side facing a Balanced one SHALL concede 8%–20% fewer goals and score 8%–20% fewer.

#### Scenario: Calibration sample
- **WHEN** 20,000 matches of an Attacking side against a Balanced side with identical squads are simulated
- **THEN** the Attacking side's goals for and goals against are both 8%–20% above the Balanced-vs-Balanced average

## MODIFIED Requirements

### Requirement: Chance generation
Each minute SHALL produce an attacking chance with probability 14% plus up to 10 points scaled by midfield dominance, plus 1 point for each Attacking side and minus 1 point for each Defensive side. The attacking side MUST be chosen by a midfield-weighted share bounded to 32%–68%.

#### Scenario: Even midfields
- **WHEN** both midfields are equal and both sides are Balanced
- **THEN** a chance occurs with 14% probability and each side gets it half of the time

#### Scenario: Total midfield dominance
- **WHEN** one side holds almost all the midfield strength and both sides are Balanced
- **THEN** the chance probability approaches 24% and that side gets at most 68% of chances

#### Scenario: Two Attacking sides
- **WHEN** both midfields are equal and both sides are Attacking
- **THEN** a chance occurs with 16% probability

#### Scenario: Attacking against Defensive
- **WHEN** one side is Attacking and the other Defensive
- **THEN** the mentality adjustments cancel and the probability is the same as for two Balanced sides

### Requirement: In-match fatigue
Each minute, every player's seven core skills SHALL be scaled by `1 - P × (minutes played / 90) × ((99 - Stamina) / 99)`, where P is 0.30 for a Balanced side, 0.33 for an Attacking side and 0.27 for a Defensive side at that minute, and every later stage MUST read these effective skills.

#### Scenario: Stamina 99
- **WHEN** a Stamina-99 player has played 90 minutes
- **THEN** his effective skills equal his raw skills

#### Scenario: Stamina 0 at full time
- **WHEN** a Stamina-0 player of a Balanced side has played 90 minutes
- **THEN** each of his core skills is reduced by 30%

#### Scenario: Not yet on the pitch
- **WHEN** a player has played 0 minutes this match
- **THEN** he is unaffected whatever his Stamina

#### Scenario: Attacking side at full time
- **WHEN** a Stamina-0 player of an Attacking side has played 90 minutes
- **THEN** each of his core skills is reduced by 33%
