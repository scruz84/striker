# Tasks

## 1. Header

- [ ] 1.1 In `frontend/src/theme.ts`, set `MuiAppBar.defaultProps.color` to `'default'` and override `colorDefault` with `surface` / `onSurface`; in `frontend/src/app/Header.tsx` render the logo in `primary.main` and add a `useScrollTrigger({ disableHysteresis: true, threshold: 0 })` that switches the bar to `surfaceContainer`; verify with an extended `frontend/src/app/Header.test.tsx` covering "Page at the top" and "Content under the header", plus a case that the bar still slides away after scrolling 64 px down on a 360 px screen (there is no test for it today).

## 2. Buttons, chips and segmented buttons

- [ ] 2.1 Add the `tonal` button variant (`ButtonPropsVariantOverrides` augmentation plus a `MuiButton.variants` entry) and the 40 px pointer height; verify "Tonal button" and "Touch height" in a new `frontend/src/theme.components.test.tsx` (rendered with `lightTheme`, checked with `getComputedStyle`).
- [ ] 2.2 Change `MuiChip` to 8 px corners, 32 px / 24 px heights, and filled `colorPrimary` chips to `secondaryContainer` / `onSecondaryContainer`, keeping the 44 px `::after` hit area; verify "Compact status chip" and "Selected filter chip" in `theme.components.test.tsx`.
- [ ] 2.3 Review the 7 `color="primary"` chips (`grep -rn 'Chip' frontend/src | grep 'color="primary"'`) and confirm each is a selected or highlighted marker; record any that is not in the PR description, without changing page files.
- [ ] 2.4 Style `MuiToggleButtonGroup` / `MuiToggleButton` as segmented buttons (design decision 4); verify "Position filter in the transfer market" and "Compact mentality picker" with component tests on `marketUi.tsx`'s position filter and `MatchLineupForm`, and "No overflow at 360 px" with the mobile smoke test.

## 3. Cards, tabs, switches, progress, menus and dialogs

- [ ] 3.1 Add the `elevated` and `filled` `Card` variants (`CardPropsVariantOverrides` augmentation) and the outlined card's `outlineVariant` border; verify "Filled card in dark mode" in `theme.components.test.tsx` with `darkTheme`.
- [ ] 3.2 Override `MuiTabs`/`MuiTab`, `MuiSwitch`, `MuiLinearProgress`, `MuiMenu`/`MuiPopover` and `MuiDialog` (including `paperFullScreen`) as in design decisions 5 and 6; verify "Switch on", "Switch tap area", "Dialog in light mode" and "Full-screen dialog on a phone" in `theme.components.test.tsx`, and that `features/admin/WorldTab.test.tsx` still passes.
- [ ] 3.3 Screenshot the dashboard, squad, lineup, transfer market, league and a dialog at 360 px and 1280 px in light and dark mode (Playwright, `deviceScaleFactor: 2`) before and after, and attach them to the PR; verify the header, chips, segmented buttons and switches match the spec values.

## 4. Docs

- [ ] 4.1 Update the `buildComponents` comment in `frontend/src/theme.ts` and the Frontend section of `ARCHITECTURE.md` to list the `tonal` button and the `elevated`/`filled` card variants as the way pages ask for M3 emphasis; verify the documented prop names compile in `theme.components.test.tsx`.
- [ ] 4.2 Extend the `ui-theme` spec's Code map when archiving (`theme.ts` `buildComponents`, `app/Header.tsx`, `theme.components.test.tsx`); no locale strings change.

## 5. Checks

- [ ] 5.1 Run `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run && npm run build` and confirm all pass; there is no backend change, so the backend `go build ./... && go vet ./... && go test ./...` run only confirms nothing else broke.
- [ ] 5.2 Run the mobile smoke test (`npm run smoke:mobile` against a seeded backend) and confirm no new overflow, font or tap-target finding.

## Workflow follow-up

- Implement and archive after `add-m3-color-roles`.
