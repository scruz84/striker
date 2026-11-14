# Spec Delta

## MODIFIED Requirements

### Requirement: League kickoff schedule
A league's kickoff weekdays SHALL come from its own admin-set override, or else the app-wide default. Its kickoff time SHALL come from its override, or else the country's league kickoff time (local, default 14:00) plus 15 minutes for each tier below tier 1. Kickoffs are dated on the country's local clock. Fixtures are generated a season at a time on that cadence.

#### Scenario: Staggered tiers
- **WHEN** a 5-tier Spanish country with the default 14:00 kickoff generates its fixtures
- **THEN** tier 1 kicks off at 14:00, tier 2 at 14:15, tier 3 at 14:30, tier 4 at 14:45 and tier 5 at 15:00, Madrid time

#### Scenario: Daylight saving
- **WHEN** a season's matchdays span the end of summer time in Spain
- **THEN** tier 1 still kicks off at 14:00 on the Madrid clock before and after the change

#### Scenario: Override wins
- **WHEN** the admin gives league III.2 its own schedule at 18:00
- **THEN** III.2 kicks off at 18:00 and the other tier-3 groups keep 14:30

#### Scenario: League created by a restructure
- **WHEN** a restructure creates league IV.5 in a country whose kickoff is 14:00
- **THEN** IV.5 has no override and kicks off at 14:45 local time

### Requirement: Rescheduling on schedule change
Changing a league's schedule override, clearing it, changing the app-wide default, or changing a country's league kickoff time MUST immediately re-date every unplayed match of the affected leagues onto the new schedule, keeping round order. A league with its own override is unaffected by a default or country change.

#### Scenario: Admin changes the default schedule
- **WHEN** the app-wide default schedule changes
- **THEN** every unplayed match in leagues without an override is re-dated onto it, round by round
- **AND** leagues with their own override keep their dates

#### Scenario: Admin changes a country's kickoff
- **WHEN** the admin moves Italy's league kickoff from 14:00 to 15:00
- **THEN** Italy's unplayed matches without an override are re-dated to 15:00 for tier 1, 15:15 for tier 2 and so on, and no other country's matches move

## ADDED Requirements

### Requirement: Country league kickoff setting
Each country SHALL have a league kickoff time on its local clock, from 00:00 to 22:45 on a quarter-hour. It defaults to 14:00, is set when the country is added, and the admin can change it later. It is the tier-1 time; lower tiers add 15 minutes per tier. A value that would push the deepest tier past 23:59 SHALL be rejected.

#### Scenario: Adding a country
- **WHEN** the admin adds Portugal with kickoff 13:00
- **THEN** Portugal's tier 1 kicks off at 13:00 Lisbon time and tier 5 at 14:00

#### Scenario: Too late for the deepest tier
- **WHEN** the admin sets a 5-tier country's kickoff to 23:30
- **THEN** the change is rejected, because tier 5 would kick off at 00:30 the next day
