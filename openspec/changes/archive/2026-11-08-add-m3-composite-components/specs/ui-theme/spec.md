# Spec Delta

## ADDED Requirements

### Requirement: Search bar
Every free-text search field SHALL be an M3 search bar: a pill on `surfaceContainerHigh`, 56 px tall (48 px in a compact filter row), with a leading search icon and its placeholder as the accessible name. While it holds text it SHALL show a trailing clear button named "Clear search" that empties it and returns focus to it.

#### Scenario: Search page
- **WHEN** a player opens the search page in light mode
- **THEN** the field is a 56 px pill on `#e8e9e2` with a search icon and no clear button

#### Scenario: Clearing a search
- **WHEN** the player has typed "ribera" and taps "Clear search"
- **THEN** the field is empty, keeps the focus, and the clear button is gone

#### Scenario: Compact filter
- **WHEN** the national teams filter or the admin users list shows its search field
- **THEN** it is the same pill, 48 px tall

### Requirement: Bottom sheet
A sheet that slides up from the bottom SHALL be an M3 modal bottom sheet on `surfaceContainerLow` with 28 px top corners and a centred 32×4 px drag handle, at most 640 px wide and 90 % of the window tall. Its footer SHALL stay clear of the bottom safe area.

#### Scenario: Market filters
- **WHEN** a player opens the transfer market's filters on a 360 px phone in light mode
- **THEN** a sheet on `#f3f4ee` with 28 px top corners and a drag handle slides up, with the filters, "Clear" and "Show N results"

### Requirement: Stat tiles
A glanceable figure with a label SHALL be shown on a filled card (`surfaceContainerHighest`) with its icon and label in `onSurfaceVariant` and the value in the `stat` type style. A tile that leads somewhere SHALL be a single link covering the whole card. A value may keep a meaning colour (a negative budget in error red).

#### Scenario: Dashboard KPIs
- **WHEN** a player opens the dashboard in light mode
- **THEN** budget, squad, position and fans show as `#e2e3dd` tiles, each a link to its page

#### Scenario: Negative budget
- **WHEN** the club's budget is below zero
- **THEN** the budget value shows in the theme's error colour
