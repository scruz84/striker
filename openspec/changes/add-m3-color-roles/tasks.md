# Tasks

## 1. Colour roles

- [ ] 1.1 In `frontend/src/theme.ts`, replace `themeFromSourceColor` with a `buildScheme(isDark)` that returns a `DynamicScheme` (variant `TONAL_SPOT`, contrast 0) over the legacy core palettes (design decision 1), and make `buildPalette` read roles from it; verify with `npx tsc --noEmit` and that `palette.primary.main` is still `#006e1c` / `#78dc77`.
- [ ] 1.2 Add the `surfaceDim`, `surfaceBright`, `surfaceContainerLowest` … `surfaceContainerHighest` and `onSurface` fields to the `Palette` / `PaletteOptions` augmentation and to `buildPalette`; map `surfaceVariant` to `surfaceContainerHighest` and `background.default`/`background.paper` to `surface`; verify with `npx tsc --noEmit`.
- [ ] 1.3 Change the `MuiPaper.elevation1` override to `backgroundColor: theme.palette.surfaceContainerLow` (no shadow) and keep `mixHex` only for `buildRatingScale`; verify that `palette.rating` is unchanged by comparing it before and after in the new test.

## 2. Tests

- [ ] 2.1 Add `frontend/src/theme.test.ts` with a local `contrastRatio` helper and the cases of the `ui-theme` spec: "Brand colour unchanged" (both modes), "Surface container levels" (both modes), "Text on a container", "Secondary text on the highest surface", "Outline on the page", plus a loop asserting ≥ 4.5:1 for every `on*` pair and every surface level in both modes; verify with `npx vitest run src/theme.test.ts`.
- [ ] 2.2 Check the surfaces visually: run `npm run dev` and screenshot the dashboard, squad and league pages at 360 px and 1280 px in light and dark mode (Playwright, `deviceScaleFactor: 2`), and attach them to the PR; verify that raised panels show `#f3f4ee` / `#1a1c19` and outlined panels keep the page colour ("Raised panel", "Outlined panel" scenarios).

## 3. Docs

- [ ] 3.1 Update the `frontend/src/theme.ts` header comment and the Frontend section of `ARCHITECTURE.md` (it says "Material 3, single seed color") to name `DynamicScheme`, the surface-container roles and the pinned-colour test; verify the paths it names exist.
- [ ] 3.2 Give the new `ui-theme` spec a "Code map" section when this change is archived (`frontend/src/theme.ts`: `buildScheme`, `buildPalette`, `buildRatingScale`; `frontend/src/theme.test.ts`); no locale strings change.

## 4. Checks

- [ ] 4.1 Run `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run && npm run build` and confirm all pass; there is no backend change, so the backend `go build ./... && go vet ./... && go test ./...` run only confirms nothing else broke.
- [ ] 4.2 Run the mobile smoke test (`npm run smoke:mobile` against a seeded backend) and confirm no new overflow, font or tap-target finding.

## Workflow follow-up

- Archive `add-m3-color-roles` after merge, before archiving `complete-m3-component-theme`, so the `ui-theme` spec is created by this change.
