# Spec Delta

## MODIFIED Requirements

### Requirement: Season rollover once per country
A country's rollover (aging, loans, promotion/relegation including playoff outcomes, restructure, prizes, cup start, classification reset, season bump, fixtures) SHALL run under a session advisory lock keyed `hashtext(country)`. Inside the lock it re-reads from the database the current season, its match counts and its playoff state. New fixtures MUST be built from the leagues as they stand after promotion/relegation and any restructure.

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

### Requirement: Playoff created once
A country's playoff ties SHALL be created under the same per-country lock as its rollover, after re-reading inside the lock whether the season already has a playoff. A unique index on (country, season) for playoffs MUST reject a second one. The playoff's ties are simulated under the existing per-tie cup lock.

#### Scenario: Two replicas see the league end
- **WHEN** two replicas' season jobs find the same country's league matches all broadcast
- **THEN** exactly one playoff with one set of ties is created, and each team gets one qualification notification

#### Scenario: Cup advance ignores the playoff
- **WHEN** the cup job runs while every playoff tie is resolved
- **THEN** it creates no second round and crowns no champion for the playoff
