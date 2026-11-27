# Spec Delta

## ADDED Requirements

### Requirement: Restructure leaves no team behind
After a restructure, every non-dissolved team of the country SHALL play in a league of its structure. A team planned for dissolution that is not dissolved is protected, and the lowest-ranked unprotected CPU team takes its place on the dissolution list. A team still in a league outside the structure is placed with the same ordering. A league outside the structure is then empty and archived.

#### Scenario: Club claimed during the restructure
- **WHEN** a country shrinks from 310 to 150 places and the CPU club ranked 200th gets a user before it is dissolved
- **THEN** that club plays next season in the bottom tier, the CPU club ranked 150th is dissolved instead, and every bottom-tier group has 10 teams

#### Scenario: Dissolution error
- **WHEN** dissolving one planned CPU club fails with a database error
- **THEN** that club is kept like a protected one, another CPU club is dissolved instead, and the old league it was in is archived

#### Scenario: Leftover groups from an earlier restructure
- **WHEN** a country whose structure is 1/2/4 x 10 has active leagues I.1, II.1, II.2, III.1, III.2 and III.3 with 10 teams each (no III.4)
- **THEN** it plays its season in legacy mode, and its next rollover creates III.4, deals the 30 tier-3 teams into the four groups, adds 10 new CPU teams, and leaves 1/2/4 groups of 10

### Requirement: Restructure result is checked
After a restructure, the country's active leagues SHALL be compared with its structure. If they differ, the rollover SHALL stop before the cup step, and the next rollover tick runs the restructure again. After the 3rd failed attempt in the same rollover, the rollover continues, an error naming every differing league is logged, and the country stays pending for its next rollover.

#### Scenario: Transient failure
- **WHEN** creating group III.4 fails on the first attempt and succeeds 5 minutes later
- **THEN** the second tick finishes the restructure, and the season advances once with 1/2/4 groups

#### Scenario: Persistent failure
- **WHEN** the country still differs from its structure after 3 attempts
- **THEN** the season advances anyway, every league keeps its fixtures, the log names the differing leagues, and the admin still lists the country as pending

### Requirement: Admin sees why a country is pending
The admin overview SHALL list, for each country pending a restructure, every difference between its active leagues and its structure. Each difference is one of: an extra group (tier, group, name, teams), a missing group (tier, group), a group of the wrong size (name, teams, expected), or a shape that breaks the doubling rule. The World tab SHALL show these under the pending notice.

#### Scenario: Leftover groups
- **WHEN** the admin opens the World tab while ES has 1/2/3 groups against a 1/2/4 x 10 structure
- **THEN** under the pending notice it reads, for ES, that group III.4 is missing and that the shape breaks the doubling rule

#### Scenario: Nothing pending
- **WHEN** every active country matches its structure
- **THEN** the overview's details list is empty and the World tab shows no pending notice
