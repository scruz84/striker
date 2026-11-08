# Proposal

## Why

The client is already built on MUI with a Material 3 colour layer (`frontend/src/theme.ts`), but
that layer uses the legacy `Scheme` API of `@material/material-color-utilities`. It has no M3
surface-container roles (`surfaceContainerLowest` … `surfaceContainerHighest`), so the theme fakes
elevation by blending 5 % (light) / 10 % (dark) of the primary green into the paper colour, and
pages have no tokens for the filled cards, navigation bar, search bar and dialogs of the M3 spec.
This is phase 0 of the Material 3 migration plan: every later phase (component theme, adaptive
navigation, page migration) needs these roles first.

## What Changes

- The theme's colours come from an M3 `DynamicScheme` instead of the legacy `Scheme`. The core
  palettes stay the ones the app uses today, so the brand colours do not move: primary stays
  `#006e1c` (light) / `#78dc77` (dark).
- The palette gains the surface roles `surfaceDim`, `surfaceBright` and the five
  `surfaceContainer*` levels, readable from any component through the theme.
- The page background becomes the M3 `surface` role and raised panels (`Paper` at its default
  elevation) use `surfaceContainerLow` instead of the primary-tinted blend. **Visible change:**
  raised panels lose their faint green tint and read as a neutral, slightly darker tone.
- The `on*Container` text colours move to the current M3 tones (tone 30 instead of tone 10 in
  light mode), e.g. the live hero's text on the error container goes from `#410002` to `#93000a`.
  Every pair keeps at least 7:1 contrast.
- A unit test pins the contrast of every role pair and the unchanged primary colours.

## Capabilities

### New Capabilities

- `ui-theme`: the client's visual design system: where colours come from, the contrast every
  colour pair guarantees, and which surface each kind of container sits on. Later phases of the
  migration add the component styles to the same capability.

### Modified Capabilities

None. The team dashboard's live hero keeps using the error-container role pair its spec already
requires; only the tone of that pair changes.

## Impact

- Frontend only: `frontend/src/theme.ts` (scheme construction, palette type augmentation, the
  `MuiPaper` elevation override, removal of the tint blend), a new `frontend/src/theme.test.ts`.
- No component or page file changes, no new dependency (`@material/material-color-utilities`
  0.4 already ships `DynamicScheme`), no backend, API, migration or locale change.

## Out of scope

- Component shape and state changes (chips, buttons, segmented buttons, app bar, tabs…): change
  `complete-m3-component-theme`.
- The navigation bar, navigation rail and page-by-page migration: later phases.
- Dynamic colour per club (seed taken from the crest) and the medium/high contrast levels.
- The domain colours that are not theme roles: medal gold/silver/bronze, the pitch stripes, the
  cartoon avatar inks and club crest colours.

## Open questions for the owner

1. **Loss of the green tint on raised panels.** The M3 spec replaced tonal tint with neutral
   surface containers in 2023, and this proposal follows it. Keeping a tint would mean staying
   off-spec. Confirm that neutral raised panels are acceptable.
2. **Darker `on*Container` text.** This proposal adopts the current M3 tones (tone 30). Pinning
   the old tone 10 is possible but diverges from what MUI and Material components will expect.
