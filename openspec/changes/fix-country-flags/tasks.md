# Tasks

## 1. Audit and assets

- [x] 1.1 Re-run the audit: compare the 63 codes of `backend/common/countries.go` `KnownCountries` with `COUNTRY_NAME_KEYS` and `countryFlag.ts`; list every code whose flag is not its own (expected: `en`, `sco`, `wal`).
- [x] 1.2 Vendor the 63 SVGs from `flag-icons` 7.6.0 into `frontend/public/flags/<code>.svg` (with its LICENSE and a README); confirm `gb-eng`, `gb-sct`, `gb-wls` and `xk` exist.
- [x] 1.3 Check the size of the 63 SVGs; svgo cannot shrink the heavy coats-of-arms flags (es, mx, rs, bo, me, hr, ec: 29-182 KB), so they stay as is, served as lazy static files.

## 2. Component

- [x] 2.1 Create `frontend/src/components/CountryFlag.tsx` with the code → asset map (`en`→`gb-eng`, `sco`→`gb-sct`, `wal`→`gb-wls`, others by alpha-2) and the `<img>` rendering from design.md; unknown or missing code renders nothing.
- [x] 2.2 Localized name as `alt` and MUI `Tooltip` text via `COUNTRY_NAME_KEYS`.
- [x] 2.3 Add `nationalTeamCountry()` to `utils/nationalTeamName.ts` and a `TeamNameWithFlag` component (flag only for national teams).

## 3. Existing call sites

- [x] 3.1 `PlayerProfileRail.tsx`, `SquadList.tsx`, `transfers/marketUi.tsx` (`PlayerCell`): replace the emoji.
- [x] 3.2 `TeamDetailsPage.tsx` header and `DashboardPage.tsx` (`SuggestedLiveMatch`): replace the emoji.
- [x] 3.3 Delete `utils/countryFlag.ts` and confirm nothing else imports it.

## 4. New call sites, frontend only

- [x] 4.1 National teams: `NationalTeamsPage`, `CupPage`, `CupTieDetailsPage`, `TeamMatchesSection`, `PlayerDetailsPage` call-ups use `TeamNameWithFlag`.
- [x] 4.2 Players with `birthCountry` already loaded: `PlayerChip`, `SelectedPlayerPanel`, `SubstitutionsEditor`, `TeamRatingsColumn`, `PlayerStatsPopover`, `TransferMarketPage`, `RecentLeagueTransfers`.

## 5. Backend `birthCountry` for summary rows

- [x] 5.1 `ScorerStanding`: field + `p.birth_country` in `GetPlayerTeamForScorerStmtSql`; dashboard leader and division top scorers show the flag.
- [x] 5.2 `PlayerSearchResult`: field + `SearchPlayersStmtSql`; `SearchPage` shows the flag.
- [x] 5.3 `TeamPlayerEvolution`: field + `GetLatestPlayerEvolutionsForTeamStmtSql`; `LatestEvolutionsPage` shows the flag.
- [x] 5.4 Frontend types gain the optional `birthCountry`; a Go integration test asserts the field is filled (`TestSummaryRowsCarryBirthCountry`), and `TestKnownCountriesHaveFlagFiles` guards the flag files.

## 6. Tests and docs

- [x] 6.1 Vitest: every `KnownCountries` code has a flag, no two codes share an asset, `en`/`sco`/`wal` pairwise different.
- [x] 6.2 Component tests: localized `alt`/tooltip, nothing for an undefined or unknown code, `TeamNameWithFlag` shows a flag for "Scotland" and none for a club name.
- [ ] 6.3 Add the `country-flags` Code map to the main spec on archive (flags dir, `CountryFlag.tsx`, `TeamNameWithFlag.tsx`, `nationalTeamCountry`, the three `birthCountry` fields and the guard tests) and mention flags in `ARCHITECTURE.md`'s components line.
- [ ] 6.4 Check all 63 flags render (contact sheet screenshot done); the in-app light/dark and Spanish-UI pass is left to the owner's review of the running app.

## 7. Final checks

- [ ] 7.1 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`
- [ ] 7.2 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`
