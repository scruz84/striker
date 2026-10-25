# Design

## Context

`countryFlag(code)` in `frontend/src/utils/countryFlag.ts` returns an emoji string and is called by
five components: `PlayerProfileRail`, `SquadList`, `marketUi` (`PlayerCell`), `TeamDetailsPage` and
`DashboardPage` (`SuggestedLiveMatch`). Its map covers all 63 `KnownCountries` codes, but `en`,
`sco` and `wal` all return 🇬🇧. Country names already come from `COUNTRY_NAME_KEYS` (en/es locale
files), which each call site uses as the tooltip. The service worker has no `fetch` handler, so
static assets are served by the browser/HTTP cache only. See proposal.md - Why.

## Goals / Non-Goals

**Goals:**
- One correct flag per code, identical on every OS, behind a single component.
- A test that ties the flag set to `KnownCountries`.

**Non-Goals:**
- No locale change, no SQL migration, no change to how `birthCountry` is generated.

## Decisions

**SVG flags instead of emoji.** Scotland and Wales have no regional-indicator emoji, and Windows
renders none at all, so no emoji mapping can fix the bug. Alternatives: keep emoji and add tag
sequences for the home nations (still blank on Windows and older Android - rejected); a web font
that draws flag emoji, e.g. Twemoji (adds a font download and still needs per-platform CSS - rejected);
SVG files (chosen).

**Source of the SVGs: vendored from `flag-icons` (MIT) into `frontend/public/flags/<code>.svg`.**
Only the 63 files we need, named by the game's own code, so no mapping table is needed at runtime
and the component just builds `/flags/<code>.svg`. Static files in `public/` are not inlined into
the JS bundle (Vite inlines small `?url` assets as data URIs) and are fetched lazily. Alternatives:
the `flag-icons` package with its CSS (emits all ~270 flags, extra dependency) and importing via
`import.meta.glob` (inlines the small ones) - both rejected. svgo cannot shrink the heavy
coats-of-arms flags (es, mx, rs, bo, me: 55-180 KB), so they ship as they are.

**Code mapping.** `en` → `gb-eng`, `sco` → `gb-sct`, `wal` → `gb-wls`, `xk` → `xk`, every other
`KnownCountries` code → the ISO 3166-1 alpha-2 file of the same name (`in`, `id`, `ie`, `ci`, ...).
Adding a country is a new file in `public/flags/`; `common.TestKnownCountriesHaveFlagFiles` fails until it exists.

**One component.** `<CountryFlag code size />` renders an `<img>` (`loading="lazy"`,
`decoding="async"`, fixed 4:3 box, 1px subtle outline so white-dominated flags like Japan stay
visible on light cards) with `alt` and tooltip set to the localized country name. Unknown/missing
code → `null`. Call sites keep their current spacing and size (1rem inline, 1.5rem on the team
header) by passing `height`. The old `countryFlag()` helper is deleted so nothing keeps using emoji.

**Guard tests.** `CountryFlag.test.tsx` asserts every `COUNTRY_NAME_KEYS` code has a flag file,
no two codes share an image, and `en`/`sco`/`wal` differ. Vite cannot import the backend's Go file,
so the drift against `KnownCountries` is caught on the Go side by `TestKnownCountriesHaveFlagFiles`.

**New call sites.** A national team's name is its country's English name, so
`utils/nationalTeamName.ts` gains `nationalTeamCountry(rawName)` (same `CODE_BY_ENGLISH_NAME`
lookup as `localizedTeamName`) and a `TeamNameWithFlag` helper component renders flag + localized
name only when the name is a national team (otherwise the plain name). It replaces the bare
`localizedTeamName(...)` in JSX at: `NationalTeamsPage` (fixtures, World Cup, standings, champion),
`CupPage` ties, `CupTieDetailsPage`, `TeamMatchesSection` and `PlayerDetailsPage` call-ups. String-
interpolated uses (`t('cup.byeRests', {team})`, `pensWinner`) stay text-only. Player flags go next
to the name in `PlayerChip`, `SelectedPlayerPanel`, `SubstitutionsEditor`, `TeamRatingsColumn`,
`PlayerStatsPopover`, `TransferMarketPage`, `RecentLeagueTransfers` (all already hold a `Player`
with `birthCountry`).

**Backend: additive `birthCountry`.** `ScorerStanding`, `PlayerSearchResult` and
`TeamPlayerEvolution` get `BirthCountry string json:"birthCountry,omitempty"`, filled by adding
`p.birth_country` to `GetPlayerTeamForScorerStmtSql`, `SearchPlayersStmtSql` and
`GetLatestPlayerEvolutionsForTeamStmtSql`. No schema change (the column exists); the frontend types
gain the optional field. Cached payloads without the field simply show no flag until refreshed.

## Risks / Trade-offs

- [63 extra image requests on first load of a busy page] → files are tiny, lazy, content-hashed and
  cached immutably; a squad list uses a handful of distinct flags.
- [A few flags are heavy SVGs (coats of arms: Spain, Mexico, Serbia...)] → measure in the task
  list; if any exceeds ~25 KB, use the package's simplified variant or optimise with svgo.
- [Visual change on all five screens] → keep sizes identical and check both themes.
- [Tooltip/screen-reader text regress] → alt text asserted in a component test.

## Migration Plan

Frontend-only; ships with a normal frontend deploy. Rollback = revert the commit (no data or
schema involved).
