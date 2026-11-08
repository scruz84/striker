# Proposal

## Why

Material 3 uses one icon style throughout: outlined glyphs, filled only for a selected state.
After phases 0–3 (`add-m3-color-roles`, `complete-m3-component-theme`, `add-adaptive-navigation`,
`add-m3-composite-components`), navigation, the dashboard KPIs and the search fields follow that
rule. The pages still mix in MUI's two-tone icons:
- 20 different glyphs, in 14 files;
- among them the dashboard's card headers, the cup and league pages, the national teams tabs,
  friendlies, help and search results.

Two-tone glyphs read as a different, older icon set next to the outlined ones. This is phase 4
of the plan: the page-level part of the migration, which comes down to the icon set now that
phases 1 and 3 restyled every surface and component from the theme.

## What Changes

- Every `@mui/icons-material/*TwoTone` import in the app becomes the `*Outlined` glyph of the same
  name. No glyph changes meaning, colour or size.
- A unit test fails the build if a two-tone icon is imported again, so the rule holds without
  review.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ui-theme`: adds the icon style requirement (outlined glyphs; filled only where a selected
  state calls for it, as `app-navigation` already states for navigation entries).

## Impact

- Frontend only: imports and JSX names in the 14 files listed in design.md, and a new
  `frontend/src/iconStyle.test.ts`.
- No string, backend, API or migration change.

## Out of scope

- Icons that are already filled on purpose (status glyphs such as the warning and info icons in
  `Alert`, the check of a selected chip, the filled navigation entry of the current page).
- Replacing `Paper` with `Card` in pages. Both draw the same M3 surfaces since phase 1, so the swap
  would change code but nothing a player sees.
- Phase 5 polish (motion, state layers, contrast), which has its own change.
