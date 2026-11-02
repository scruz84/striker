# fanbase Specification

## Purpose

The per-team Fanbase index (0–99): how big and loyal a club's support is, how it drifts slowly
with results and division, and what it feeds into (socios, visiting crowds, commercial income).

## Requirements

### Requirement: Fanbase index
Every team SHALL have a Fanbase from 1 to 99, starting at 50. It is public on any team's page and shown on the owner's finance page.

#### Scenario: New team
- **WHEN** a team is created
- **THEN** its Fanbase is 50

### Requirement: Result drift
After each league match or cup tie (never a friendly) a team SHALL have a 15% chance of a result drift: +1 for a win, −1 for a loss, nothing for a draw.

#### Scenario: Win with a successful roll
- **WHEN** a team wins and its 15% roll succeeds
- **THEN** its Fanbase rises by 1

### Requirement: Tier pull
After each league match or cup tie a team SHALL independently have a 5% chance to move its Fanbase one point toward its division's target: 80 / 50 / 30 for tiers 1 / 2 / 3 (deeper tiers use 30).

#### Scenario: Promoted club
- **WHEN** a team with Fanbase 50 now plays in tier 1 and its 5% roll succeeds
- **THEN** its Fanbase becomes 51

### Requirement: Milestone notification
When Fanbase changes, the owner SHALL be notified (`NOTIFICATION_FANBASE`) only if it crossed into a different ten-point band.

#### Scenario: 49 to 50
- **WHEN** Fanbase moves from 49 to 50
- **THEN** the owner is notified

#### Scenario: 51 to 52
- **WHEN** Fanbase moves from 51 to 52
- **THEN** no notification is sent

### Requirement: Commercial factor
Sponsorship, season tickets and merchandising SHALL be multiplied by `0.8 + 0.4 × Fanbase / 99`.

#### Scenario: Biggest club
- **WHEN** a team has Fanbase 99
- **THEN** its sponsorship, season tickets and merchandising are ×1.2

### Requirement: Crowd influence
Fanbase SHALL affect attendance only through socios targets and as the visiting side's rival factor (see socios). It MUST NOT affect the match simulation.

#### Scenario: Big visitor
- **WHEN** a Fanbase-99 team visits
- **THEN** the home crowd is ×1.15 compared with ×0.85 for a Fanbase-0 visitor

## Code map

- `backend/engine/fanbase.go` (`applyFanbaseDrift`, `driftFanbaseForResult`,
  `driftFanbaseTierPull`, `FanbaseTierTarget`, `notifyFanbaseMilestone`,
  `fanbaseCommercialFactor`); storage `common/fanbase.go`.
- Exposed via `common.TeamDetails.Fanbase` and `FinancesResponse.Fanbase` (`FanbaseCard`).
- Reaches the simulation only through the attendance: the visiting side's Fanbase feeds
  `rivalFactor` (`engine/stadium.go`), and the drawn crowd sets `homeAdvantageMultiplier`
  (`engine/initial_simulation.go`).
