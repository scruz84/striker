# player-development Specification

## Purpose

How players change over time: match-by-match skill drift toward a hidden personal ceiling, the
Stamina and Form attributes, aging on its own birthday cycle, and the season-end retirement,
contract and academy steps (plus the free-agent equivalent).

## Requirements

### Requirement: Match-triggered skill drift
After every league match, cup tie and friendly, each player who featured SHALL get an independent chance for each skill used by the position he played most minutes at in that match. The chance per skill is minutes played in that match ÷ 900.

#### Scenario: Full match as a striker
- **WHEN** a striker plays all 90 minutes
- **THEN** each of Shot, Pass and Dribbling has a 10% chance to drift and his Defense never moves

#### Scenario: Friendly
- **WHEN** a player features in a friendly
- **THEN** his skills can drift exactly as after a competitive match

#### Scenario: Position skill sets
- **WHEN** the most-played position is decided
- **THEN** goalkeeper drifts GoalKeeper; central defender Defense; full-back Lateral and Defense; central midfielder Midfield and Pass; wide midfielder Lateral, Midfield and Pass; striker Shot, Pass and Dribbling; wide striker Lateral, Shot, Pass and Dribbling

### Requirement: Drift amount
A drifting skill SHALL change by the age trend (+1 under 24, 0 from 24 to 29, −1 from 30) plus the trainer bonus plus a random −1/0/+1, clamped to [1, the player's own ceiling]. The trainer bonus MUST only apply while the age trend is not negative.

#### Scenario: Elite trainer, young player
- **WHEN** a 20-year-old's skill drifts at a team whose trainer has skill 80 or more
- **THEN** the delta is +1 (age) +2 (trainer) ±1

#### Scenario: Decent trainer
- **WHEN** the trainer's skill is between 30 and 79
- **THEN** the bonus is +1

#### Scenario: Veteran with a coach
- **WHEN** a 31-year-old's skill drifts at a team with an elite trainer
- **THEN** no trainer bonus applies and the delta is −1 ±1

### Requirement: Hidden skill ceilings
Every created player SHALL get a per-skill ceiling rolled once: 70% get +5–15 over the starting value, 25% +15–30, 5% +30–45, capped at 99. Ceilings MUST never be exposed in any API response.

#### Scenario: Skill at its ceiling
- **WHEN** a skill already equals its ceiling and drifts upward
- **THEN** it stays at the ceiling

#### Scenario: Player without stored ceilings
- **WHEN** a player has no ceiling row
- **THEN** 99 is used for every skill

### Requirement: Progression log and notification
When at least one skill drifted after a match, the change SHALL be persisted and logged in `player_progression_log`, and the team owner MUST receive one `NOTIFICATION_PROGRESSION` linking to `/players/evolutions?logIds=…` with exactly that match's log rows. Nothing is written when nothing drifted.

#### Scenario: Two players evolved
- **WHEN** two of a team's players gained a skill point after a match
- **THEN** the owner gets one notification for 2 players linking to those two log rows only

### Requirement: Stamina drift
After a match, each player who featured SHALL get the same minutes-scaled chance to drift Stamina by an age trend (+1 under 24, 0 to 29, −1 to 33, −2 from 34) ±1, clamped to [1, 99]. There is no trainer bonus and players who did not feature are untouched.

#### Scenario: 35-year-old plays a full match
- **WHEN** his Stamina drifts
- **THEN** it changes by −2 ±1

### Requirement: Form drift
Every squad player's Form SHALL update after each match: a player who featured gets +2 plus round((match rating − 6.0) × 2); a squad player who did not feature loses 1. Form is clamped to [1, 99] and missing Form reads as 50.

#### Scenario: Poor performance
- **WHEN** a player features with a match rating of 4.5
- **THEN** his Form changes by +2 − 3 = −1

#### Scenario: Benched
- **WHEN** a squad player does not feature
- **THEN** his Form drops by 1

### Requirement: Stamina and Form stay out of progression reporting
Stamina and Form changes MUST NOT create `player_progression_log` rows or progression notifications.

#### Scenario: Only Form changed
- **WHEN** a match changes only Form and Stamina values
- **THEN** the owner receives no progression notification

### Requirement: New player attributes
New players SHALL start with Form 50 and a Stamina drawn uniformly from 50–90, independent of their skills.

#### Scenario: Academy prospect created
- **WHEN** an academy player is generated
- **THEN** his Form is 50 and his Stamina is between 50 and 90

### Requirement: Birthday cycle
A daily job (01:00 UTC) SHALL age by one year every player (rostered, free agent or academy) whose `birth_day` anchor is at least 20 days old, rolling the anchor forward 20 days per year applied, and catching up all missed cycles.

#### Scenario: Job missed for 45 days
- **WHEN** a player's anchor is 45 days old
- **THEN** he ages 2 years and his anchor moves forward 40 days

### Requirement: Season-end player rollover
When a country's season rolls over, each of its rostered players SHALL first be checked for retirement (forced at 40; from 33 the chance is 15% per year past 32), then have his contract shortened by one season, releasing him to free agency at zero. Teams left short are topped up from the academy.

#### Scenario: 35-year-old
- **WHEN** a 35-year-old is checked
- **THEN** he retires with 45% probability and, if he does, no contract step applies

#### Scenario: Contract runs out
- **WHEN** a player's remaining contract reaches zero
- **THEN** he becomes a free agent

#### Scenario: Other countries
- **WHEN** one country rolls over
- **THEN** players in countries still mid-season are untouched

### Requirement: Free-agent progression
A daily job (02:00 UTC) SHALL check every free agent: retirement by age as above, otherwise advance his unsigned counter; at 2 or more he retires regardless of age. A retiring free agent MUST become an unemployed trainer candidate. Free agents never drift skills.

#### Scenario: Second unsigned run
- **WHEN** a free agent's unsigned counter reaches 2
- **THEN** he is converted into a trainer on the open market

## Code map

- Drift: `backend/engine/progression.go` (`updatePlayerSkillsAfterMatch`, `driftSkillsForPosition`,
  `driftSkill`, `trainerBonus`, `driftStamina`, `driftFormPlayed`, `driftFormBenched`,
  `MatchSkillGrowthMinutesReference`), triggered by `updateSkillsAfterPlayedMatch`
  (`engine/simulation.go`).
- Ceilings: `common.rollSkillCap`, `common.GetPlayerSkillCaps` (`common/database.go`).
- Ratings used by Form: `common.ComputeMatchRatings` (`common/matchratings.go`).
- Aging: `engine/birthdays.go` (`ProcessBirthdays`, `BirthdayCycleDays`).
- Rollover: `ProgressPlayers`/`ProgressFreeAgents`, `ShouldRetire`, `common.AdvancePlayerContract`,
  `common.AdvanceFreeAgencyClock`, `common.RetireFreeAgentAsTrainer`, `common.CreateAcademyPlayers`.
