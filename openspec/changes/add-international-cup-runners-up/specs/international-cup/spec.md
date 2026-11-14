# Spec Delta

## MODIFIED Requirements

### Requirement: Regional cup entrants
Each fresh country SHALL enter once, with the champion and the runner-up (final positions 1 and 2) of its most recent tier-1 season (lowest league id when it has several tier-1 groups). With no recorded runner-up it enters with its champion alone. The engine MUST record, per edition, each entrant team with its country, final position and that domestic season; a country is fresh again only once it has a champion from a newer season.

#### Scenario: Champion already played
- **WHEN** France's season-4 top two played the last European Cup and France has not finished season 5
- **THEN** France is not fresh and does not enter the next edition

#### Scenario: Two titles since the last edition
- **WHEN** Spain finished seasons 5 and 6 since it last entered with its season-4 top two
- **THEN** Spain enters with its season-6 champion and runner-up, and its season-5 top two never play the cup

#### Scenario: Top two of each country
- **WHEN** a European Cup starts with Spain and France fresh
- **THEN** it has 4 entrants: Spain's 1st and 2nd and France's 1st and 2nd of their latest season

#### Scenario: Missing runner-up
- **WHEN** Italy's latest tier-1 season has a champion but no recorded 2nd place
- **THEN** Italy enters with its champion only and the edition still starts

#### Scenario: Existing editions on upgrade
- **WHEN** the migration runs on a database whose last European Cup had Spain's season-3 champion
- **THEN** that entry is kept as Spain's 1st place and Spain is fresh only after finishing season 4

## ADDED Requirements

### Requirement: Regional cup draw
With N entrants and P the largest power of two ≤ N, round 1 SHALL give 2P − N byes when N ≠ P. Byes go to champions first, then runners-up, each by current league points (ties at random), unless moving a bye to a team of the same position avoids a same-country tie. The other teams are paired champion vs a runner-up of another country where possible, and never two teams of one country when another pairing exists. Byes alternate with real ties so two byes never meet in round 2.

#### Scenario: Two countries
- **WHEN** Spain (A1, A2) and France (B1, B2) enter
- **THEN** round 1 is A1 vs B2 and B1 vs A2, with no byes

#### Scenario: Three countries
- **WHEN** Spain, France and Italy enter with 6 teams and Spain's and France's champions have the most points among the champions
- **THEN** A1 and B1 get byes, C1 plays A2 or B2, and the other two runners-up play each other

#### Scenario: Four countries
- **WHEN** four countries enter with 8 teams
- **THEN** there are no byes and every round-1 tie is a champion against another country's runner-up

#### Scenario: Bye that avoids a same-country tie
- **WHEN** Spain (A1, A2) and France (B1, no runner-up) enter and B1 has more points than A1
- **THEN** A1 gets the bye and B1 plays A2, instead of A1 playing A2

#### Scenario: Round 2 after byes
- **WHEN** A1 and B1 got byes in a 6-team edition
- **THEN** in round 2 each of them plays a round-1 winner, not each other
