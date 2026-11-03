# socios Specification

## Purpose

Club members (socios) are the paying core of a home crowd: demand comes from members, not seats.
This covers how many people come to a home match, how socios grow or shrink each season, and the
revenue-maximising ticket price.

## Requirements

### Requirement: Socios count
Every team SHALL have a socios count, starting at 500, shown on the owner's finance page next to Fanbase.

#### Scenario: New team
- **WHEN** a team is created
- **THEN** it has 500 socios

### Requirement: Attendance model
A home match's crowd SHALL be `min(capacity, socios × price × form × rival × weather × noise)`, never negative. There is no minimum occupancy.

#### Scenario: Default price, neutral conditions
- **WHEN** a team with 500 socios charges 10 in cloudy weather with neutral form against a Fanbase-50 visitor
- **THEN** about 500 attend (± 10% noise)

#### Scenario: Capacity cap
- **WHEN** demand exceeds the stadium's capacity
- **THEN** attendance equals capacity

### Requirement: Attendance factors
The factors SHALL be: price `1 − (price − 10) / 30` clamped to [0, 1.3]; form `0.8 + 0.4 × points in the last 5 league matches / 15` (1.0 without history); rival `0.85 + 0.30 × visitor Fanbase / 99`, × (1 + 0.1 × cup round), × 1.10 for a Rival match or × 1.20 for a Clásico (see rivalries); weather sunny 1.05, cloudy 1.0, windy 0.92, rain 0.82; noise ±10%.

#### Scenario: Price 39
- **WHEN** a team charges 39, the highest price allowed
- **THEN** the price factor is 1/30 (about 3% of the crowd at the default price)

#### Scenario: Price 40
- **WHEN** the price factor is evaluated at 40 or more (no team can be charging that)
- **THEN** it is 0 and nobody would attend

#### Scenario: Five straight wins
- **WHEN** the home side won its last five league matches
- **THEN** the form factor is 1.2

#### Scenario: Cup round 2
- **WHEN** a cup tie in round 2 is played
- **THEN** the rival factor is multiplied by 1.2

#### Scenario: Clásico league match
- **WHEN** a Fanbase-50 visitor is the home side's Clásico rival in a league match
- **THEN** the rival factor is (0.85 + 0.30 × 50/99) × 1.20 ≈ 1.20

#### Scenario: Friendly between rivals
- **WHEN** two Clásico rivals play a friendly
- **THEN** no rivalry multiplier is applied

### Requirement: Weather drawn once
The weather SHALL be drawn once per match (league match, cup tie or friendly), before the simulation, and used for attendance and the match report. The attendance is drawn right after it, also before the simulation, so the same crowd sets the home advantage (see match-simulation) and the gate revenue.

#### Scenario: Rainy match
- **WHEN** a match's weather is rain
- **THEN** the report shows rain and attendance used ×0.82

#### Scenario: Rainy cup tie
- **WHEN** a cup tie's weather is drawn as rain
- **THEN** its attendance uses ×0.82 and its stored report weather is rain

#### Scenario: Sunny friendly
- **WHEN** a friendly's weather is drawn as sunny
- **THEN** its attendance uses ×1.05 and its stored report weather is sunny

#### Scenario: Form of the match itself
- **WHEN** the attendance of a league match is drawn
- **THEN** its form factor uses the home side's last 5 league matches before this one

### Requirement: Optimal ticket price
The optimal price SHALL be the integer price from 1 to 39 that maximises expected `price × min(capacity, demand)` against an average visitor in average weather, ties going to the lower price.

#### Scenario: Demand below capacity
- **WHEN** a club's socios cannot fill its ground
- **THEN** the optimal price is moderate (around 20)

#### Scenario: Demand above capacity
- **WHEN** a club's socios exceed its ground
- **THEN** the optimal price is the one that just fills it

#### Scenario: Shown to the owner
- **WHEN** an owner opens the finance page
- **THEN** the recommended ticket price shown is this optimal price for their team

### Requirement: Season-end socios drift
At each season rollover a team's socios SHALL close 30% of the gap to `2.2 × Fanbase^1.5 × position factor`, the position factor running from 1.10 for the champion to 0.90 for last; the result is rounded and at least 1.

#### Scenario: Big tier-1 club
- **WHEN** a Fanbase-80 champion has 1,000 socios
- **THEN** its target is about 1,730 and it ends the rollover with about 1,219

### Requirement: Socios notification
The owner SHALL be notified (`NOTIFICATION_FANBASE`, keys `sociosGrew`/`sociosShrank`) when socios change by at least 5% at season end. Unowned teams get no notification.

#### Scenario: Small change
- **WHEN** socios move from 500 to 510
- **THEN** no notification is sent

## Code map

- Attendance and price: `backend/engine/stadium.go` (`CalculateAttendance`, `priceFactor`,
  `formFactor`, `rivalFactor`, `weatherFactor`, `optimalTicketPrice`, `RecommendedTicketPrice`). The weather and
  the crowd are drawn once per match, before the simulation, by `drawMatchdayConditions` /
  `drawMatchdayCrowd` (`engine/stadium.go`), called from `simulateMatch`, `simulateCupTie` and
  `simulateFriendly`; the returned `matchdayCrowd` sets the home advantage, is credited by
  `creditTicketRevenue` (through `UpdateTeamFinancesAfterMatch`) / `creditFriendlyTicketRevenue`
  after the save commits, and feeds the stored report flavor.
- Drift: `engine/socios.go` (`sociosTarget`, `driftSocios`, `updateSociosAtSeasonEnd`), called
  from `awardSeasonTicketsAndMerchandising`; storage `common/socios.go`.
- Initial backfill for existing teams: `resources/sql/init80_socios.sql`.
- Rivalry multiplier: `rivalryAttendanceFactor` (`RivalryRivalAttendanceFactor`,
  `RivalryClasicoAttendanceFactor`) inside `rivalFactor`, from `matchdayContext.rivalryLevel`, which
  `simulateMatch` / `simulateCupTie` read at kickoff with `common.KickoffRivalryLevel` (friendlies
  pass none). Tests: `engine/rivalries_test.go` (`TestRivalryEffects`).
