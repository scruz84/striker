# Proposal

## Why

Phases 0–2 of the Material 3 plan restyled MUI's own components from the theme and rebuilt the
navigation. Three recurring patterns are still assembled by hand on each page, each a little
differently, and none matches its M3 counterpart:

- **Search fields.** The search page, the national teams filter and the admin users list each
  build an outlined `TextField` with a search adornment. M3 has a dedicated search bar: a 56 px
  pill on a tonal surface, with a way to clear it.
- **The market filter sheet.** It is a bottom `Drawer` with 20 px corners on the default paper
  colour and no drag handle. M3's bottom sheet has a handle, 28 px corners and a tonal surface.
- **Dashboard KPI tiles.** Outlined cards with a coloured icon and an overline label. M3 shows
  this kind of glanceable figure on filled cards with a quiet label.

This is phase 3 of the plan: one shared M3 component per pattern, used everywhere the pattern
appears today.

## What Changes

- **`SearchBar`**, in a new `frontend/src/components/m3/` folder:
  - a pill on `surfaceContainerHigh`, 56 px tall, or 48 px when compact;
  - a leading search icon and the placeholder as its accessible name;
  - a trailing clear button that shows only while there is text.

  It replaces the hand-built fields on the search page, the national teams filter (compact) and
  the admin users list (compact).
- **`BottomSheet`**:
  - an M3 modal bottom sheet on `surfaceContainerLow`, with 28 px top corners and a 32×4 px drag
    handle;
  - at most 640 px wide and 90 % of the window tall;
  - an optional title row with a close button and an optional footer that respects the bottom
    safe area.

  The market's `FilterSheet` becomes a thin wrapper around it.
- **`StatTile`**: a filled card (`surfaceContainerHighest`) with an icon and label in
  `onSurfaceVariant`, the value in the app's `stat` style, and optionally a link covering the
  whole tile. The dashboard's four KPI tiles (budget, squad, position, fans) use it, and keep
  their links and value colours.
- **New strings:** `common.clearSearch` in English ("Clear search") and Spanish ("Borrar
  búsqueda").

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ui-theme`: adds the search bar, bottom sheet and stat tile requirements next to the component
  styles from phase 1.

## Impact

- Frontend only:
  - new `frontend/src/components/m3/SearchBar.tsx`, `BottomSheet.tsx` and `StatTile.tsx`, with
    tests;
  - adoption in `features/search/SearchPage.tsx`,
    `features/nationalteams/NationalMatchFilters.tsx`, `features/admin/UsersTab.tsx`,
    `features/transfers/marketUi.tsx` (`FilterSheet`) and `features/dashboard/DashboardPage.tsx`
    (KPI tiles);
  - `i18n/locales/{en,es}.json`.
- No backend, API or migration change.

## Out of scope

- Switching the remaining pages to M3 card styles and outlined icons (phase 4: page migration).
- Turning other dialogs into bottom sheets: `ResponsiveDialog` is already an M3 full-screen
  dialog on phones.
- Search suggestions or a search view (M3's expanded search): the search page lists results
  itself.

## Open questions for the owner

1. **KPI tile icons in neutral colour.** M3 keeps decoration icons quiet (`onSurfaceVariant`)
   so the figures stand out. Today the icons are green. This proposal follows M3, and the
   budget's red when negative and the fans' rating colour stay.
