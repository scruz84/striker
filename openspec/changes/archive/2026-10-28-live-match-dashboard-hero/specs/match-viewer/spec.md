# Spec Delta

## MODIFIED Requirements

### Requirement: Scoreboard subtitle
The scoreboard subtitle SHALL read `<competition label> · <kickoff date and time> · <weather emoji>`. Any part that is unknown is left out. The labels are:
- League match: "<division name> · Jornada N de T" (just "Jornada N" while the division details are still loading).
- Cup tie: "<cup name> · <round name>".
- World Cup group match: "Mundial · Grupo X · Jornada N".
- Any other friendly: "Amistoso".

#### Scenario: Cup quarter-final in the rain
- **WHEN** a played quarter-final of "Copa del Rey" had rain
- **THEN** the subtitle reads "Copa del Rey · Cuartos de final · 21/10/2026 20:00 · 🌧️"

#### Scenario: Old friendly without stored weather
- **WHEN** a friendly played before this change is opened
- **THEN** the subtitle reads "Amistoso · <date>" with no weather emoji

#### Scenario: World Cup group fixture
- **WHEN** the second-matchday match of World Cup group B is opened
- **THEN** the subtitle reads "Mundial · Grupo B · Jornada 2 · <date>"

#### Scenario: Live league match names its division
- **WHEN** a viewer opens a live matchday-7 match of division "Segunda.2", which has 20 teams (38 matchdays), on a sunny day
- **THEN** the subtitle reads "Segunda.2 · Jornada 7 de 38 · <date> · ☀️", the same in the scheduled, live and finished views
