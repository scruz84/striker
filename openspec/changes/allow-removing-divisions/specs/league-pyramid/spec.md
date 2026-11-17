# Spec Delta

## MODIFIED Requirements

### Requirement: Structure fits existing teams
Saving a structure, global or per country, SHALL be rejected when its places (the sum of groups × teams per group) are fewer than the user-owned teams of any active country it applies to, naming the country and both numbers. Fewer places than a country's teams is allowed: the surplus CPU teams are dissolved at that country's next rollover (see "Surplus CPU teams are dissolved").

#### Scenario: Shrinking below the current team count
- **WHEN** an active country has 310 teams, 12 of them user-owned, and the admin saves 1/2/4/8 groups of 10 (150 places) and confirms
- **THEN** the structure is saved, and the country's next rollover dissolves 160 CPU teams

#### Scenario: Not enough places for the users
- **WHEN** an active country has 40 user-owned teams and the admin saves 1/2 groups of 10 (30 places)
- **THEN** the save is rejected, naming the country, 30 places and 40 user teams

### Requirement: Restructure detection
At rollover, a country SHALL be restructured when its active leagues differ from its effective structure (its own structure if it has one, otherwise the global one) in the number of tiers, the groups of any tier, or the teams per group. Saving a structure, or setting or removing a country's own, SHALL change nothing until each affected country's next rollover. A country already in its effective structure is left alone.

#### Scenario: Old world
- **WHEN** a country has 1/4/16 groups and the saved structure is 1/2/4/8/16
- **THEN** its next rollover restructures it

#### Scenario: Country with its own structure
- **WHEN** the global structure is 1/2/4/8/16 and Andorra has its own 1/2 structure
- **THEN** Andorra's next rollover reshapes it into 2 tiers and leaves every other country on 5

#### Scenario: Removing a country's own structure
- **WHEN** the admin removes Andorra's own 1/2 structure
- **THEN** Andorra's next rollover reshapes it into the global structure, topping up with new CPU teams

### Requirement: Restructure fills groups and tops up
Within a tier, teams SHALL be dealt to groups in a snake order by their position in the ordering (1..G, then G..1, and so on). When a structure has more places than teams, the bottom tier's empty places SHALL be filled with newly generated CPU teams, created like those of a new world. When it has fewer places than teams, the teams beyond the last place are dissolved, never placed.

#### Scenario: Top-up
- **WHEN** 210 teams are restructured into 310 places
- **THEN** 100 new CPU teams are created in tier 5, each with a full squad, lineup and AI level

#### Scenario: Snake order in tier 2
- **WHEN** tier 2 gets ordered teams t1..t20 into 2 groups
- **THEN** group 1 gets t1, t4, t5, t8, … and group 2 gets t2, t3, t6, t7, …

#### Scenario: Removing the bottom tier
- **WHEN** a 1/2/4/8/16 country with 310 CPU teams is restructured into 1/2/4/8
- **THEN** the first 150 teams in the ordering are placed in tiers 1-4 and the other 160 are dissolved

## ADDED Requirements

### Requirement: Per-country structure
An admin SHALL be able to give an active country its own full structure (tiers, names, groups, teams per group), validated by the same pyramid rules and limits as the global one, and to remove it again. The admin overview MUST show which countries have their own structure and which are pending a restructure.

#### Scenario: Invalid country structure
- **WHEN** the admin saves 1/4 groups as Andorra's own structure
- **THEN** the save is rejected with "tier 2 must have exactly 2 groups (twice tier 1)"

#### Scenario: Global change and an overridden country
- **WHEN** Andorra has its own structure and the admin saves a new global structure
- **THEN** Andorra keeps its own structure and is not marked pending by the global change

### Requirement: Confirming a structure that dissolves teams
A save, global or per country, that would leave any active country with fewer places than teams SHALL first be refused with the number of CPU teams each country would lose, and SHALL go through only when the same save is sent again with explicit confirmation. A save that dissolves nothing needs no confirmation.

#### Scenario: Admin is warned
- **WHEN** the admin saves 1/2/4/8 groups of 10 for a world whose countries have 310 teams each
- **THEN** the save is refused, listing 160 teams to dissolve in each country, and nothing changes

#### Scenario: Admin confirms
- **WHEN** the admin sends the same save with confirmation
- **THEN** the structure is saved and each country shrinks at its own next rollover

### Requirement: Surplus CPU teams are dissolved
When a restructure has fewer places than teams, the teams to dissolve SHALL be the last in the restructure ordering that are CPU-owned. A user-owned team that would fall beyond the last place takes the last place of the bottom tier instead, and the lowest-ranked CPU team placed there is dissolved in its place. Every team dissolved is recorded with the season it ended in.

#### Scenario: User team at the bottom
- **WHEN** a country shrinks from 310 to 150 places and a user's team is 300th in the ordering
- **THEN** that team plays next season in the bottom tier and the 150th team in the ordering, a CPU team, is dissolved instead

### Requirement: Dissolved team
A dissolved team SHALL never play, be assigned to a user, act as a CPU team, or take part in the market again. Its players become free agents, its trainer returns to the market, and its pending offers, listings, loans and unplayed friendlies end. Its past matches, standings and history MUST stay viewable, and its page MUST read "Dissolved in season N".

#### Scenario: Players freed
- **WHEN** a CPU team with 18 senior players and 3 academy prospects is dissolved
- **THEN** its 18 senior players are free agents and its 3 academy prospects retire

#### Scenario: Old result still opens
- **WHEN** a viewer opens a season-3 match a dissolved team played
- **THEN** the match shows as before, with the team's name linking to its page marked "Dissolved in season 4"

#### Scenario: Loan in progress
- **WHEN** a team is dissolved while it has a player out on loan and one in on loan
- **THEN** the player out on loan becomes a free agent at the end of his loan, and the player in on loan returns to his club at once
