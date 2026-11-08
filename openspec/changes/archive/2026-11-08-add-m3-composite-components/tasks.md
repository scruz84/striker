# Tasks

## 1. Search bar

- [x] 1.1 Add `frontend/src/components/m3/SearchBar.tsx` (design decision 2) and the `common.clearSearch` strings in `i18n/locales/en.json` ("Clear search") and `es.json` ("Borrar búsqueda"); verify "Search page" (56 px, `#e8e9e2`, no clear button), "Clearing a search" (empty value, focus back on the input, button gone) and "Compact filter" (48 px) in a new `components/m3/SearchBar.test.tsx`.
- [x] 1.2 Use `SearchBar` in `features/search/SearchPage.tsx` (autofocused, 56 px), `features/nationalteams/NationalMatchFilters.tsx` and `features/admin/UsersTab.tsx` (compact); verify their existing tests still pass (the fields keep their accessible names) and grep finds no other search `TextField` with a search icon adornment.

## 2. Bottom sheet

- [x] 2.1 Add `frontend/src/components/m3/BottomSheet.tsx` (design decision 3) and rebuild `FilterSheet` in `features/transfers/marketUi.tsx` on it, keeping its props; verify "Market filters" in a new `components/m3/BottomSheet.test.tsx` (paper colour, 28 px top corners, the handle, title and footer) and that `TransferMarketPage` tests still pass.

## 3. Stat tiles

- [x] 3.1 Add `frontend/src/components/m3/StatTile.tsx` (design decision 4) and use it for the dashboard KPI tiles with outlined icons; verify "Dashboard KPIs" (filled `#e2e3dd` tiles, each one link to its page) and "Negative budget" (error colour) in a new `components/m3/StatTile.test.tsx`, and that `DashboardPage` tests still pass.

## 4. Real app checks

- [x] 4.1 Against a local seeded backend, screenshot the search page, the national teams page, the market with the filter sheet open and the dashboard at 360 px and 1280 px in light and dark mode, before and after, and run the mobile smoke test; verify no new finding over `main`.

## 5. Docs and checks

- [x] 5.1 Extend the `ui-theme` spec's Code map with `components/m3/` when archiving, and mention `components/m3/` in `ARCHITECTURE.md`'s `components/` line; verify the paths exist.
- [x] 5.2 Run `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run && npm run build`, and the backend `go build ./... && go vet ./... && go test ./...`; all pass.
