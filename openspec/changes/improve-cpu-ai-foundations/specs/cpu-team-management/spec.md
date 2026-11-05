# Spec Delta

## MODIFIED Requirements

### Requirement: CPU captain
At kickoff of every match a CPU team SHALL name as captain its oldest outfield starter who is not the outgoing player of one of its scheduled substitutions, ties going to the lowest player id, and leave both set-piece takers automatic. If every outfield starter is due off, the oldest outfield starter is captain.

#### Scenario: Veteran captain
- **WHEN** a CPU team's outfield starters are aged 24, 31 and 33 among others, with 33 the oldest, and none of them is scheduled off
- **THEN** the 33-year-old is its captain

#### Scenario: Veteran due off
- **WHEN** the 33-year-old is the outgoing player of one of the team's scheduled substitutions
- **THEN** the 31-year-old is its captain

#### Scenario: Takers
- **WHEN** a CPU team wins a penalty
- **THEN** its taker is chosen by the automatic rule

## ADDED Requirements

### Requirement: CPU sponsor deal
On each management pass a Manager or Director that has played at least 6 league matches this season and has not yet chosen a deal this season SHALL compute each deal's expected payout per match from its season win rate w and draw rate d, and switch to the best one when it is not its current deal. Ties keep the current deal. A Caretaker never changes its deal.

#### Scenario: Payout formulas
- **WHEN** the expected payouts are computed, in units of the tier's base sponsorship
- **THEN** Steady is 1.3, Balanced is 1 + 0.6 × w + 0.3 × d and Performance is 0.6 + 1.2 × w + 0.4 × d

#### Scenario: Strugglers take the guarantee
- **WHEN** a Manager has 1 win and 2 draws in 8 league matches (w = 0.125, d = 0.25)
- **THEN** Steady (1.3) beats Balanced (1.15) and Performance (0.85), and it switches to Steady, locked for the season

#### Scenario: Runaway leader
- **WHEN** a Director has 7 wins and 1 draw in 9 league matches (w ≈ 0.778, d ≈ 0.111)
- **THEN** Performance (≈ 1.578) beats Balanced (≈ 1.50) and it switches to Performance

#### Scenario: Too early
- **WHEN** a Director has played 5 league matches this season
- **THEN** it keeps its current deal

### Requirement: Manager trainer
A Manager with no trainer SHALL hire, on its management pass, the most skilled available trainer whose skill is below 80; it MUST NOT fire a trainer it has. A Director keeps its rule (hire the best, or swap for one at least 10 better). A Caretaker never hires.

#### Scenario: Elite coaches left alone
- **WHEN** a Manager with no trainer finds trainers of skill 88, 79 and 45 on the market
- **THEN** it hires the 79

#### Scenario: Only elite coaches
- **WHEN** every available trainer has skill 80 or more
- **THEN** the Manager hires nobody

#### Scenario: Better coach appears
- **WHEN** a Manager already has a trainer of skill 40 and one of skill 75 becomes available
- **THEN** it keeps its trainer

### Requirement: Director cup orders
At kickoff of a cup tie a Director SHALL add, after its other conditional orders, "from 75, if drawing" with mentality Attacking when its kickoff XI's strength is at least the opponent's, and Defensive otherwise. A draw goes to a penalty shootout won by either side with probability 50%, so a weaker side settles for it.

#### Scenario: Favourite level late
- **WHEN** a Director whose XI is worth 1,050 against 1,000 is drawing at the start of minute 75 of a cup tie
- **THEN** it switches to Attacking in minute 75

#### Scenario: Underdog level late
- **WHEN** a Director whose XI is worth 950 against 1,000 is drawing at the start of minute 80 of a cup tie
- **THEN** it switches to Defensive

#### Scenario: League match
- **WHEN** a Director plays a league match
- **THEN** it adds no drawing order

### Requirement: AI level calibration
An opt-in simulation (`AI_LEVEL_SIM=1`) SHALL play 4,000 matches of each level against a Caretaker on a neutral ground, both sides with the same squad, each side planning its XI, formation, substitutions, mentality and orders as at a league kickoff, and report each side's win, draw and loss rates and points per match. A Manager MUST score more points per match than the Caretaker it plays, and a Director more than a Manager does.

#### Scenario: Running it
- **WHEN** an operator runs `AI_LEVEL_SIM=1 go test ./engine -run TestAiLevelCalibration -v`
- **THEN** the log lists, for Manager vs Caretaker and Director vs Caretaker, both sides' win/draw/loss rates and points per match
