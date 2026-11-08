# Spec Delta

## ADDED Requirements

### Requirement: Top app bar
The header SHALL use the `surface` role as background, `onSurface` for its icons and the `primary` role for the logo. While the page is scrolled more than 0 px under it, the header SHALL use `surfaceContainer`. It MUST NOT cast a shadow. On phones it keeps hiding on scroll down and returning on scroll up.

#### Scenario: Page at the top
- **WHEN** a signed-in user opens the dashboard in light mode and has not scrolled
- **THEN** the header background is `#f9faf4`, the logo is `#006e1c` and the icons are `#1a1c19`

#### Scenario: Content under the header
- **WHEN** the same user scrolls the dashboard down by 200 px on a 1280 px screen
- **THEN** the header stays visible with background `#eeeee8`

### Requirement: Button emphasis levels
Buttons SHALL come in four emphasis levels: filled (`primary` / `onPrimary`), tonal (`secondaryContainer` / `onSecondaryContainer`), outlined (`outline` border, `primary` text) and text. Every level SHALL be pill-shaped, at least 40 px tall with a pointer and 44 px on touch screens, with sentence-case labels.

#### Scenario: Tonal button
- **WHEN** a tonal button is rendered in light mode
- **THEN** its background is `#d5e8cf`, its label `#3b4b38`, and it has no border or shadow

#### Scenario: Touch height
- **WHEN** any button is rendered on a 360 px touch screen
- **THEN** it is at least 44 px tall

### Requirement: Chip shape
Chips SHALL have 8 px corners, 32 px height at the default size and 24 px at the compact size, with clickable chips keeping a tap area of at least 44 px. A selected filter chip SHALL use `secondaryContainer` / `onSecondaryContainer` with a leading check icon; an unselected one an `outlineVariant` border.

#### Scenario: Compact status chip
- **WHEN** a compact chip with the success colour is shown in a list
- **THEN** it is 24 px tall with 8 px corners and keeps its success colour

#### Scenario: Selected filter chip
- **WHEN** the user selects a filter chip in light mode
- **THEN** it turns `#d5e8cf` with `#3b4b38` text and shows a check icon before its label

### Requirement: Segmented buttons
A single-choice button group SHALL render as M3 segmented buttons: a pill-shaped outline in the `outline` role, 1 px dividers between segments, the selected segment filled with `secondaryContainer` and `onSecondaryContainer` text. By default a segment is at least 40 px tall with a pointer and 44 px on touch screens; a group a page deliberately compacts keeps its own height.

#### Scenario: Position filter in the transfer market
- **WHEN** the transfer market's position filter is shown with "MID" selected in light mode
- **THEN** the group is pill-shaped with a `#72796f` outline and only the "MID" segment has the `#d5e8cf` fill

#### Scenario: Compact mentality picker
- **WHEN** the lineup page shows its mentality picker on a 360 px touch screen
- **THEN** it keeps its compact 36 px height and takes the segmented-button colours and pill outline

#### Scenario: No overflow at 360 px
- **WHEN** any segmented group is shown on a 360 px screen
- **THEN** the page does not scroll horizontally

### Requirement: Card styles
A card SHALL support three styles with 12 px corners: elevated (`surfaceContainerLow`, level-1 shadow), filled (`surfaceContainerHighest`, no shadow) and outlined (`surface` with an `outlineVariant` border, no shadow).

#### Scenario: Filled card in dark mode
- **WHEN** a filled card is shown in dark mode
- **THEN** its background is `#333531` and it casts no shadow

### Requirement: Tabs, switches and progress
Tabs SHALL mark the active tab with a 3 px `primary` indicator with rounded top corners, the active label in `primary` and inactive labels in `onSurfaceVariant`. A switch SHALL have a 52×32 px track: off, `surfaceContainerHighest` with a 2 px `outline` border and a 16 px `outline` thumb; on, a `primary` track and a 24 px `onPrimary` thumb. A linear progress bar SHALL be 4 px tall with rounded ends on a `secondaryContainer` track.

#### Scenario: Switch on
- **WHEN** the league page's "Live" switch is on in light mode
- **THEN** its track is `#006e1c` and its thumb is a 24 px `#ffffff` circle

#### Scenario: Switch tap area
- **WHEN** a switch is shown on a 360 px touch screen
- **THEN** its tap area is at least 44 px tall

### Requirement: Menus and dialogs
Menus and popovers SHALL use `surfaceContainer` with 4 px corners. Dialogs SHALL use `surfaceContainerHigh` with 28 px corners, except the full-screen dialogs phones get, which keep square corners.

#### Scenario: Dialog in light mode
- **WHEN** a confirmation dialog opens on a 1280 px screen in light mode
- **THEN** its background is `#e8e9e2` with 28 px corners

#### Scenario: Full-screen dialog on a phone
- **WHEN** a responsive dialog opens full-screen on a 360 px phone
- **THEN** it has square corners
