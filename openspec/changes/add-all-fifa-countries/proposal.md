# Proposal

## Why

The game knows only 63 countries. Their name pools are small: 30 first names and 30 surnames for
the newest countries, at most 60/60 for the oldest. Spain is the exception, with a real dataset of
~29,800 first names and ~28,800 surnames. A league of 18-man squads already repeats names, and
national teams and the World Cup cannot represent most of the football world. Country lists in the
UI are also in no human order. The Leagues page sorts its tabs by country code, so in Spanish they
read "Alemania, Inglaterra, España, Francia".

## What Changes

- Add every FIFA member association that is missing: 148 new countries, 211 in total (55 UEFA,
  10 CONMEBOL, 35 CONCACAF, 46 AFC, 54 CAF, 11 OFC). Each gets everything a country needs today:
  `KnownCountries` entry, name-pool table pair, invented place names, team and stadium naming
  patterns, timezone, region, flag, and en/es name. Countries stay inactive until an admin
  activates them, as now.
- Give every country, new and existing, a long name pool: at least **300 distinct first names and
  500 distinct surnames**, romanized to Latin script and reflecting the country's real football
  population (e.g. France includes names of African and Maghrebi origin). Spain keeps its existing
  dataset, which is already far larger.
- Northern Ireland joins the home nations under the invented code `nir`, the same convention as
  `sco`/`wal`.
- Every UI list or tab strip of countries is sorted alphabetically by the country's **localized**
  name, using the UI language's collation. This covers the Leagues tabs, the onboarding country
  picker, the admin countries list and the admin seasons-by-country table. The onboarding picker
  and the admin list also switch from the backend's English `name` to the localized name.

## Capabilities

### New Capabilities
- `country-catalog`: which countries the game can generate (the FIFA membership), the
  per-country data each one must have (name pools and their minimum sizes, region, timezone,
  naming data), and how country lists are ordered in the UI.

### Modified Capabilities
- `country-flags`: the flag-coverage requirement grows from 63 to 211 codes and gains Northern
  Ireland (`nir` → `gb-nir`) as a fourth distinct home nation.

## Impact

- Backend: `common/countries.go`, `common/database.go` (`nameTableSuffixByCountry`),
  `common/teamnames.go`, `common/stadiumnames.go`, `common/timezones.go`, `common/regions.go`.
  New migrations under `resources/sql/`: one schema file plus one generated name-pool data file
  per country, about 4 MB of SQL in total.
- Frontend: `api/types.ts` (`COUNTRY_NAME_KEYS`), `components/avatar/traits.ts`
  (`COUNTRY_REGIONS`), 148 new `public/flags/*.svg`, en/es locale `countries.*` keys, a shared
  country-sorting helper, `LeaguesPage`, `SelectCountryPage`, admin `WorldTab` and `OverviewTab`.
- Startup: the name-pool files are re-applied on every start, like all migrations, but their
  guards make them a cheap no-op after the first run.
- No API shape changes, no economy changes, no new engine jobs.

## Out of scope

- Real city, club or stadium names: new countries get invented place names, like today.
- More place names for the existing 63 countries.
- Weighting draws by name frequency (draws stay uniform, as now), and foreign-born players in a
  country's squads.
- Activating any new country by default; `SEEDED_COUNTRIES` and `DefaultSeededCountries` stay
  unchanged.
- Changing the Spanish name dataset.

## Open questions for the owner

1. **Oceania region.** The game has four regions. The plan folds the 11 OFC nations into `asia`
   (Australia is already AFC), so they play the Asian Cup and pair with Asian national teams.
   The alternative is a fifth `oceania` region, which needs its own cup name, skin-tone weights
   and region type. Default: fold into `asia`.
2. **Pool size.** The default is 300 first names / 500 surnames per country. Is that "long
   enough", or should it be larger, e.g. 500/1,000? Larger pools grow the SQL linearly.
3. **Membership date.** The list is FIFA's 211 members as of 2026. Russia stays: it is
   suspended, not expelled, and already in the game.
