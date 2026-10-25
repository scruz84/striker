# country-flags Specification

## Purpose
Defines which flag the UI shows for a country code, so players and teams are always identified with
their own country's flag on every platform, including the home nations that share a state.

## Requirements

### Requirement: One flag per country
Every code in `common.KnownCountries` SHALL map to exactly one flag, and no two codes SHALL share a
flag image. The mapping SHALL be total: adding a country to `KnownCountries` without a flag fails
the frontend tests.

#### Scenario: Home nations are distinct
- **WHEN** the flags for `en`, `sco` and `wal` are resolved
- **THEN** they are the flags of England (St George's Cross), Scotland (Saltire) and Wales (Y Ddraig Goch)
- **AND** none of them is the United Kingdom flag

#### Scenario: Every known country has a flag
- **WHEN** the frontend tests iterate the 63 country codes of `KnownCountries`
- **THEN** each code resolves to a flag and no two codes resolve to the same image

#### Scenario: Kosovo
- **WHEN** a player's `birthCountry` is `xk`
- **THEN** the Kosovo flag is shown, not a blank or a two-letter placeholder

### Requirement: Platform-independent rendering
Flags SHALL be rendered as images, not emoji, so they look identical on Windows, macOS, Linux,
Android and iOS.

#### Scenario: Windows browser
- **WHEN** a manager opens a squad list on Windows
- **THEN** each player shows a flag image, never the letters of a country code

### Requirement: Player flags
A flag derived from the player's `birthCountry` SHALL appear next to a player's name on the
profile, squad list, transfer market, lineup editors, match ratings, recent league transfers,
player popover, search results, top scorers and latest evolutions.

#### Scenario: Scottish player in a squad
- **WHEN** a player with `birthCountry` `sco` appears in the squad list, market or profile
- **THEN** the Saltire is shown next to their name

#### Scenario: Player in a summary list
- **WHEN** a Scottish player appears in search results, the top scorers or the latest evolutions
- **THEN** the API row carries `birthCountry` `sco` and the Saltire is shown next to the name

### Requirement: Team flags
A club's flag, derived from the team's `country`, SHALL appear next to its name on the team page and
the dashboard's suggested live match. A national team's flag SHALL appear next to its name wherever
that name is shown: national teams page, World Cup, cup ties, team matches and player call-ups.

#### Scenario: Scottish club
- **WHEN** the team page of a club whose `country` is `sco` is opened
- **THEN** the Saltire is shown next to the club name

#### Scenario: National team in a fixture
- **WHEN** a friendly between the Scotland and Wales national teams is listed
- **THEN** each side shows its own flag next to its name, never the UK flag

#### Scenario: Ordinary club name
- **WHEN** an ordinary club's name is shown in a list that has no country for it
- **THEN** no flag is shown

### Requirement: Accessible label and unknown codes
Each flag SHALL carry the country's localized name (from `COUNTRY_NAME_KEYS`) as tooltip and
accessible label. A missing or unrecognized code SHALL render no flag and SHALL NOT raise an error.

#### Scenario: Localized tooltip
- **WHEN** the UI language is Spanish and a flag for `sco` is hovered
- **THEN** the tooltip reads "Escocia"

#### Scenario: Legacy player without birth country
- **WHEN** a player has no `birthCountry`
- **THEN** no flag is rendered and the row layout is unchanged

## Code map

- Flags: `frontend/public/flags/<code>.svg`, one per `KnownCountries` code (source `flag-icons`, MIT; see
  its README: `en`=`gb-eng`, `sco`=`gb-sct`, `wal`=`gb-wls`), rendered by
  `frontend/src/components/CountryFlag.tsx`.
- National team names: `utils/nationalTeamName.ts` (`nationalTeamCountry`, `localizedTeamName`) and
  `components/TeamNameWithFlag.tsx`.
- Player rows: `BirthCountry` on `common.ScorerStanding`, `common.PlayerSearchResult` and
  `common.TeamPlayerEvolution` (`topscorers.go`, `search.go`, `playerprogression.go`).
- Guards: `common.TestKnownCountriesHaveFlagFiles`, `common.TestSummaryRowsCarryBirthCountry`,
  `components/CountryFlag.test.tsx`.
