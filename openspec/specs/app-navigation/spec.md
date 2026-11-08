# app-navigation Specification

## Purpose
How a signed-in player with a team moves between the app's sections at every window width: which
Material 3 navigation layout shows, what it contains and how it looks, and how it sits next to the
header and the page content.

## Requirements

### Requirement: Navigation layout by window width
A signed-in player with a team SHALL get exactly one navigation layout: a navigation bar under 900 px wide, a navigation rail from 900 px to 1199 px, and a standard navigation drawer from 1200 px. A visitor who is signed out, or still choosing a country, SHALL get none.

#### Scenario: Phone
- **WHEN** a player opens the dashboard on a 360 px wide screen
- **THEN** the navigation bar shows at the bottom, and neither a rail nor a permanent drawer shows

#### Scenario: Laptop
- **WHEN** the same player opens it on a 1024 px wide screen
- **THEN** a navigation rail shows on the left, and neither the bar nor a permanent drawer shows

#### Scenario: Desktop
- **WHEN** the same player opens it on a 1280 px wide screen
- **THEN** the standard navigation drawer shows on the left, and neither the bar nor the rail shows

### Requirement: Navigation bar
The bar SHALL be 80 px tall plus the bottom safe area, on `surfaceContainer`, with five entries: Dashboard, Squad, Lineup, Market and More. The current entry SHALL show a 64×32 px pill in `secondaryContainer` behind a filled icon and a bold label. Other entries SHALL show an outlined icon. Labels SHALL be 12 px, and the page SHALL keep 80 px plus the safe area clear above the bar.

#### Scenario: Current entry
- **WHEN** a player is on `/transfers` in light mode
- **THEN** the Market entry shows a 64×32 px `#d5e8cf` pill behind a filled icon, and the other entries have no pill

#### Scenario: Last content not hidden
- **WHEN** a player scrolls to the bottom of any page on a 360 px phone
- **THEN** the page's last line ends above the bar's 80 px

#### Scenario: Labels fit
- **WHEN** the bar shows in Spanish on a 360 px phone
- **THEN** "Panel", "Plantilla", "Alineación", "Mercado" and "Más" each fit on one line at 12 px without horizontal page scroll

### Requirement: Navigation rail
The rail SHALL be an 80 px wide column that stays visible under the header while the page scrolls. It SHALL hold a menu button, which opens the modal navigation drawer, and then Dashboard, Squad, Lineup and Market, each with a 56×32 px pill indicator and a 12 px label, styled like the navigation bar. Every rail entry SHALL be at least 48 px tall.

#### Scenario: Rail entry selected
- **WHEN** a player is on `/dashboard/lineup` on a 1024 px screen
- **THEN** the Lineup entry shows the `secondaryContainer` pill with a filled icon

#### Scenario: Less frequent destination
- **WHEN** the player on a 1024 px screen taps the rail's menu button and then Finances
- **THEN** the modal drawer closes and the finances page opens

#### Scenario: Rail stays put
- **WHEN** the player scrolls a long page down by 600 px on a 1024 px screen
- **THEN** the rail is still fully visible under the header

### Requirement: Modal navigation drawer
"More" on the bar and the rail's menu button SHALL open every destination, grouped as My club, Competitions, Social and Other, in a modal drawer from the left on `surfaceContainerLow` with 16 px corners on its open side. Each item SHALL be a 56 px tall pill, and the current one SHALL use `secondaryContainer` with `onSecondaryContainer` text. Choosing an item SHALL close the drawer.

#### Scenario: Current item
- **WHEN** a player on `/cup/ties/7` opens the drawer from "More" in light mode
- **THEN** the Cup item is a `#d5e8cf` pill with a filled icon

#### Scenario: Admin entry
- **WHEN** an admin opens the drawer
- **THEN** the Other group ends with Admin, and for a player who is not an admin it does not

### Requirement: Standard navigation drawer
From 1200 px the grouped navigation SHALL be a permanent drawer next to the page with the modal drawer's items, on the page `surface`. It SHALL stay under the header while the page scrolls, scrolling on its own when it is taller than the window, and MUST never cover the header.

#### Scenario: Header stays readable
- **WHEN** a player scrolls the dashboard down by 400 px on a 1280 px screen
- **THEN** the header's logo is still fully visible and the drawer starts right below the header

### Requirement: Navigation icons
Every navigation entry in the bar, the rail and both drawers SHALL show an outlined icon, and a filled icon of the same glyph while it is the current entry.

#### Scenario: Icon swaps on selection
- **WHEN** the player moves from `/dashboard` to `/transfers` on a phone
- **THEN** the Dashboard entry switches to the outlined dashboard icon and the Market entry to the filled swap icon

## Code map

- Layout choice: `frontend/src/app/navigation.ts` (`useNavMode`, `NavMode`, `NAV_RAIL_WIDTH`,
  `DESKTOP_HEADER_HEIGHT`, `STICKY_NAV_COLUMN_SX`, `isNavSelected`), wired in
  `frontend/src/app/Layout.tsx`.
- Destinations: `frontend/src/app/navItems.tsx` (`useNavGroups` for the drawers,
  `usePrimaryDestinations` for the bar and the rail, `navIcon` for the outlined / filled pairs).
- Navigation bar: `frontend/src/app/BottomNav.tsx` (`BOTTOM_NAV_HEIGHT`, also read by
  `features/lineup/LineupSaveBar.tsx`) with its look in `frontend/src/theme.ts`
  (`MuiBottomNavigation`, `MuiBottomNavigationAction`).
- Rail: `frontend/src/app/NavRail.tsx`. Drawers: `frontend/src/app/NavDrawer.tsx` (`modal` and
  `standard` variants, `NAV_DRAWER_WIDTH`).
- Tests: `navigation.test.ts`, `BottomNav.test.tsx`, `NavRail.test.tsx`, `NavDrawer.test.tsx`,
  `Layout.test.tsx` (all in `frontend/src/app/`), with `frontend/src/testUtils/viewport.ts`
  (`stubViewportWidth`).
- Strings: `nav.bottom.*`, `nav.groups.*`, `header.nav.*` in `i18n/locales/{en,es}.json`.

