# Design

## Context

See proposal.md for the motivation; requirements are in `specs/ui-theme/spec.md`.

- `frontend/src/theme.ts` builds `lightTheme` / `darkTheme` with `createTheme` from
  `buildScheme(isDark)`, a `DynamicScheme` with `contrastLevel: 0` (phase 0). It sets no
  `transitions` and no palette `action`, so MUI's Material 2 easings (`0.4, 0, 0.2, 1` …), its
  durations (enter 225 ms, leave 195 ms), its 4 % hover and its 12 % focus apply.
- `frontend/src/app/ThemeModeProvider.tsx` picks between the two themes from the stored light / dark
  mode.
- MUI's transitions (`Slide`, `Fade`, `Grow`, `Collapse`, `Dialog`, `Drawer`, the bar and chips)
  all read `theme.transitions`. Hover and focus colours read `palette.action.hover` /
  `hoverOpacity` / `focus` / `focusOpacity`.

## Goals / Non-Goals

**Goals:**
- Theme-level only: no component or page changes.
- Accessibility preferences honoured without new UI.

**Non-Goals:**
- Per-component motion choreography (container transforms, shared-axis transitions).

## Decisions

### 1. Map MUI's easing slots onto M3 curves

MUI exposes four easings and uses them in fixed places: `easeInOut` for in-place changes,
`easeOut` for entering, `easeIn` for leaving, `sharp` for temporary exits. They map to M3 standard,
emphasized decelerate, emphasized accelerate and standard accelerate respectively. The durations
go to M3 tokens: `enteringScreen` 400 (medium4), `leavingScreen` 200 (short4), `standard` 300
(medium2), `short` 200, `shorter` 150, `shortest` 100, `complex` 400. Exported as
`M3_TRANSITIONS` so a component animating by hand can reuse them.

### 2. State layers through `palette.action`

`action.hover = alpha(onSurface, 0.08)`, `hoverOpacity: 0.08`, `focus = alpha(onSurface, 0.1)`,
`focusOpacity: 0.1`, `activatedOpacity: 0.1`, `active = onSurfaceVariant`. Both the colour and the
opacity are set, because MUI precomputes `hover`/`focus` and components use either one. Pressed
states use MUI's ripple, unchanged.

### 3. Reduced motion in `CssBaseline`

`@media (prefers-reduced-motion: reduce)` sets `animation-duration` / `transition-duration` to
`0.01ms !important` and `animation-iteration-count: 1` on every element. Using 0.01 ms rather than 0
keeps `transitionend` / `animationend` firing, which react-transition-group waits for.

### 4. High contrast: `buildTheme(mode, highContrast)` + `prefers-contrast`

`buildScheme(isDark, contrastLevel)` passes `1` (M3 high) instead of `0`. `buildTheme` wraps
`createTheme` and builds four themes at load (light / dark × standard / high). The two existing
exports keep their names, so tests and imports do not change.
`ThemeModeProvider` reads `useMediaQuery('(prefers-contrast: more)')` and picks the matching
theme. The rating scale is derived from the scheme's `outline` and `primary`, so it follows.

*Alternative considered:* a contrast toggle in the account menu. Rejected for this phase: it would
need strings, persistence and a design for three levels, while the OS setting already reaches
everyone who needs it.

## Risks / Trade-offs

- [400 ms entering dialogs feel slower than MUI's 225 ms] → It is M3's value for full-surface
  entries, on a decelerating curve that shows most of the motion early. Exits get faster
  (200 ms).
- [High-contrast primary is nearly black in light mode (`#003509`)] → This is M3's high-contrast
  output for the seed. Its hue is still green, and it is used only when the player asked the
  system for more contrast.
