# Design

## Context

See proposal.md for the motivation; requirements are in `specs/ui-theme/spec.md`. Current code:

- Search fields: `features/search/SearchPage.tsx` (autofocused outlined `TextField`, two-tone
  search adornment), `features/nationalteams/NationalMatchFilters.tsx` and
  `features/admin/UsersTab.tsx` (small outlined `TextField`s with an `aria-label` on the input).
  All three are controlled (`value` / `onChange`).
- `features/transfers/marketUi.tsx` → `FilterSheet`: a bottom `Drawer` (20 px top corners,
  default paper, `maxWidth: 560`, `90dvh`) with a title row and close button, a scrolling body and
  a footer ("Clear" / "Show N results") padded for the safe area. Three market tabs use it.
- `features/dashboard/DashboardPage.tsx` → KPI tiles: `Grid` of `Card variant="outlined"` +
  `CardActionArea` links, two-tone icons in `primary`, overline label, `stat` value with an
  optional colour.
- Phase 1 already gives `Card variant="filled"` (`surfaceContainerHighest`) and the palette
  roles every component needs.

## Goals / Non-Goals

**Goals:**
- Three small components with the M3 look built in, and every current instance of each pattern
  moved onto them, so there is no second way left.
- No change in behaviour: same values, links, focus and accessible names, except for the new
  clear button.

**Non-Goals:**
- A component library beyond these three (phase 4 adds what page migration needs).

## Decisions

### 1. `components/m3/` folder

The new components live in `frontend/src/components/m3/`, separate from the app-specific shared
components in `components/`, because they are design-system building blocks with no domain
knowledge.

### 2. `SearchBar`: an `InputBase` in a pill, not a restyled `TextField`

`SearchBar({ value, onChange, placeholder, compact?, autoFocus?, sx? })` renders a `Box`
(`borderRadius: '9999px'`, `bgcolor: surfaceContainerHigh`, height 56 or 48, `px: 2`, focus-within
outline in `primary`), with a `SearchIcon` in `onSurfaceVariant`, an `InputBase` that takes
`inputProps={{ 'aria-label': placeholder }}` and `type="search"` (so its role is `searchbox`;
the input stretches to the pill's full height, which keeps the tap target at 56 / 48 px for the
mobile smoke test), and an `IconButton`
(`aria-label` = `common.clearSearch`) shown only when `value !== ''`. The button calls
`onChange('')` and refocuses the input through a ref. `onChange` receives the string, not the
event, which is all the three call sites use.

*Alternative considered:* theme-only styling of a `TextField` variant. Rejected: the clear
button and pill geometry would still need to be added at every call site.

### 3. `BottomSheet`

`BottomSheet({ open, onClose, title?, footer?, children })` is a bottom-anchored `Drawer` with
paper `bgcolor: surfaceContainerLow`, `borderTopLeftRadius`/`borderTopRightRadius: 28`,
`maxWidth: 640`, `width: '100%'`, `mx: 'auto'`, `maxHeight: '90dvh'`. It renders:
- a drag handle (`32×4`, `borderRadius: 2`, `onSurfaceVariant` at 40 % opacity, `my: 1.5`,
  `aria-hidden`);
- an optional title row (`h2`, close button);
- a scrolling body;
- an optional footer with `pb: max(16px, env(safe-area-inset-bottom))`.

The handle is visual only: swipe-to-dismiss is out of scope, and the backdrop and Escape close
it as today. `FilterSheet` keeps its props and passes its two buttons as `footer`.

### 4. `StatTile`

`StatTile({ icon, label, value, valueColor?, to? })` renders `Card variant="filled"`, with a
`CardActionArea component={RouterLink}` when `to` is set (`minHeight: 80`, `p: 1.5`). Inside: the
icon (forced to `onSurfaceVariant`, small) and the label as `overline` in `onSurfaceVariant`,
then the value as `Typography variant="stat"` at 1.5 rem with `valueColor`. The dashboard's tile
list keeps building its four entries and maps them to `StatTile`, with outlined icons instead of
two-tone ones.

## Risks / Trade-offs

- [The clear button adds a tab stop] → It only exists while there is text, and it is a labelled
  button, so keyboard and screen-reader users get the same affordance.
- [Filled tiles have less contrast against the page than outlined ones] → The `stat` value
  colour (`onSurface`) on `surfaceContainerHighest` is 13:1 (pinned by `theme.test.ts`), and the
  tile edge is a 0.9:1 tonal step, which is how M3 draws filled cards.

## Migration Plan

Frontend-only and stateless; ships with the next frontend deploy, rolls back by reverting.
