# Proposal

## Why

Phases 0 and 1 of the Material 3 plan (`add-m3-color-roles`, `complete-m3-component-theme`) gave
the app M3 colours and components, but its navigation is still Material 2. On phones, the bottom
bar is 56 px tall with tinted icons and no active indicator. On every screen from 900 px up, a
232 px permanent sidebar takes the left edge, even on a 1024 px laptop where that width is
better used by the page. That sidebar also scrolls with the page and paints over the sticky
header, which hides the logo as soon as the user scrolls (visible on `main` today). This is
phase 2 of the plan: M3 adaptive navigation.

## What Changes

- **Three navigation layouts by window width** (M3 window size classes, on the app's existing
  breakpoints):
  - under 900 px: a navigation bar;
  - 900–1199 px: a navigation rail;
  - from 1200 px: a standard navigation drawer.
- **Navigation bar (phones).** The bottom bar becomes the M3 navigation bar:
  - 80 px tall on `surfaceContainer`;
  - a 64×32 pill active indicator in `secondaryContainer`, with the icon filled when active and outlined otherwise;
  - 12 px labels.

  It keeps the same five entries: Dashboard, Squad, Lineup, Market, More. **Visible change:** the
  bar is 24 px taller, and pages leave 80 px for it instead of 56.
- **Navigation rail (900–1199 px, new).**
  - An 80 px column, sticky under the header.
  - A menu button that opens the full navigation, then the four daily destinations, with the
    same pill indicator.
  - It replaces the permanent sidebar at those widths, so the page gains 152 px.
- **Modal navigation drawer.** "More" on the bar and the rail's menu button open the full
  grouped navigation as an M3 modal drawer:
  - `surfaceContainerLow` background with 16 px outer corners;
  - 56 px pill items, the current one in `secondaryContainer`.
- **Standard navigation drawer (from 1200 px).** The permanent sidebar keeps its groups, takes
  the same pill items, and becomes sticky under the header with its own scroll. It no longer
  paints over the header, which fixes the hidden logo.
- **Icons.** All navigation entries use outlined icons, and filled ones for the current entry,
  instead of two-tone.

## Capabilities

### New Capabilities

- `app-navigation`: how the signed-in player moves between the app's sections at each window
  width: which navigation layout shows, what each one contains and looks like, and how it
  relates to the header and the page content.

### Modified Capabilities

None. The `ui-theme` component styles do not change, and the new capability owns the navigation
look.

## Impact

- Frontend only:
  - `frontend/src/app/` (`navigation.ts`, `BottomNav.tsx`, `NavDrawer.tsx`, `Layout.tsx`);
  - a new `navItems.tsx` (the shared destination list);
  - a new `NavRail.tsx`;
  - `frontend/src/theme.ts` (navigation bar styles);
  - `frontend/src/features/lineup/LineupSaveBar.tsx`, which reads `BOTTOM_NAV_HEIGHT` and needs
    no change itself.
- No new locale strings: the rail reuses `nav.bottom.*`, and the drawer reuses `header.nav.*`
  and `nav.groups.*`.
- No backend, API or migration change.

## Out of scope

- The page-by-page migration to M3 cards and composite components (phases 3 and 4).
- A collapsible / expandable rail, and a FAB in the rail: the app has no primary creation action.
- Changing which destinations the bar and the rail show, or the grouping of the full navigation.
- The header's content: logo, search, bell and avatar are unchanged.

## Open questions for the owner

1. **Rail between 900 and 1199 px.** This proposal follows M3 and gives laptops a rail, so the
   less frequent destinations (Matches, Finances, Leagues, Cup, …) are one extra tap away
   through the menu button. The alternative is to keep the permanent sidebar from 900 px and use
   the rail nowhere.
2. **Taller phone bar (80 px).** This is the M3 size. The old 56 px would keep more page visible
   on small phones, but the icon and indicator no longer fit with a label at that height.
