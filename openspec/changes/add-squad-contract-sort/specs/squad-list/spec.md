# Spec Delta

## Purpose

How a club's squad view sorts its players and flags the contracts that need a renewal decision.

## ADDED Requirements

### Requirement: Contract sort
The squad view of the manager's own club SHALL offer a Contract sort. Picking it MUST sort by seasons left on the contract, most first, show the seasons left on every row and list the whole squad without position groups. The direction arrow MUST flip the order. A rival club's squad MUST NOT offer it, since its contract terms are redacted.

#### Scenario: Longest contracts first
- **WHEN** a manager whose squad has contracts of 3, 2, 2 and 1 seasons picks the Contract sort
- **THEN** the rows read 3, 2, 2, 1 seasons left in one list, the two 2-season players in name order

#### Scenario: Flip the order
- **WHEN** the manager then flips the direction arrow
- **THEN** the 1-season player comes first and the 3-season player last

#### Scenario: Rival squad
- **WHEN** a manager opens another club's squad
- **THEN** the Sort by menu has no Contract option

### Requirement: Sort tie-break
Players equal on the chosen sort field SHALL be listed by name, A-Z, whichever direction the arrow points.

#### Scenario: Equal contracts
- **WHEN** two players both have 2 seasons left and the squad is sorted by contract, descending or ascending
- **THEN** the one whose name comes first alphabetically is listed first

### Requirement: Last-season warning
On the manager's own squad, every player with at most `CONTRACT_EXPIRY_WARNING_YEARS` (1) season left SHALL show a "Last season" chip in the row, whatever the sort. A player on loan to the club MUST NOT show it, since renewing is the parent club's call.

#### Scenario: Last season
- **WHEN** a player of the own squad has 1 season left
- **THEN** the row shows "Last season" under any sort

#### Scenario: Loaned-in player
- **WHEN** a player loaned in from another club has 1 season left
- **THEN** the row shows no "Last season" chip
