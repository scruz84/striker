# Tasks

## 1. Shared destinations and mode

- [x] 1.1 Add `NavMode` and `useNavMode()` to `frontend/src/app/navigation.ts` (design decision 1); verify with a test in `navigation.test.ts` that stubs `matchMedia` for 360, 1024 and 1280 px and gets `bar`, `rail` and `drawer`.
- [x] 1.2 Add `frontend/src/app/navItems.tsx` with `useNavGroups()`, `usePrimaryDestinations()` and `navIcon` (outlined / filled pairs, admin gating, `alsoMatches`), moving the list out of `NavDrawer.tsx`; verify with `npx tsc --noEmit` and the drawer tests of task 3.2.

## 2. Navigation bar

- [x] 2.1 Add the `MuiBottomNavigation` / `MuiBottomNavigationAction` overrides to `frontend/src/theme.ts`, set `BOTTOM_NAV_HEIGHT = 80`, and rewrite `BottomNav.tsx` on `usePrimaryDestinations()` (flat `Paper`, no per-action `sx`); verify "Current entry" and "Icon swaps on selection" in a new `frontend/src/app/BottomNav.test.tsx` (computed pill size and colour, filled/outlined icon test ids), and update the `BOTTOM_NAV_HEIGHT` mock in `Layout.test.tsx`.

## 3. Rail and drawers

- [x] 3.1 Add `frontend/src/app/NavRail.tsx` (design decision 4); verify "Rail entry selected" and that the menu button calls `onMenu` and exposes `aria-expanded`, in a new `frontend/src/app/NavRail.test.tsx`.
- [x] 3.2 Rewrite `NavDrawer.tsx` as `variant: 'modal' | 'standard'` with M3 pill items, the 256 px width, the modal `surfaceContainerLow` paper with 16 px corners and the sticky standard column (design decision 5); verify "Current item" (`/cup/ties/7` selects Cup), "Admin entry" and that choosing an item calls `onClose`, in a new `frontend/src/app/NavDrawer.test.tsx`.
- [x] 3.3 Wire `useNavMode()` into `Layout.tsx` (design decision 6); verify "Phone", "Laptop" and "Desktop" in `Layout.test.tsx` (with `matchMedia` stubbed and the three components mocked to markers), and that a club-less user gets no navigation.

## 4. Real app checks

- [x] 4.1 Against a local seeded backend, screenshot the dashboard at 360, 1024 and 1280 px in light and dark mode, with the modal drawer open at 360 and 1024 px, before and after; verify "Rail stays put" and "Header stays readable" by scrolling 600 px and 400 px and checking that the rail is in view and the logo's bounding box is not covered (`elementFromPoint` at its centre returns the logo).
- [x] 4.2 Run the mobile smoke test (`npm run smoke:mobile`) and confirm "Labels fit" and "Last content not hidden" add no new finding over `main`.

## 5. Docs and checks

- [x] 5.1 Update `ARCHITECTURE.md`'s `app/` line (navigation: bar / rail / drawer, `navItems.tsx`, `useNavMode`) and give the `app-navigation` spec a Code map when archiving; verify the paths exist. No locale strings change.
- [x] 5.2 Run `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run && npm run build`, and the backend `go build ./... && go vet ./... && go test ./...` to confirm nothing else broke; all pass.
