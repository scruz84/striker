# Spec Delta

## Purpose

Defines which countries the game can generate a world for (every FIFA member association), the
data each country must carry to generate players, teams and national sides, and the order in
which country names are listed in the UI.

## ADDED Requirements

### Requirement: FIFA membership catalog
The set of known countries SHALL be exactly the 211 FIFA member associations: 55 UEFA, 10 CONMEBOL, 35 CONCACAF, 46 AFC, 54 CAF and 11 OFC. Each SHALL have one unique lowercase code: its ISO 3166-1 alpha-2 code, except `en`, `sco`, `wal`, `nir` (home nations) and `xk` (Kosovo). A known country SHALL stay inactive until an admin activates it.

#### Scenario: Catalog size
- **WHEN** the known countries are listed
- **THEN** there are 211 entries and no two share a code or an English name

#### Scenario: Home nations
- **WHEN** the codes `en`, `sco`, `wal` and `nir` are looked up
- **THEN** they are England, Scotland, Wales and Northern Ireland, four separate countries

#### Scenario: Territories that are FIFA members
- **WHEN** the codes `fo`, `gi`, `hk`, `mo`, `tw`, `ps`, `pr`, `cw`, `aw`, `pf` and `nc` are looked up
- **THEN** they are Faroe Islands, Gibraltar, Hong Kong, Macau, Chinese Taipei, Palestine, Puerto Rico, Curaçao, Aruba, Tahiti and New Caledonia

#### Scenario: New country is inactive
- **WHEN** the game is initialized with the default seeded countries
- **THEN** only `es`, `en`, `fr` and `de` are active, and Fiji (`fj`) is listed in the admin countries list as not active

### Requirement: Long name pools
Every known country SHALL have its own pool of at least 300 distinct first names and at least 500 distinct surnames, written in Latin script. Each pool SHALL reflect the names common among that country's footballers, including its main communities of foreign origin. Spain's existing pool SHALL be kept unchanged. A player born in a country SHALL draw both names from that country's pool.

#### Scenario: New country pool size
- **WHEN** the name pool of Iceland (`is`) is counted
- **THEN** it has at least 300 distinct first names and at least 500 distinct surnames

#### Scenario: Existing country pool grows
- **WHEN** the name pool of Russia (`ru`), which had 30 first names and 30 surnames, is counted after the change
- **THEN** it has at least 300 distinct first names and at least 500 distinct surnames, and the original 30 + 30 are still present

#### Scenario: Non-Latin scripts are romanized
- **WHEN** a player is generated for Japan, South Korea, Greece or Saudi Arabia
- **THEN** both names are written in Latin letters (diacritics allowed), never in kana, hangul, Greek or Arabic script

#### Scenario: Name variety in a squad
- **WHEN** an 18-player squad is generated for a country other than Spain
- **THEN** each name is drawn uniformly from the pool, so the chance that two given players share a surname is at most 1 in 500

#### Scenario: Unknown code
- **WHEN** a player is generated for a code that is not a known country
- **THEN** the Spanish pool is used, as today

### Requirement: Complete country data
Every known country SHALL have: a region (`europe`, `americas`, `asia` or `africa`), a valid IANA timezone, at least 16 invented place names, at least one team-name pattern, at least one stadium-name pattern, a flag, and an English and Spanish name. A country missing any of these SHALL fail the backend or frontend tests.

#### Scenario: Region follows the confederation
- **WHEN** the regions of the new countries are looked up
- **THEN** UEFA members are `europe`, CONMEBOL and CONCACAF members are `americas`, AFC and OFC members are `asia`, and CAF members are `africa`

#### Scenario: Pacific nation
- **WHEN** New Zealand (`nz`) is active and the international cup starts
- **THEN** its tier-1 champion plays in the Asian Cup, and its national team prefers Asian opponents for friendlies

#### Scenario: Activating a new country
- **WHEN** an admin activates Jamaica (`jm`)
- **THEN** its leagues, teams, players, stadiums and national team are created with Jamaican names and the `America/Jamaica` timezone, with no error and no fallback to another country's data

#### Scenario: Missing data fails the build
- **WHEN** a code is added to the known countries without a timezone, region, name pool, place names, flag or locale name
- **THEN** a backend or frontend test fails, naming the code

### Requirement: Alphabetical country lists
Every UI list, menu or tab strip of countries SHALL be sorted by the country's name in the current UI language, using that language's collation (accent- and case-insensitive). It SHALL show that localized name. This covers the Leagues page tabs, the onboarding country picker, the admin countries list and the admin seasons-by-country table.

#### Scenario: Leagues tabs in Spanish
- **WHEN** the active countries are `de`, `en`, `es` and `fr` and the UI is in Spanish
- **THEN** the Leagues page tabs read "Alemania, España, Francia, Inglaterra"

#### Scenario: Leagues tabs in English
- **WHEN** the same countries are active and the UI is in English
- **THEN** the tabs read "England, France, Germany, Spain"

#### Scenario: Accents do not push a name to the end
- **WHEN** Austria, Azerbaijan and Belgium are listed in Spanish
- **THEN** the order is "Austria, Azerbaiyán, Bélgica"

#### Scenario: Onboarding picker
- **WHEN** a new user with the UI in Spanish opens the country picker
- **THEN** the options show Spanish names ("Inglaterra", not "England") in alphabetical order

#### Scenario: Admin countries list
- **WHEN** an admin opens the world tab's countries list
- **THEN** all 211 countries are listed by localized name in alphabetical order, each with its code and its active state

#### Scenario: Language switch
- **WHEN** the user switches the UI language while a sorted country list is on screen
- **THEN** the list re-sorts by the names in the new language, and the Leagues page keeps the same country selected

### Requirement: Country-typical avatars
A player's generated avatar SHALL depend on their birth country, not only on its continent: every known country belongs to an appearance group whose odds shape skin tone, hair colour, hair style and facial hair, while every option stays reachable. A missing or unknown birth country SHALL fall back to central Europe's group.

#### Scenario: Neighbours on one continent differ
- **WHEN** many avatars are generated for Fiji (`fj`) and for Japan (`jp`)
- **THEN** Fiji's skew to dark skin tones and curly hair, Japan's to lighter tones and straight black hair, although both are in the `asia` region for the international cup

#### Scenario: New country has a group
- **WHEN** the frontend tests iterate the 211 known country codes
- **THEN** each resolves to an appearance group and no code relies on the fallback by accident

#### Scenario: Stable face
- **WHEN** the same player seed and birth country are used twice
- **THEN** the same face is produced

