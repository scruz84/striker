# Design

## Context

See proposal.md for the motivation; requirements are in `specs/app-navigation/spec.md`. Current
code:

- `frontend/src/app/Layout.tsx` renders `Header`, then a flex row with `NavDrawer` and `<main>`,
  and `BottomNav` below `md` (900 px) for a signed-in user with a team. `<main>` gets
  `pb: BOTTOM_NAV_HEIGHT + safe area` while the bar shows.
- `NavDrawer.tsx` builds the grouped destination list inline (two-tone icons). From `md` it is a
  `permanent` MUI `Drawer` whose paper is `position: relative` inside that row: it scrolls with the
  page, and because MUI gives drawer paper `zIndex.drawer` (1200) above `zIndex.appBar` (1100),
  it paints over the sticky header once scrolled. Below `md` it is a `temporary` drawer opened by
  the bar's "More".
- `BottomNav.tsx` builds its own four destinations (two-tone icons) plus "More";
  `BOTTOM_NAV_HEIGHT = 56` is also read by `LineupSaveBar` (sticky save bar above the bottom bar)
  and mocked in `Layout.test.tsx`.
- The header is 56 px below `md` and 64 px from `md`, always shown from `md` (it only hides on
  scroll on phones). `DetailBackBar`/`Subheader` switch on `md` as well.
- The theme (phase 1) has no `MuiBottomNavigation*` overrides yet.

## Goals / Non-Goals

**Goals:**
- One destination list feeding the bar, the rail and both drawers, so they never disagree.
- The layout switch lives in one hook, so `Layout` and the components agree on the mode.
- Fix the drawer-over-header bug structurally (sticky column under the header), not with a
  z-index tweak alone.

**Non-Goals:**
- Changing the `md` breakpoint that the header, `DetailBackBar` and `Subheader` use. The rail
  range sits inside "desktop" for them, as the permanent sidebar did.

## Decisions

### 1. `useNavMode()` on the existing breakpoints

`navigation.ts` gains `type NavMode = 'bar' | 'rail' | 'drawer'` and
`useNavMode()`: `lg` and up (1200 px) → `drawer`, `md` up (900 px) → `rail`, else `bar`.
M3's window classes put the rail at 600–839 dp and the drawer from 840 dp. The app's own
breakpoints are kept because the header, back bar and lineup save bar already switch at `md`,
and moving phones-in-landscape (600–899 px) off the bottom bar would be a separate decision.

### 2. Shared destinations: `navItems.tsx`

`useNavGroups()` returns the four groups that `NavDrawer` builds today, the same `to`,
`alsoMatches` and admin gating, with `{ icon, selectedIcon }` (`*Outlined` / filled from
`@mui/icons-material`). `usePrimaryDestinations()` returns Dashboard, Squad (only once the team
is known), Lineup and Market, with the short `nav.bottom.*` labels, for the bar and the rail.
`navIcon(item, selected)` picks the glyph (a function, so the module exports no component
next to its hooks). Selection stays as today: the bar and rail match
the exact path (anything else selects "More" on the bar and nothing on the rail); the drawers use
`isNavSelected` with `alsoMatches`.

### 3. Navigation bar styles in the theme

`MuiBottomNavigation.root` → `height: 80`, `surfaceContainer`. `MuiBottomNavigationAction`:
`gap: 4`, `paddingTop: 12`, colour `onSurfaceVariant`, selected `onSurface`; the icon gets
`width: 64, height: 32, padding: '4px 20px', boxSizing: 'border-box', borderRadius: 16` and
`secondaryContainer` / `onSecondaryContainer` when selected; the label is 12 px, 700 when
selected (MUI grows the selected label to 14 px by default, which is dropped). The bar is the
app's only `BottomNavigation`, so the theme is the right place, and `BottomNav.tsx` loses its
per-action `sx`. `BOTTOM_NAV_HEIGHT` becomes 80. `BottomNav` keeps its `Paper` wrapper for the
fixed position and safe-area padding, without the old elevation and top border: M3's bar is a
flat tonal surface.

### 4. `NavRail.tsx`

A `<nav>` `Box` (sharing `STICKY_NAV_COLUMN_SX` from `navigation.ts` with the standard drawer): `width: 80`, `flexShrink: 0`, `position: sticky`, `top: 64` (the `md` header
height), `height: calc(100dvh - 64px)`, `overflowY: auto`, `bgcolor: surface`, column of:
- a menu `IconButton` (48 px, `MenuIcon`, `aria-label` = `nav.bottom.more`, `aria-expanded`
  bound to the drawer state) that calls `onMenu`;
- the four `usePrimaryDestinations()` as `ButtonBase` links (`component={RouterLink}`), each a
  column of a 56×32 pill (`secondaryContainer` when selected) and a 12 px label, min height 56,
  `aria-current="page"` when selected.

No MUI component exists for an M3 rail, so it is a small custom component styled from palette
roles only.

### 5. `NavDrawer`: modal or standard, M3 items

`NavDrawer({ variant: 'modal' | 'standard', open, onClose })`. Item styling is local (`sx` on
`ListItemButton` inside the drawer), not a global `MuiListItemButton` override, because other
lists in the app use `ListItemButton` with their own look: `minHeight: 56`, `borderRadius: 28`,
`mx: 1.5`, `px: 2`, selected `secondaryContainer`/`onSecondaryContainer` (icon included).
`NAV_DRAWER_WIDTH` goes from 232 to 256 to keep the labels on one line inside the 12 px side
margins.
- `modal`: `temporary`, anchored left, paper `surfaceContainerLow` with
  `borderRadius: '0 16px 16px 0'`, `keepMounted` as today.
- `standard`: not an MUI `Drawer` any more but a sticky `<nav>` column (`top: 64`,
  `height: calc(100dvh - 64px)`, `overflowY: auto`, `bgcolor: surface`), so it never stacks
  above the header. This is the bug fix.

*Alternative considered:* keep the permanent `Drawer` and lower its paper's z-index below the
app bar. Rejected: the sidebar would still scroll away with the page, and a drawer that disappears
on scroll is the real defect.

### 6. `Layout`

`const mode = useNavMode()`; `showNav = !!loggedUser && !loggedUser.pendingCountrySelection`.
Bar when `mode === 'bar'`, `NavRail` when `rail`, `NavDrawer variant="standard"` when `drawer`,
and `NavDrawer variant="modal"` (opened by "More" or the rail menu) in the other two modes. The
`<main>` bottom padding uses the new `BOTTOM_NAV_HEIGHT`.

## Risks / Trade-offs

- [The bar is 24 px taller on phones] → Owner question 2. The smoke test guards that content is
  not hidden ("Last content not hidden"), and `LineupSaveBar` follows `BOTTOM_NAV_HEIGHT`
  automatically.
- [Laptop users lose one-tap access to less frequent pages] → Owner question 1. The menu button
  is the rail's first entry, and the full list is one tap away.
- [`100dvh` support] → It is supported by every evergreen browser the app targets. Where it is
  not, the declaration is dropped and the column only loses its own scroll.
- [Translated labels wider than 72 px on the bar] → Spanish is the longest locale today
  ("Alineación"); the smoke test checks for horizontal overflow at 360 px.

## Migration Plan

Frontend-only and stateless: ships with the next frontend deploy, rolls back by reverting.
