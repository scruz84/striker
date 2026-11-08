# Design

## Context

See proposal.md for the motivation; the requirement is in `specs/ui-theme/spec.md`. Two-tone
icons are imported by default path (`@mui/icons-material/<Name>TwoTone`) under a local name that
ends in `TwoToneIcon`, in these files:

- `frontend/src/components/RivalryChip.tsx`
- `frontend/src/features/cup/InternationalCupPage.tsx`
- `frontend/src/features/cup/CupPage.tsx`
- `frontend/src/features/team/TeamDetailsPage.tsx`
- `frontend/src/features/leagues/LeaguesPage.tsx`
- `frontend/src/features/leagues/RecentLeagueTransfers.tsx`
- `frontend/src/features/search/SearchPage.tsx`
- `frontend/src/features/nationalteams/WorldCupTab.tsx`
- `frontend/src/features/nationalteams/RoundTab.tsx`
- `frontend/src/features/nationalteams/NationalMatchRow.tsx`
- `frontend/src/features/help/HelpPage.tsx`
- `frontend/src/features/dashboard/DashboardPage.tsx`
- `frontend/src/features/division/DivisionDetailsPage.tsx`
- `frontend/src/features/friendlies/FriendliesPage.tsx`

All 20 glyphs (EmojiEvents, SportsSoccer, Public, Handshake, Groups, SwapHoriz, Whatshot, Sports,
Smartphone, Shuffle, School, Person, Outbound, LiveTv, Leaderboard, Flag, Dashboard,
CalendarToday, Apartment, AccountBalanceWallet) exist as `*Outlined` in the installed
`@mui/icons-material`.

## Goals / Non-Goals

**Goals:**
- A mechanical rename: same glyph, same props, same place.
- A guard so the next two-tone import is caught by `npm test`, not by review.

**Non-Goals:**
- Choosing different glyphs, colours or sizes.

## Decisions

### 1. Rename import path and local name together

`<Name>TwoTone` → `<Name>Outlined` in the import path and `<Name>TwoToneIcon` →
`<Name>OutlinedIcon` in the local name and every JSX use, with a scripted find-and-replace over
the 14 files. If a file already imports the outlined icon, the duplicate import is dropped. The
compiler (`tsc -b`, `noUnusedLocals`) catches a missed reference or a leftover import.

### 2. `iconStyle.test.ts`

A vitest test walks `frontend/src` (Node `fs`, skipping `node_modules`) and fails, listing
file and line, for any import matching `@mui/icons-material/[A-Za-z]+TwoTone`. It reads source
text rather than rendered output, because a two-tone icon in a rarely rendered branch would never
show up in a render test.

*Alternative considered:* an oxlint `no-restricted-imports` rule. Rejected for now: the
project's `.oxlintrc.json` has no import restrictions yet, and lint warnings do not fail the
build, while a test does.

## Risks / Trade-offs

- [Outlined glyphs are lighter than two-tone ones at small sizes] → They are M3's default weight
  and already sit next to them in navigation. The screenshots in the PR compare dashboard, cup,
  leagues and national teams before and after.
