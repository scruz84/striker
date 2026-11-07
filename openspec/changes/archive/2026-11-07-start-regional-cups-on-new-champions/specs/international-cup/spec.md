# Spec Delta

## ADDED Requirements

### Requirement: Regional cup start
Every 5 minutes the engine SHALL check each region of the active countries. A region MUST start a new edition when none of its editions is running, at least 2 of its countries with a tier-1 league are fresh (latest tier-1 champion has not entered yet), and either all of them are fresh or one has 2 or more unused champions.

#### Scenario: Every country has a new champion
- **WHEN** Spain, France and Italy are in europe, all three have finished a season since the last European Cup, and no European Cup is running
- **THEN** the next check starts a European Cup with the three latest champions

#### Scenario: Waiting for the last country
- **WHEN** Spain and France have new champions but Italy is still playing its season and no country has finished two seasons since it last entered
- **THEN** no European Cup starts yet

#### Scenario: A slow country does not hold the region back
- **WHEN** Italy is still playing and Spain finishes a second season since it last entered while France has one new champion
- **THEN** a European Cup starts with Spain's latest champion and France's, and Italy enters a later edition

#### Scenario: Edition still running
- **WHEN** every country has a new champion but the current European Cup has not finished
- **THEN** no new edition starts until it is completed

#### Scenario: One country only
- **WHEN** only one country in a region has a new champion, however many seasons it has finished
- **THEN** no edition starts in that region

### Requirement: Regional cup entrants
Each fresh country SHALL enter once, with its most recent tier-1 champion (lowest league id when it has several tier-1 groups). The engine MUST record, per edition, each country and the domestic season of the champion it sent; a country is fresh again only once it has a champion from a newer season.

#### Scenario: Champion already played
- **WHEN** France's season-4 champion played the last European Cup and France has not finished season 5
- **THEN** France is not fresh and does not enter the next edition

#### Scenario: Two titles since the last edition
- **WHEN** Spain finished seasons 5 and 6 since it last entered with its season-4 champion
- **THEN** Spain enters with its season-6 champion and its season-5 champion never plays the cup

#### Scenario: Existing editions on upgrade
- **WHEN** the migration runs on a database whose last European Cup had Spain's season-3 champion
- **THEN** that entry is recorded and Spain is fresh only after finishing season 4

### Requirement: Regional cup editions
An edition SHALL be created with its entries and first round in one transaction, numbered 1, 2, 3... within its region, and run as a single-elimination cup whose rounds kick off 24 hours apart. The cups are European Cup, Copa America, Asian Cup and African Cup.

#### Scenario: Third edition
- **WHEN** the americas region already had two Copa America editions and a new one starts
- **THEN** it is shown as "Copa America - Edition 3"

#### Scenario: Failure while writing
- **WHEN** writing the first round fails
- **THEN** no cup row or entry remains, and the next check tries again

### Requirement: Regional cup start on several replicas
The regional cup check SHALL run under one global session lock taken without waiting, and re-read each region's state inside it.

#### Scenario: Three replicas tick together
- **WHEN** three replicas run the check at once with a European Cup due
- **THEN** exactly one European Cup edition is created
