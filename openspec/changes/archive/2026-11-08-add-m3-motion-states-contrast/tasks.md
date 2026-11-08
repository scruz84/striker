# Tasks

## 1. Theme

- [x] 1.1 Add `M3_TRANSITIONS`, the palette `action` state layers, the reduced-motion `CssBaseline` rule and `buildTheme(mode, highContrast)` with `contrastLevel` in `buildScheme` to `frontend/src/theme.ts`, exporting `highContrastLightTheme` / `highContrastDarkTheme` next to the existing themes (design decisions 1–4); verify with `npx tsc -b`.
- [x] 1.2 Pick the high-contrast theme in `ThemeModeProvider` from `prefers-contrast: more`; verify "Increase contrast in light mode" and "No preference" in a new `frontend/src/app/ThemeModeProvider.test.tsx` with `matchMedia` stubbed.

## 2. Tests

- [x] 2.1 Extend `frontend/src/theme.test.ts`: "Dialog opens" (easings and durations), "Hovering a list item" (`action.hover`), the reduced-motion rule in `MuiCssBaseline`, "Increase contrast in light / dark mode" (pinned primaries and `onSurfaceVariant`, 7:1 for every pair, 4.5:1 for the outline), and that the standard themes are unchanged.

## 3. Checks

- [x] 3.1 Against a local seeded backend, screenshot the dashboard at 360 px with Playwright's `forcedColors`/`contrast: 'more'` emulation in light and dark mode, and check with `reducedMotion: 'reduce'` that the navigation drawer opens without a transition (computed `transition-duration` of its paper is `1e-05s`); run the mobile smoke test with no new finding.
- [x] 3.2 Extend the `ui-theme` spec's Code map when archiving (`buildTheme`, `M3_TRANSITIONS`, `ThemeModeProvider`), and run `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run && npm run build` and the backend `go build ./... && go vet ./... && go test ./...`; all pass.
