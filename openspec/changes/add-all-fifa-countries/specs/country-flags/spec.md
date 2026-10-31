# Spec Delta

## MODIFIED Requirements

### Requirement: One flag per country
Every code in `common.KnownCountries` SHALL map to exactly one flag, and no two codes SHALL share a
flag image. The mapping SHALL be total: adding a country to `KnownCountries` without a flag fails
the frontend tests.

#### Scenario: Home nations are distinct
- **WHEN** the flags for `en`, `sco`, `wal` and `nir` are resolved
- **THEN** they are the flags of England (St George's Cross), Scotland (Saltire), Wales (Y Ddraig Goch) and Northern Ireland (the Ulster Banner used by its football association, `flag-icons` `gb-nir`)
- **AND** none of them is the United Kingdom flag

#### Scenario: Every known country has a flag
- **WHEN** the frontend tests iterate the 211 country codes of `KnownCountries`
- **THEN** each code resolves to a flag and no two codes resolve to the same image

#### Scenario: Kosovo
- **WHEN** a player's `birthCountry` is `xk`
- **THEN** the Kosovo flag is shown, not a blank or a two-letter placeholder

#### Scenario: Tahiti
- **WHEN** a player's `birthCountry` is `pf`
- **THEN** the French Polynesia flag is shown, the flag Tahiti's FIFA association plays under
