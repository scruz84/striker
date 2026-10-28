# Proposal

## Why

The team dashboard opens with a large green "Next match" hero card. While the owner's own match is
being broadcast, that hero already jumps ahead to the *following* fixture, and the match that is
actually being played right now is relegated to a small info alert above it (plus a LIVE chip in
the "Last result" card). The most important thing on the page at that moment is the live match, so
it should own the hero slot. Separately, the match viewer's subtitle for a league match only says
"Jornada N de T", so a viewer who arrived from the worldwide live feed or a notification cannot tell
which division the match belongs to.

## What Changes

- While the owner's team has a match inside its live broadcast (league, cup tie or friendly), the
  dashboard hero card shows that match instead of the next fixture: a "LIVE" header with the match
  kind, both teams, the current score and minute (or "Half-time") as broadcast, and a primary
  "Watch live" button to the match viewer.
- The live hero uses a distinct light-red surface (the theme's M3 error-container tone, in light and
  dark mode) so it is never confused with the green "Next match" card.
- When the broadcast completes, the hero goes back to the "Next match" card without a reload.
- The separate "Your match against X is live right now!" alert is removed (the hero replaces it),
  and the "Last result" card shows the latest *finished* match instead of a LIVE chip for the one in
  progress.
- The match viewer's subtitle for a league match is prefixed with the division name:
  "<division> · Jornada N de T · <date> · <weather>", in every phase (scheduled, live, finished).

Out of scope:
- Any backend or API change: the live score and minute come from the match's existing live
  broadcast channel, and the division name from the division details the league route already loads.
- Showing goal scorers, cards or events in the hero (the viewer is one tap away; red cards and
  injuries stay withheld per the live spoiler rule).
- Changing the worldwide "Live around the world" card.
- Cup and friendly subtitles, which already name their competition.

Open questions for the owner:
- Should the live hero also carry a one-line teaser of the next fixture (opponent and date)
  underneath, so the countdown is still reachable mid-match? This plan leaves it out; the next match
  reappears in the hero as soon as the broadcast ends.

## Capabilities

### New Capabilities
- `team-dashboard`: what the owner's team dashboard shows in its hero slot (next match vs. own match
  in progress), how the live state is presented, and what the "Last result" card shows meanwhile.

### Modified Capabilities
- `match-viewer`: the "Scoreboard subtitle" requirement's league-match label gains the division name.

## Impact

- Frontend only: `features/dashboard/DashboardPage.tsx` (hero selection, new live hero, removal of
  `LiveMatchBanner`, `LastResultCard` input), `features/match/matchViewModel.ts` (`fromWeekMatch`
  label), `theme.ts` (expose `errorContainer` / `onErrorContainer`), a shared live-path helper in
  `utils/matchSummary.ts`, and `i18n/locales/{en,es}.json`.
- One extra WebSocket connection from the dashboard, only while the owner's match is live.
- New spec `openspec/specs/team-dashboard/spec.md`; `match-viewer` spec subtitle and code map updated.
