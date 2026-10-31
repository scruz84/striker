# Design

## Context

Per-country data is spread across seven hand-kept tables, each with its own guard test:

- `common.KnownCountries` (`countries.go`): code + English name. It is also the source of the
  national team's display name (`nationalTeamName`) and of the reverse lookup in
  `frontend/src/utils/nationalTeamName.ts`, which reads the names from `en.json`.
- `nameTableSuffixByCountry` (`database.go`): an allowlist mapping each code to its
  `names_<code>` / `lastnames_<code>` tables. `GetRandomName` builds the table name from it and
  draws uniformly (`ORDER BY random() LIMIT 1`; the `freq` column is ignored).
- `placeNamesByCountry`, `teamNamePatternsByCountry` (`teamnames.go`) and
  `stadiumNamePatternsByCountry` (`stadiumnames.go`).
- `CountryTimezone` (`timezones.go`), `countryRegions` (`regions.go`) and its TypeScript twin
  `COUNTRY_REGIONS` (`components/avatar/traits.ts`).
- `COUNTRY_NAME_KEYS` (`api/types.ts`), `countries.<code>` in `en.json`/`es.json`, and
  `public/flags/<code>.svg` (flag-icons v7.6.0, MIT).

Name pools today: `es` is a real dataset of ~29,800 / ~28,800 names. The rest are hand-written:
60/60 (`de en fr it pt`), ~52 (`br jp ng`), ~40 (most of the 2nd wave), and 30/30 for the 23
countries added by `init68`/`init69`. Those migrations guard each INSERT with
`WHERE NOT EXISTS (SELECT 1 FROM <table>)`, so they cannot add rows to a table that already has
some.

Every `initN_*.sql` is re-applied on every startup under `schemaMigrationLockKey`
(`common.EnsureSchema`). `collectSchemaFiles` also walks subdirectories other than `history/`.
Postgres's `docker-entrypoint-initdb.d` mount only runs top-level files, but `EnsureSchema`
applies the rest on the first start.

The outbound network blocks Wikidata and similar name sources, so pools cannot be fetched at
build or run time. They are written into the repository.

Countries reach the UI in backend order. The league directory is `ORDER BY country` (code), the
onboarding list and admin list follow `KnownCountries` order, and the onboarding picker and admin
list print the backend's English `name`.

## Goals / Non-Goals

**Goals:**
- One reviewable source per country's name pool, generated into idempotent SQL.
- Pools can grow again later (a "v3") without the empty-table guard problem.
- Fewer hand-kept parallel tables where one can be derived from another.
- A single frontend helper for localized, collated country ordering.

**Non-Goals:**
- Changing the API: no endpoint changes order or shape, and sorting happens in the client,
  where the UI language is known.
- Weighted name draws, or changing how `GetRandomName` queries.

## Decisions

### D1. Name lists live as text sources, and SQL is generated from them
Each country gets `utils/namepools/<code>/first.txt` and `.../last.txt`: one name per line, UTF-8,
in Latin script. `utils/gen_name_pools.py` validates and generates the SQL. Validation rejects
anything outside Latin letters, diacritics, space, `'`, `-` and `.`, rejects names longer than 40
characters, removes duplicates case-insensitively, and requires at least 300 first names and 500
surnames. The SQL is committed, so the backend needs no Python at build time.
*Alternative:* hand-written SQL like `init68`. It is unreviewable at 170k rows and offers no
validation. *Alternative:* fetching from Wikidata at build time is blocked by the network policy
and would make builds non-reproducible.

### D2. Migrations
- `resources/sql/init90_fifa_countries.sql` adds:
  - `name_pool_versions (country varchar(8) PRIMARY KEY, version int NOT NULL)`;
  - `CREATE TABLE IF NOT EXISTS` for the 148 new `names_<code>` / `lastnames_<code>` pairs,
    with the same columns as the existing ones.
- `resources/sql/namepools/init91_name_pool_<code>.sql` adds one generated file per country
  except `es`, 210 files. All share number 91 and are ordered by name (`schemaFileSortKey`).
  Each file, applied in one `Exec` (one implicit transaction), runs:

  ```sql
  INSERT INTO names_xx ("name", freq)
  SELECT v.n, 1.0 FROM (VALUES ('Ana'),('Bo')) AS v(n)
  WHERE NOT EXISTS (SELECT 1 FROM name_pool_versions WHERE country = 'xx' AND version >= 2)
    AND NOT EXISTS (SELECT 1 FROM names_xx e WHERE e."name" = v.n);
  -- same for lastnames_xx
  INSERT INTO name_pool_versions (country, version) VALUES ('xx', 2)
  ON CONFLICT (country) DO UPDATE SET version = GREATEST(name_pool_versions.version, 2);
  ```

  The version check is an uncorrelated subquery. Postgres runs it once, as a one-time filter, so
  after the first run the `VALUES` list is parsed but never scanned. The per-row `NOT EXISTS`
  keeps the old 30–60 names and adds only new ones, so existing pools grow instead of being
  replaced. A later enlargement bumps the version to 3.
  *Alternative:* `DELETE` and reinsert. It is simpler, but it throws away names the old pools
  had, for no gain.
  *Alternative:* seed from Go at startup with embedded text files. It avoids re-parsing about
  4 MB of SQL per start, but goes against the "schema/seed data is an initN migration" rule. It
  stays the fallback if task 3.10 measures the startup cost as too high (more than 2 s).

### D3. Derive the name-table allowlist from `KnownCountries`
`nameTableSuffixByCountry` becomes a map built once from `KnownCountries`. Each code must match
`^[a-z]{2,3}$`, or the build panics at init, so the SQL safety property is kept. That removes one
of seven parallel lists. The other lists carry real per-country content and stay explicit, each
behind its guard test.

### D4. Team, stadium, region and timezone data
- **Place names:** at least 16 invented, culturally plausible place names per new country,
  added to `placeNamesByCountry` and grouped by confederation with comments.
- **Naming patterns:** language-group slices (`spanishTeamPatterns`, `englishTeamPatterns`,
  `frenchTeamPatterns`, `portugueseTeamPatterns`, `arabicTeamPatterns`,
  `slavicTeamPatterns`, …) are shared by countries in the same group. Existing per-country slices
  stay as they are. Countries with a distinct football style get their own slice, e.g. `dk`/`no`
  `"%s BK"`, `"%s IF"`. Patterns never embed a real club name.
- **Stadium patterns** follow the same grouping.
- **Region:** confederation → region (`europe`, `americas`, `asia`, `africa`). OFC goes into
  `asia` (proposal open question 1). If the owner picks a fifth `oceania` region instead, it adds
  a `Region` union member, `SKIN_TONE_WEIGHTS.oceania`, `internationalCupNames["oceania"]` and its
  i18n key. It does not change any other task.
- **Timezone:** the capital's IANA zone (e.g. `fj` → `Pacific/Fiji`, `pf` → `Pacific/Tahiti`,
  `nir` → `Europe/London`). `time/tzdata` is already embedded.

### D5. Flags and names
- Copy the 148 missing 4:3 SVGs from the same flag-icons v7.6.0: `nir` ← `gb-nir`, the others
  by ISO code. Update `public/flags/README.md`.
- English names in `KnownCountries` must equal `countries.<code>` in `en.json` exactly, because
  `nationalTeamName.ts` reverse-maps them. FIFA usage: "Chinese Taipei", "Ivory Coast" (already
  used), "Curaçao", "Tahiti", "Eswatini", "Czechia", "DR Congo", "Congo". Existing names are not
  renamed (e.g. "Turkey" stays).
- Spanish names follow common Spanish press usage: "China Taipéi", "RD del Congo", "Congo",
  "Esuatini", "Chequia", "Islas Feroe", "Irlanda del Norte", "Tahití".

### D6. Frontend ordering helper
`frontend/src/utils/countryOrder.ts`:
- `compareCountryNames(a, b, t, language)` compares the localized names with a cached
  `Intl.Collator(language, { sensitivity: 'base' })`;
- `useSortedCountries(items, getCode)` is a hook that reads `t`/`i18n.resolvedLanguage` and
  returns a sorted copy, memoized on the language.

Call sites:
- `LeaguesPage`: sort the directory and keep the selected tab by **country code**, not index,
  so a language switch keeps the selection.
- `SelectCountryPage`: localized label and sorted options.
- `WorldTab` countries list: localized `name (code)` and sorted.
- `OverviewTab` seasons table: localized name next to the code, sorted.

The backend order is left alone, so API consumers and tests are unaffected.

### D7. Tests
- **Go unit tests:**
  - `KnownCountries` has 211 unique codes and names, with per-confederation counts;
  - `TestEveryKnownCountryHasNamingData` also requires 16 or more place names;
  - a new `TestNamePoolFiles` parses the embedded `namepools/*.sql` and requires one file for
    every code except `es`, at least 300 distinct first names and 500 distinct surnames per file,
    and Latin script only;
  - the existing timezone, region and flag-file tests cover the rest.
- **Go integration test:** apply the schema twice on a test DB. Pool counts are unchanged by the
  second run, and an old 30-name pool keeps its original rows.
- **Vitest:**
  - `countryOrder.test.ts`: Spanish/English orders from the spec scenarios, and the
    accent case;
  - `LeaguesPage` tab order, and selection kept across a language switch;
  - `CountryFlag.test.tsx` threshold goes from 63 to 211, and `nir` is distinct from `en`;
  - every `COUNTRY_NAME_KEYS` code has a `COUNTRY_REGIONS` entry and an en/es string.

This change adds no `TRANSACTION_*` reasons and no notification keys.

## Risks / Trade-offs

- [Model-written name lists can be implausible, contain duplicates, or form famous real
  players' full names] → The generator validates script, length and duplicates. Each
  confederation batch gets a review pass against known football squads. First and last names
  are drawn independently, as today, so full real names appear only by chance. `legend_names`
  stays the deliberate source of those.
- [About 4 MB of SQL is re-parsed on every start, while holding the schema lock] → After the
  first run, the one-time version filter skips the scans. Task 3.10 measures warm-start time and
  switches to D2's Go-seeding fallback if it adds more than 2 s.
- [The first start after deploy inserts about 170k rows while holding the schema lock, so svc
  and engine wait] → This is a one-off. Plain `INSERT … SELECT` of a few MB takes seconds.
- [The binary grows by about 4 MB from the embedded SQL] → Acceptable.
- [A 211-row admin list is long] → It is alphabetical, as requested. A filter box is a possible
  follow-up and is not in scope.
- [Hand-kept TS `COUNTRY_REGIONS` drifts from Go `countryRegions`] → A new Vitest checks that
  every code has a region. The Go test checks the backend side.

## Migration Plan

1. Deploy. On the first start, `init90` creates the tables, and each `init91_name_pool_<code>`
   fills or extends its pool and records version 2.
2. Nothing else changes until an admin activates a new country.
3. Existing players keep their names. Only newly generated players, including referees, draw
   from the larger pools.
4. Rollback: the old binary ignores the extra tables, rows and `name_pool_versions`, so no down
   migration is needed.


## Code map for the country-catalog spec

Archive does not carry a Code map out of a change delta, so this is the text to put at the end of
`openspec/specs/country-catalog/spec.md` when the change is synced and archived.

- Catalog: `backend/common/countries.go` (`KnownCountries`, `IsKnownCountry`), with the per-country
  data in `regions.go` (`countryRegions`, `RegionForCountry`), `timezones.go` (`CountryTimezone`),
  `teamnames.go` (`placeNamesByCountry`, `teamNamePatternsByCountry`, shared `*TeamPatterns` slices,
  `RandomTeamName`) and `stadiumnames.go` (`stadiumNamePatternsByCountry`, `RandomStadiumName`).
- Name pools: `common/database.go` (`GetRandomName`; `nameTableSuffixByCountry` is derived from
  `KnownCountries` and panics on a code that is not 2-3 lowercase letters),
  `resources/sql/init90_fifa_countries.sql` (`name_pool_versions` and the new `names_<code>` /
  `lastnames_<code>` tables), `resources/sql/namepools/init91_name_pool_<code>.sql` (generated, one
  per country but `es`), `utils/namepools/<code>/{first,last}.txt` (the sources),
  `utils/gen_name_pools.py` (validation and SQL generation) and `utils/build_name_pool_sources.py`
  (how the first lists were assembled, see `utils/namepools/README.md`).
- Frontend data: `api/types.ts` (`COUNTRY_NAME_KEYS`), `components/avatar/traits.ts`
  (`COUNTRY_REGIONS`), `i18n/locales/{en,es}.json` (`countries.<code>`), `public/flags/<code>.svg`.
- Alphabetical lists: `utils/countryOrder.ts` (`compareCountryNames`, `sortByCountryName`,
  `useSortedCountries`, `localizedCountryName`), used by `features/leagues/LeaguesPage.tsx`,
  `features/onboarding/SelectCountryPage.tsx`, `features/admin/WorldTab.tsx` (`CountriesSection`)
  and `features/admin/OverviewTab.tsx` (`SeasonsByCountryTable`). The API keeps its order; sorting
  happens in the client, where the UI language is known.
- Guards: `common.TestKnownCountriesAreFIFAMembership`, `TestKnownCountriesNamedExamples`,
  `TestEveryKnownCountryHasNamingData`, `TestRandomNamesUseEachCountrysOwnPlaceNames`,
  `TestNameTableSuffixRejectsUnsafeCodes`, `TestNamePoolFiles`, `TestRegionForCountry_NewMembersFollowConfederation`,
  `TestEveryCountryHasTimezone`-style tests in `timezones_test.go`, and the `STRIKER_TEST_DB_DSN`
  integration check in `schema_integration_test.go`; on the client `api/countryData.test.ts`,
  `utils/countryOrder.test.ts`, `components/CountryFlag.test.tsx`, and the page tests next to the
  four call sites.

