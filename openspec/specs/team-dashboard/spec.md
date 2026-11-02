# team-dashboard Specification

## Purpose

The owner's team dashboard: what its hero slot shows (the next fixture, or the team's own match while
it is being broadcast) and how the rest of the page avoids duplicating or spoiling that live match.

## Requirements

### Requirement: Hero shows the next match
When none of the team's matches is live, the dashboard hero SHALL show the team's next fixture across
league, cup and friendly matches: the earliest one whose kickoff is still in the future, with
opponent, home/away, kickoff date, countdown, head-to-head record, "Edit lineup" and "View match". It
uses the green primary-container surface. With no upcoming fixture it shows "No upcoming match scheduled."

#### Scenario: Friendly before the next league match
- **WHEN** the team has a friendly tomorrow at 18:00 and a league match the day after
- **THEN** the hero shows the friendly with its countdown on the green card

### Requirement: Own live match takes the hero
A team match is live from its kickoff until its broadcast reports completion, for at most 2 hours
after kickoff. While one is live, the hero SHALL show it instead of the next fixture. If two are
live, it shows the one with the latest kickoff. A match marked played before its kickoff is not live
until kickoff arrives.

#### Scenario: Match in progress
- **WHEN** the owner opens the dashboard 30 minutes after their league match kicked off
- **THEN** the hero shows that match, not the following fixture

#### Scenario: Simulated early
- **WHEN** the team's match was simulated 10 minutes before its 20:00 kickoff and it is 19:55
- **THEN** the hero still shows it as the next match with its countdown, not as live

#### Scenario: Broadcast older than the window
- **WHEN** a match kicked off 2 hours and 1 minute ago and no completion was received
- **THEN** it is no longer live and the hero shows the next fixture

### Requirement: Live hero content
The live hero SHALL show a pulsing "LIVE" label with the match kind, both teams with badges in
home/away order, the score and minute from the match's live broadcast, and a primary "Watch live"
button that opens the match viewer. During the half-time break the minute reads "Half-time" instead.

#### Scenario: Score as broadcast
- **WHEN** the owner's cup tie is at minute 63 of its broadcast with a 1-2 scoreline so far, and the final result stored in the database is 3-2
- **THEN** the hero reads "1 - 2" and "63'", never "3 - 2"

#### Scenario: Half-time
- **WHEN** the broadcast is in its half-time pause
- **THEN** the hero shows the score at minute 45 and "Half-time" instead of a minute

#### Scenario: Before the first broadcast minute
- **WHEN** kickoff has passed but the broadcast has not published its first minute yet
- **THEN** the hero shows "0 - 0" with the "LIVE" label and no minute

#### Scenario: Broadcast channel unreachable
- **WHEN** the live connection gives up reconnecting while the match is still within its live window
- **THEN** the hero keeps the "LIVE" label and "Watch live" button but shows no score or minute

### Requirement: Live hero colour
The live hero SHALL use the theme's error-container surface with its on-error-container text colour
(a light red in light mode, a dark red in dark mode), so it is visually distinct from the green
next-match card. Text and buttons on it meet WCAG AA contrast (4.5:1) in both modes.

#### Scenario: Dark mode
- **WHEN** the dashboard is rendered in dark mode while the team's match is live
- **THEN** the hero uses the dark-scheme error-container colour, not the primary container

### Requirement: Hero returns to the next match at full time
When the live broadcast reports completion, the dashboard SHALL switch the hero back to the next
fixture without a page reload, and the finished match becomes the "Last result".

#### Scenario: Final whistle while watching the dashboard
- **WHEN** the owner keeps the dashboard open and the broadcast sends its completion message
- **THEN** the hero shows the next fixture's green card and "Last result" shows the finished match's score

### Requirement: No duplicate live notices
While the team's own match is live, the dashboard SHALL NOT show a separate "your match is live"
alert, and the "Last result" card SHALL show the latest match whose broadcast has finished, never the
live one.

#### Scenario: Last result during a live match
- **WHEN** the team won 2-0 last week and is now playing a live friendly
- **THEN** "Last result" shows the 2-0 win and no "your match is live" alert appears above the hero

## Code map

- Dashboard: `frontend/src/features/dashboard/DashboardPage.tsx` (`selectLiveMatch`, `useOwnLiveMatch`,
  `selectLastResult`, `LiveMatchHero`, `NextMatchHero`, `LastResultCard`).
- Pending panel: `frontend/src/features/dashboard/PendingPanel.tsx` (data wiring, snooze buttons, phone
  "Show all (N)"), `features/dashboard/pendingItems.ts` (`buildPendingItems`, the severity order and the
  `FORECAST_WARNING_WEEKS` / `TICKET_PRICE_GAP` / `CONTRACT_EXPIRY_WARNING_YEARS` thresholds),
  `features/dashboard/pendingSnooze.ts` (localStorage snooze until local midnight, content key),
  `features/finances/cashFlowForecast.ts` (`computeCashFlowForecast`) and the `?ticketPrice=1` deep link
  that makes `StadiumCard` (`features/finances/FinancesPage.tsx`) open its price dialog.
- Live broadcast: `frontend/src/api/useLiveMatch.ts` (`useLiveMatch`, `mightStillBeLive`, `isPast`).
- Shared helpers: `frontend/src/utils/matchSummary.ts` (`liveMatchPath`), `utils/liveChip.ts` (pulsing LIVE
  style), `theme.ts` (`errorContainer` / `onErrorContainer` tokens).
- Strings: `dashboard.pending.*`, `dashboard.liveNow`, `dashboard.watchLive`, `dashboard.liveMatchMinute`, `match.halftime` in
  `i18n/locales/{en,es}.json`.
