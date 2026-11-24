# Spec Delta

## Purpose

States which stored match data the game keeps and for how long, so an admin can bound database
growth by archiving the per-minute detail of old seasons while every match keeps its record. It also
bounds the append-only logs that are only ever read over a short recent window.

## ADDED Requirements

### Requirement: Instant archive setting
The game SHALL have one global instant archive setting: whether it is enabled (default off) and how many finished seasons keep their per-minute instants (default 1, an integer from 0 to 10). While it is off, fixtures lose their instants only through the instant retention by days.

#### Scenario: Fresh world
- **WHEN** a world is created or an existing database is upgraded
- **THEN** the archive is off and keeps 1 finished season

#### Scenario: Off keeps every season
- **WHEN** the archive is off and the daily archive pass fires
- **THEN** no fixture within the days kept loses its instants, whatever its season

### Requirement: Admin configures the archive
An admin SHALL see and change both values in the admin web's Rules tab. A save with a count outside 0-10 MUST be rejected with no change. Only admins can read or change the setting.

#### Scenario: Turning it on
- **WHEN** an admin enables the archive with 2 seasons kept and saves
- **THEN** the admin overview reports it enabled with 2 seasons kept

#### Scenario: Count out of range
- **WHEN** an admin saves 11 seasons kept
- **THEN** the save is rejected and the setting is unchanged

#### Scenario: Not an admin
- **WHEN** a non-admin user calls the archive setting endpoint
- **THEN** the request is refused and the setting is unchanged

### Requirement: Instants are kept for a number of days
The game SHALL have a global instant retention: how many days a played fixture keeps its per-minute instants after kickoff (default 7, an integer from 0 to 365, 0 meaning for good). Every day, whether the season archive is on or off, every played league match, cup tie and friendly (World Cup group matches included) that kicked off more than that many days ago SHALL lose its instants.

#### Scenario: Old match
- **WHEN** fixtures keep 7 days and a league match, a cup tie and a friendly kicked off 8 days ago
- **THEN** the next daily pass deletes their instants and keeps their results, chronicle, ratings and lineups

#### Scenario: Recent match within the days kept
- **WHEN** fixtures keep 7 days and a match kicked off 2 days ago
- **THEN** it keeps its instants, so its live replay and zone duel still work

#### Scenario: Instants expire with the archive off
- **WHEN** the season archive is off and fixtures keep 7 days
- **THEN** the daily pass still deletes the instants of every fixture older than 7 days

#### Scenario: World Cup group match
- **WHEN** a World Cup group match kicked off more than the days kept ago
- **THEN** it loses its instants like any other fixture

#### Scenario: Zero days keeps instants
- **WHEN** an admin saves 0 days
- **THEN** no fixture loses its instants through this rule, and only the season archive, if on, deletes them

### Requirement: Admin configures the instant retention
An admin SHALL see and change the instant retention's days in the admin web's Rules tab. A save outside 0-365 MUST be rejected with no change. Only admins can change it.

#### Scenario: Changing the days
- **WHEN** an admin saves 14 days
- **THEN** the admin overview reports 14 days and fixtures keep their instants for 14 days

#### Scenario: Days out of range
- **WHEN** an admin saves 366 days
- **THEN** the save is rejected and the setting is unchanged

### Requirement: Which league matches and domestic cups are archived
With the archive on and K seasons kept, a country whose current season is S SHALL have archived every played league match of its leagues with season ≤ S − 1 − K, and every tie of its domestic cups and playoffs of those seasons once that cup is completed. The current season is never archived.

#### Scenario: Keeping one season
- **WHEN** the archive is on with 1 season kept and Spain is in season 5
- **THEN** Spain's league matches, cup ties and playoff ties of seasons 1-3 lose their instants, and those of seasons 4 and 5 keep them

#### Scenario: Keeping none
- **WHEN** the archive is on with 0 seasons kept and Spain is in season 5
- **THEN** only Spain's season 5 matches keep their instants

#### Scenario: Countries advance on their own
- **WHEN** Spain is in season 5 and France in season 4, with 1 season kept
- **THEN** Spain's season 3 is archived and France's season 3 is not

### Requirement: Which international cups are archived
A cup belonging to a region (regional club cups and international knockout cups) SHALL be archived once completed and its edition E satisfies E ≤ N − 1 − K, where N is the newest edition of that region and K the seasons kept.

#### Scenario: Regional edition
- **WHEN** the archive keeps 1 season and a region has editions 1-4, with edition 2 completed
- **THEN** the ties of editions 1 and 2 lose their instants, and those of editions 3 and 4 keep them

### Requirement: Which friendlies are archived
With the season archive on, a played friendly SHALL also be archived when its kickoff is earlier than the first league kickoff of season S − K in its home team's country (S being that country's current season, K the seasons kept). This covers national-team friendlies and World Cup group matches. If that season has no league match, no friendly of that country is archived.

#### Scenario: Friendly from an archived season
- **WHEN** the archive keeps 1 season, Spain is in season 5, and a Spanish club hosted a friendly before Spain's first season-4 league match
- **THEN** that friendly loses its instants

#### Scenario: Recent friendly
- **WHEN** the same club hosted a friendly after Spain's first season-4 league match
- **THEN** that friendly keeps its instants

### Requirement: Archiving keeps the match record
Archiving SHALL delete only per-minute instants. A match's result, events, stored ratings, lineups, substitutions, orders, attendance, weather and referees MUST stay, and an archived match MUST appear in standings, history, statistics and the match viewer as before.

#### Scenario: Viewing an archived match
- **WHEN** a viewer opens an archived league match
- **THEN** it shows its score, chronicle and stored ratings, and its Tácticas tab shows only the lineups pitch

### Requirement: Daily archive pass
The engine SHALL run the archive pass once a day: first the instant retention by days, then, while the archive is on, the season archive, in batches of at most 200 fixtures per statement, until no fixture to archive is left. Instants already deleted are never restored, even if the archive is turned off or the count raised.

#### Scenario: First pass after turning it on
- **WHEN** an admin turns the archive on over several old seasons
- **THEN** the next daily pass deletes all their instants batch by batch, and later passes find nothing left from those seasons

#### Scenario: Raising the count later
- **WHEN** seasons 1-3 were archived and the admin then raises the seasons kept from 1 to 3
- **THEN** seasons 1-3 stay without instants and no newer season is archived until it falls outside the new count

### Requirement: Old ledger rows
Every day the engine SHALL delete a team's ledger rows older than 180 days that are not about a player (wages, gate, sponsorship, prize money, maintenance, season income, admin adjustments, club resets). Ledger rows about a player (transfers, release clauses, loan fees, severance) MUST stay. Budgets never change.

#### Scenario: Old wage row
- **WHEN** a club's wage row is 200 days old
- **THEN** the next daily pass deletes it and the club's budget is unchanged

#### Scenario: Old transfer row
- **WHEN** a club's transfer fee row for a player is 200 days old
- **THEN** it stays and still shows in that player's history

#### Scenario: Recent rows
- **WHEN** a club's ledger row is 10 days old
- **THEN** it stays and the finance page shows it as before

### Requirement: Old offer log rows
Every day the engine SHALL delete transfer offer log rows older than 30 days. The market's offer counts only cover the last day, so they never change.

#### Scenario: Old offer log row
- **WHEN** an offer was logged 40 days ago
- **THEN** the next daily pass deletes its log row, and an offer logged an hour ago stays
