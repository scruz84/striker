# Design

## Context

See proposal.md for the motivation; requirements are in `specs/ui-theme/spec.md`. Current state
after `add-m3-color-roles`:

- `frontend/src/theme.ts` → `buildComponents(mode)` already overrides `MuiButton` (pill, 24 px
  padding, 44 px touch, legible disabled state), `MuiChip` (pill, 44 px invisible hit area on
  clickable chips), `MuiTab` (48 px), `MuiSwitch` (touch padding), `MuiToggleButton` (44 px
  touch), `MuiPaper`, `MuiDialog` (28 px), `MuiAppBar` (no shadow). Touch rules live behind the
  `TOUCH` media query (`frontend/src/touchMedia.ts`).
- `frontend/src/app/Header.tsx` renders `<AppBar position="sticky">` with the default `primary`
  colour; its icons and links use `color="inherit"` and the logo (`Logo.tsx`) draws in
  `currentColor`. On phones `useScrollTrigger({ threshold: 64 })` slides it away.
- Usage across pages: 78 `Chip`s (51 `size="small"`, mostly status colours), 6
  `ToggleButtonGroup`s (one, the lineup mentality picker, compacted by page `sx` to 32/36 px in
  #365), 3 `Switch`es, 10 `Tabs`, 4 dialogs plus `ResponsiveDialog` (full screen on phones), no
  `Fab`, 13 `Card`s.
- One test inspects MUI class names (`WorldTab.test.tsx`, `MuiDialog-paperFullScreen`); no
  snapshot tests.

## Goals / Non-Goals

**Goals:**
- Every requirement of the spec is met by `theme.ts` plus `Header.tsx` only.
- Page-level `sx` keeps winning over the theme, so deliberate page tweaks (the compact mentality
  picker) survive.

**Non-Goals:**
- Rewriting MUI components or wrapping them in project components (phase 3).
- Changing any page file.

## Decisions

### 1. Theme overrides and MUI `variants`, no wrapper components

All styles go into `buildComponents`. New visual variants use MUI's `variants` array with module
augmentation: `ButtonPropsVariantOverrides { tonal: true }` and
`CardPropsVariantOverrides { elevated: true; filled: true }` (`outlined` already exists on
`Card`, through `Paper`). Pages opt in with `variant="tonal"` and `variant="filled"`, typed.

*Alternative considered:* `components/m3/TonalButton.tsx`-style wrappers. Rejected for this
phase: wrappers mean editing call sites, and `variants` gives the same typing with zero page
changes.

### 2. Header: colour from the AppBar, logo in primary, scroll state on desktop

`MuiAppBar.defaultProps.color = 'default'` and a `colorDefault` override with
`backgroundColor: surface`, `color: onSurface`. `Header.tsx` wraps `<Logo />` in
`sx={{ color: 'primary.main' }}` and adds a second
`useScrollTrigger({ disableHysteresis: true, threshold: 0 })` whose value switches the bar's
background to `surfaceContainer`. The phone hide-on-scroll trigger is unchanged.
`ResponsiveDialog`'s own `AppBar` already sets `color="default"` with `background.paper`, so it
is not affected.

### 3. Chips: shape only, colours kept

`MuiChip.root` → `borderRadius: 8`, `height: 32`; `sizeSmall` → `height: 24`. The 44 px `::after`
hit area stays. A selected filter chip is expressed as `color="primary"` + `variant="filled"` +
a check icon: the override maps `colorPrimary` filled chips to `secondaryContainer` /
`onSecondaryContainer`. The existing `success`/`error`/`warning`/`info` chips keep MUI's palette
colours (owner question 2 in the proposal).

*Risk checked:* the 7 `color="primary"` chips today are used as selected or highlighted
markers, which is what the new style means; each is reviewed in task 2.3.

### 4. Segmented buttons from `ToggleButtonGroup`

`MuiToggleButtonGroup` → `firstButton`/`lastButton` slots with `borderRadius: 9999`,
`grouped` border `outline`; `MuiToggleButton` → selected background `secondaryContainer`, text
`onSecondaryContainer`, `minHeight: 40` (44 under `TOUCH`), `textTransform: 'none'`. A leading
check icon on the selected segment is left to pages (it costs width at 360 px; the spec does not
require it).

### 5. Switch geometry

MUI's switch is a 58×38 root with a 14 px track and a 20 px thumb. The override sets the root to
52×32 plus 6 px padding for the tap area (44 px under `TOUCH`), the track to 32 px tall with a
2 px `outline` border when off and `primary` when on (opacity 1), and the thumb to 16 px off /
24 px on, coloured `outline` / `onPrimary`. Implemented with the `switchBase`, `thumb`, `track`
and `checked` slots; no custom component.

### 6. Surfaces of floating containers

`MuiMenu.paper` and `MuiPopover.paper` → `surfaceContainer`, `borderRadius: 4`;
`MuiDialog.paper` → `surfaceContainerHigh`, 28 px, and `paperFullScreen` → `borderRadius: 0`.
`MuiTabs.indicator` → 3 px, `borderRadius: '3px 3px 0 0'`; `MuiTab` inactive colour
`onSurfaceVariant`. `MuiLinearProgress` → 4 px, rounded, track `secondaryContainer`.
`MuiCard` → `variants` for `elevated` (`surfaceContainerLow`, level-1 shadow
`0 1px 2px rgba(0,0,0,.3), 0 1px 3px 1px rgba(0,0,0,.15)`) and `filled`
(`surfaceContainerHighest`).

## Risks / Trade-offs

- [Header loses its green band: weaker brand presence at first glance] → Owner question 1; the
  logo, primary buttons, selected states and the hero card carry the green. Mock screenshots of
  the plan show the result.
- [Pages that set colours assuming a green header (`color="inherit"` on white text)] → Only
  `Header.tsx` and `NotificationBell.tsx` sit in the bar; both use `inherit`, which now resolves
  to `onSurface`. Checked visually in task 3.2.
- [Switch override breaks MUI's internal layout on a future MUI minor] → The component test
  measures the track and thumb sizes, so a regression fails CI.
- [Taller default chips (32 px) push list layouts] → 51 of 78 chips are `size="small"`, which
  stays 24 px; the smoke test at 360 px is the overflow gate.

## Migration Plan

Frontend-only, stateless; ships with the next frontend deploy and rolls back by reverting the
commit. Must be merged after `add-m3-color-roles`.
