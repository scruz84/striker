# Spec Delta

## MODIFIED Requirements

### Requirement: Season rollover once per country
A country's rollover (aging, loans, promotion/relegation including playoff outcomes, restructure, prizes, cup start, classification reset, season bump, fixtures) SHALL run under a session advisory lock keyed `hashtext(country)`. Inside the lock it re-reads from the database the current season, its match counts, its playoff state and the country's rollover progress marker. New fixtures MUST be built from the leagues as they stand after promotion/relegation and any restructure.

#### Scenario: Three replicas see the season complete
- **WHEN** three replicas run the rollover for the same finished country at once
- **THEN** the season advances by exactly 1, one cup is created and each league gets one schedule

#### Scenario: Relegated team
- **WHEN** a team is relegated from division D1 at rollover
- **THEN** no new-season D1 fixture includes it

#### Scenario: Restructure on three replicas
- **WHEN** three replicas run the rollover for a country that must be restructured
- **THEN** its leagues are archived and created once, and the bottom tier is topped up once with exactly the missing CPU teams

## ADDED Requirements

### Requirement: Resumable season rollover
Each step of a country's rollover SHALL be recorded in the country's progress marker (`countries.rollover_season`, `rollover_step`, `rollover_state`) as soon as it completes. A rollover that finds a marker for the season it is closing MUST skip the recorded steps and continue from the next one, using the standings and moves stored with the marker. The season bump records itself in the same statement; the new season's fixtures clear the marker.

#### Scenario: Stopped after the prizes
- **WHEN** a rollover stops after promotion/relegation and the prizes, and the next tick runs it again
- **THEN** players age once, prizes and season income are paid once, one cup is created and the season advances by exactly 1

#### Scenario: Stopped after the season bump
- **WHEN** a rollover stops after moving the current season but before generating fixtures
- **THEN** the next tick generates the new season's fixtures and nothing else of the rollover runs again
