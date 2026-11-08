# Proposal

## Why

After `add-m3-color-roles`, the theme has every Material 3 colour role, but the components still
look half Material 2: the top bar is a solid green band, chips are pills (M3 chips are 8 px
rounded rectangles), there is no tonal button, the toggle groups used as filters do not look like
M3 segmented buttons, and switches, tabs, progress bars, menus and dialogs keep MUI's default
Material 2 look. This is phase 1 of the Material 3 migration plan: restyle every MUI component
the app uses from the theme alone, so the ~150 page files pick up the M3 look without being
edited.

## What Changes

- **Top app bar:** the header moves from the primary green to the `surface` role with `onSurface`
  icons and the logo in `primary`; while content is scrolled under it, it takes
  `surfaceContainer`.
- **Buttons:** a new `tonal` variant (`secondaryContainer` / `onSecondaryContainer`) next to the
  existing contained (filled), outlined and text ones. Pill shape and the 44 px touch height stay.
- **Chips:** 8 px corners instead of pills, 32 px tall (24 px for the compact size the app uses
  in lists); a selected filter chip uses `secondaryContainer` with a leading check.
- **Segmented buttons:** `ToggleButtonGroup` gets the M3 segmented-button look: pill outline,
  `outline` dividers, selected segment in `secondaryContainer`.
- **Cards:** `Card` gets the three M3 styles (`elevated`, `filled`, `outlined`) as variants, for
  the page migration of later phases. Existing `Paper`s are untouched by this change.
- **Tabs, switches, linear progress, menus, dialogs:** M3 indicator, track, thumb and container
  colours and shapes (see the spec for the exact values).
- Every touch-target, font-size and overflow rule from the mobile work (#268, #283) is kept: the
  mobile smoke test is the regression gate.

## Capabilities

### New Capabilities

None (the `ui-theme` capability is created by `add-m3-color-roles`).

### Modified Capabilities

- `ui-theme`: adds the component styles (top app bar, buttons, chips, segmented buttons, cards,
  tabs, switches, progress, menus and dialogs).

## Impact

- Frontend: `frontend/src/theme.ts` (component overrides and `variants`, type augmentation for
  the `tonal` button and the card variants), `frontend/src/app/Header.tsx` (the logo and icon
  colours stop assuming a green bar; scroll state for the container colour),
  `frontend/src/theme.test.ts` (extended), a new `frontend/src/theme.components.test.tsx`.
- No page file needs to change. No new dependency, no backend, API, migration or locale change.
- Depends on `add-m3-color-roles` (the surface-container roles).

## Out of scope

- The navigation bar (bottom bar on phones) and the navigation rail on desktop: phase 2 of the
  plan, its own change (it changes `BOTTOM_NAV_HEIGHT` and the layout).
- Migrating pages from ad-hoc `Paper`s to the card variants, and new composite components
  (search bar, bottom sheet, stat card): phases 3 and 4.
- A floating action button: the app has no FAB today; it is added with the first page that
  needs one.
- Motion (M3 easing and durations) and M3 Expressive shapes.

## Open questions for the owner

1. **Header colour.** M3 puts the top app bar on the surface colour, so the green band
   disappears and the green lives in the logo, the selected states and the primary buttons.
   Keeping a green bar is possible as a branded exception. This proposal follows M3.
2. **Status chips.** The app uses coloured small chips (success, error, warning, info) for match
   results and states, which M3 does not define. This proposal keeps their colours and only
   changes their shape. The alternative, neutral chips with a coloured icon, is a page-level
   change for phase 4.
