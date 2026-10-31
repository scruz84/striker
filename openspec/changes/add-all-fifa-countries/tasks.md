# Tasks

## 1. Country catalog (backend)

- [ ] 1.1 Extend `common.KnownCountries` to the 211 FIFA members (148 new, including `nir`), grouped by confederation with comments, using the English names from design D5; add `TestKnownCountriesAreFIFAMembership` (211 unique codes, unique names, 55/10/35/46/54/11 per confederation) and verify `go test ./common -run KnownCountries` passes
- [ ] 1.2 Replace the hand-kept `nameTableSuffixByCountry` with a map derived from `KnownCountries` at init that panics on a code not matching `^[a-z]{2,3}$` (design D3); verify `TestEveryKnownCountryHasNamingData` and a new unit test for the code check pass
- [ ] 1.3 Add every new code to `countryRegions` in `common/regions.go` by confederation (OFC → `asia`) and verify `go test ./common -run Region` passes
- [ ] 1.4 Add every new code's capital IANA zone to `CountryTimezone` and verify `go test ./common -run Timezone` passes (every zone loads)
- [ ] 1.5 Update the doc comments in `countries.go`, the "adding a country" notes in `ARCHITECTURE.md` and `backend/README.md` (name pools now come from `utils/namepools/` via the generator), and verify that the files they name exist

## 2. Team and stadium naming data

- [ ] 2.1 Add language-group team-name pattern slices in `teamnames.go` (design D4) and give every new country a `teamNamePatternsByCountry` entry, with no real club names; verify the naming-data test passes
- [ ] 2.2 Add language-group stadium pattern slices and a `stadiumNamePatternsByCountry` entry for every new country; verify the naming-data test passes
- [ ] 2.3 Add at least 16 invented place names per new country to `placeNamesByCountry`, extend `TestEveryKnownCountryHasNamingData` to require 16 or more for new codes, and verify a quick `RandomTeamName`/`RandomStadiumName` loop over every code produces no fallback to `en`

## 3. Name pools

- [ ] 3.1 Write `utils/gen_name_pools.py` (design D1): it reads `utils/namepools/<code>/{first,last}.txt`, validates Latin script, length ≤ 40 and case-insensitive duplicates, enforces ≥300 first and ≥500 last names, and writes `backend/resources/sql/namepools/init91_name_pool_<code>.sql` in the D2 format; verify it on a two-country sample and that it exits non-zero on a short or non-Latin list
- [ ] 3.2 Add `backend/resources/sql/init90_fifa_countries.sql` (`name_pool_versions` plus `CREATE TABLE IF NOT EXISTS` for the 148 new name-table pairs); verify `TestCollectSchemaFiles_*` and `TestSchemaFileSortKey_*` still pass and that the `namepools/` files sort after `init90`
- [ ] 3.3 Write the UEFA name lists (55 countries; for existing ones, start from the current pool so its names are kept) and verify the generator accepts every UEFA code
- [ ] 3.4 Write the CONMEBOL and CONCACAF name lists (45 countries) and verify the generator accepts every code
- [ ] 3.5 Write the AFC name lists (46 countries, romanized) and verify the generator accepts every code
- [ ] 3.6 Write the CAF name lists (54 countries) and verify the generator accepts every code
- [ ] 3.7 Write the OFC name lists (11 countries) and verify the generator accepts every code
- [ ] 3.8 Generate all 210 SQL files and add `TestNamePoolFiles` (one file per known code except `es`, ≥300/≥500 distinct, Latin script only, parsed from the embedded FS); verify `go test ./common -run NamePool` passes
- [ ] 3.9 Add a schema integration test that applies the schema twice and checks that pool counts are unchanged by the second run, that an old 30-name pool (`ru`) keeps its original rows and reaches ≥300/≥500, and that `name_pool_versions` holds version 2 for 210 countries; verify it passes
- [ ] 3.10 Measure a warm restart (`EnsureSchema` on an already-migrated DB) before and after; verify it adds ≤ 2 s, or switch to the Go-seeding fallback from design D2 and re-measure

## 4. Frontend country data

- [ ] 4.1 Add the 148 codes to `COUNTRY_NAME_KEYS` and `countries.<code>` to `en.json` (equal to the `KnownCountries` names) and `es.json` (design D5); verify `nationalTeamName` resolves "Northern Ireland" → `nir` in a unit test
- [ ] 4.2 Copy the 148 flag-icons v7.6.0 4:3 SVGs into `public/flags/` (`nir` ← `gb-nir`), update `public/flags/README.md`, raise the `CountryFlag.test.tsx` threshold to 211 and add the `nir` home-nation case; verify `npx vitest run src/components/CountryFlag.test.tsx` and Go `TestKnownCountriesHaveFlagFiles` pass
- [ ] 4.3 Add every new code to `COUNTRY_REGIONS` in `components/avatar/traits.ts` and add a Vitest that every `COUNTRY_NAME_KEYS` code has a region and en/es strings; verify it passes

## 5. Alphabetical country lists

- [ ] 5.1 Add `utils/countryOrder.ts` (`compareCountryNames`, `useSortedCountries`, with a cached `Intl.Collator`, `sensitivity: 'base'`) and `countryOrder.test.ts` covering the spec's Spanish, English and accent scenarios; verify it passes
- [ ] 5.2 Sort the `LeaguesPage` tabs with the helper and track the selected tab by country code; add a test for the tab order in es/en and for keeping the selection across a language switch; verify it passes
- [ ] 5.3 Make `SelectCountryPage` show localized, sorted options; add a test for Spanish labels in alphabetical order; verify it passes
- [ ] 5.4 Make the admin `WorldTab` countries list localized (`name (code)`) and sorted, and sort the `OverviewTab` seasons-by-country table by localized name showing name and code (phone cards and desktop table); verify in tests or by rendering the admin page at 360 px and desktop widths
- [ ] 5.5 Check that no other screen lists countries (`grep` for `COUNTRY_NAME_KEYS`, `.country` maps and `useOnboardingCountries`/`useAdminCountries`/`useLeagueDirectory`) and record the result in the PR description

## 6. Specs

- [ ] 6.1 Fill in the Code map for the new `country-catalog` spec (countries.go, regions.go, timezones.go, teamnames.go, stadiumnames.go, database.go `GetRandomName`, `utils/gen_name_pools.py`, `resources/sql/init90_fifa_countries.sql`, `resources/sql/namepools/`, `utils/countryOrder.ts` and its call sites, and the guard tests); update the `country-flags` Code map with the `nir` → `gb-nir` mapping; verify `openspec validate add-all-fifa-countries --strict` passes

## 7. Integration checks

- [ ] 7.1 In a local stack, activate Jamaica (`jm`) and Fiji (`fj`) from the admin UI; verify the teams, players, stadiums and national team have local names, the right kickoff timezone and the right flag, and that a Fijian national team pairs with Asian opponents
- [ ] 7.2 Backend: `cd backend && go build ./... && go vet ./... && go test ./...` passes
- [ ] 7.3 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run` passes

## Workflow follow-up

- Sync the delta specs into `openspec/specs/` and archive the change once the PR is merged.
