# Spec Delta

## MODIFIED Requirements

### Requirement: Mobile tabs
On viewports narrower than the `md` breakpoint (900 px), the page SHALL show four full-width tabs in this order: "Tabla", "Jornadas", "Datos" (league leaders and team of the week) and "Traspasos" (recent transfers only). In a past season "Tabla", "Jornadas" and "Datos" SHALL be shown. "Tabla" SHALL be selected on open, or "Datos" when the URL names a team of the week. Labels MUST fit at 360 px without clipping, each tab at least 44 px tall.

#### Scenario: Transfers on a phone
- **WHEN** a user on a 360 px phone opens the "Datos" tab
- **THEN** the leaders panel is shown, and the recent-transfers panel is shown only under "Traspasos"

#### Scenario: English labels
- **WHEN** the language is English
- **THEN** the tabs read "Table", "Fixtures", "Stats" and "Transfers"

#### Scenario: Past season on a phone
- **WHEN** a user on a 360 px phone opens `/division/7?season=2`
- **THEN** the "Tabla", "Jornadas" and "Datos" tabs are shown

### Requirement: Desktop layout
From the `md` breakpoint up, the page SHALL show the standings (8 of 12 columns) beside a stack of league leaders, team of the week and recent transfers (4 of 12 columns), with the matchday navigator and the two matchday cards side by side below, and no tabs.

#### Scenario: Desktop browse
- **WHEN** a user on a 1280 px screen taps "›" in the navigator
- **THEN** only the two matchday cards change, and the standings and the side panels stay as they were

### Requirement: Past season view
A past season SHALL be read-only: its final table from the history endpoint, with zones from the recorded promotions and relegations and a trophy on position 1, and its matchdays with the navigator starting at its last pair (T−1, T) with no chips. Its own league leaders and team of the season SHALL be shown; recent transfers SHALL be hidden. The subtitle SHALL read "Temporada N · Clasificación final".

#### Scenario: Browse season 2
- **WHEN** a user opens `/division/7?season=2` for a 10-team league
- **THEN** the table shows season 2's final standings, the cards show matchdays 17 and 18 with final scores, the leaders are season 2's, and no transfers panel is shown

#### Scenario: Promoted in a past season
- **WHEN** the season 2 champion was promoted
- **THEN** its row carries the trophy and the promotion mark
