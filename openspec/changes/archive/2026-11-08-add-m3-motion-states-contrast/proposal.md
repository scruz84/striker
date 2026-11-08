# Proposal

## Why

Phases 0–4 of the Material 3 plan brought colour, components, navigation, composite components
and icons to M3. Three parts of the M3 system that players feel rather than see are still MUI's
Material 2 defaults:

- **Motion.** Dialogs, drawers, sheets and the header use MUI's Material 2 easing and durations,
  not M3's standard and emphasized curves.
- **State layers.** Hover is a 4 % tint and focus a 12 % one. M3 uses 8 % for hover and 10 % for
  focus and press.
- **Accessibility settings.** The app ignores the system's "reduce motion" and "increase contrast"
  settings. M3 has a high-contrast scheme for the second, and the colour library we use already
  generates it.

This is phase 5 of the plan: the final polish.

## What Changes

- **M3 motion tokens in the theme.** MUI's four easings become M3 standard, emphasized decelerate,
  emphasized accelerate and standard accelerate. The durations follow M3's tokens: 200 ms to
  leave, 300 ms standard, 400 ms to enter. Everything that animates through the theme picks them
  up.
- **Reduced motion.** With the system's "reduce motion" on, transitions and animations end
  immediately, including slides, ripples and the pulsing LIVE chip.
- **M3 state layers.** Hover is 8 % of the content colour, focus and press 10 %, and inactive icons
  use `onSurfaceVariant`.
- **High contrast.** With the system's "increase contrast" on (`prefers-contrast: more`), the app
  uses M3's high-contrast scheme from the same seed. Every text/background pair reaches 7:1, and
  the brand stays green.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ui-theme`: adds the motion, reduced motion, state layer and high contrast requirements.

## Impact

- Frontend only:
  - `frontend/src/theme.ts`: `buildTheme(mode, highContrast)`, the `M3_TRANSITIONS` tokens, the
    palette `action` values, a `CssBaseline` media rule, and the two new high-contrast theme
    exports;
  - `frontend/src/app/ThemeModeProvider.tsx`, which picks the high-contrast theme;
  - tests.
- No new strings or settings screen: both accessibility behaviours follow the operating system.
  No backend, API or migration change.

## Out of scope

- An in-app contrast or motion switch. The OS settings already exist on every platform the app
  targets, and the theme mode toggle stays light / dark only.
- M3 Expressive (shape morphing, springs) and dynamic colour per club.
- The medium contrast level. The CSS media query only distinguishes "more" from no preference.
