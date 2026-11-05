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

### Requirement: Rest of the matchday in the live hero
When the live hero shows a league match, it SHALL include a "Rest of the matchday" section listing the division live scores entries of the same `weekNumber`, excluding the owner's own match, in kickoff then match id order. Each row SHALL show both teams, the score and the state ("63'", "Half-time", "FT", or the kickoff time while `pending`) and link to `/match/:id`.

#### Scenario: League match live
- **WHEN** the owner's league match of matchday 12 is live and the other 7 matches of matchday 12 are broadcasting
- **THEN** the hero lists those 7 matches with their broadcast scores and minutes, and not the owner's own match

#### Scenario: Score as broadcast
- **WHEN** another match of the matchday is at minute 63 with 1-2 and its stored result is 3-2
- **THEN** its row reads "1 - 2" and "63'", never "3 - 2"

#### Scenario: Cup tie live
- **WHEN** the owner's live match is a cup tie or a friendly
- **THEN** the hero shows no "Rest of the matchday" section and makes no live scores request

### Requirement: Rest of the matchday when there is nothing to list
The section SHALL be hidden when the division live scores list no other match of the same matchday, while the first response is loading, and when the request fails. The owner's own score, minute and "Watch live" button SHALL never depend on it.

#### Scenario: Endpoint down
- **WHEN** the division live scores request fails while the owner's league match is live
- **THEN** the hero shows the own match as before and no "Rest of the matchday" section

### Requirement: Rest of the matchday layout
On phone-width screens the section SHALL start collapsed behind a toggle that reads the summary "N matches · G goals", N being the listed matches and G the sum of their broadcast goals. On desktop it SHALL start expanded. Rows MUST fit at 360 px wide without horizontal scroll and stay readable on the hero's error-container surface.

#### Scenario: Phone summary
- **WHEN** an owner on a 360 px phone has a live league match and the other 7 matches have 12 goals between them
- **THEN** the hero shows a collapsed toggle reading "7 matches · 12 goals", and tapping it lists the 7 matches

#### Scenario: Desktop
- **WHEN** the same owner opens the dashboard on a 1280 px screen
- **THEN** the 7 matches are listed without tapping anything

#### Scenario: Spanish summary
- **WHEN** the language is Spanish and there is 1 other match with 0 goals
- **THEN** the toggle reads "1 partido · 0 goles"

### Requirement: Pending panel
The owner's dashboard SHALL show a Pending panel listing every pending item (see "Pending items") as one row with an icon, a text and one action, ordered error, warning, info, success, then by the order of "Pending items". With no pending item the panel MUST NOT be shown.

#### Scenario: Nothing pending
- **WHEN** an owner has nothing pending
- **THEN** the dashboard shows no Pending panel

#### Scenario: Ordering
- **WHEN** an owner is insolvent, has a received offer and has no trainer
- **THEN** the panel lists insolvency first, then the missing trainer, then the offer

### Requirement: Pending items
The panel SHALL include, when they apply: insolvent (error); insolvency within 4 weeks by the cash-flow forecast (warning); unavailable starters in the next fixture's lineup, with their projected replacements (warning); contracts with 1 season or less left (warning); no trainer hired (warning); ticket price 3 or more away from the recommended price (info); offers received (info); friendly challenges received (info); an academy prospect ready (success).

#### Scenario: Injured starter for the next match
- **WHEN** the lineup for the next league match has an injured left-back whose projected replacement is B
- **THEN** the panel says one starter of that match is unavailable, names B as his replacement and links to the lineup editor

#### Scenario: Injured reserve
- **WHEN** only a player outside the next fixture's XI is injured
- **THEN** no unavailable-starters item is shown

#### Scenario: Forecast
- **WHEN** the cash-flow forecast reaches −maxDebt in 3 weeks
- **THEN** a warning item links to the finance page

#### Scenario: Ticket price
- **WHEN** the ticket price is 15 and the recommended price is 11
- **THEN** an info item shows both prices and opens the ticket price dialog

#### Scenario: No trainer
- **WHEN** the club has no trainer
- **THEN** a warning item links to the coaching tab of the transfer market

### Requirement: Snoozing pending items
Every item except an error SHALL offer "Remind me tomorrow", which hides it in that browser until the next local midnight. A snoozed item MUST come back earlier if its content changes (for example a second offer arrives).

#### Scenario: Snoozed offers
- **WHEN** an owner snoozes "1 offer received" and later a second offer arrives the same day
- **THEN** the item "2 offers received" is shown again

#### Scenario: Insolvency
- **WHEN** an owner is insolvent
- **THEN** the insolvency item offers no snooze

### Requirement: Pending panel on phones
On a phone-width screen the panel SHALL show at most the 3 first items and a "Show all (N)" toggle when there are more.

#### Scenario: Five items on a phone
- **WHEN** an owner with 5 pending items opens the dashboard on a phone
- **THEN** 3 items and "Show all (5)" are shown, and the toggle reveals the other 2

## Code map

- Dashboard: `frontend/src/features/dashboard/DashboardPage.tsx` (`selectLiveMatch`, `useOwnLiveMatch`,
  `selectLastResult`, `LiveMatchHero` and its "Rest of the matchday" section `RestOfMatchday`, `NextMatchHero`,
  `LastResultCard`).
- Rest of the matchday: `frontend/src/components/DivisionLiveScores.tsx` (collapsible list, the
  "N matches · G goals" summary), `utils/liveScoreLabel.ts` (state chip: minute, half-time, FT, kickoff),
  `api/hooks.ts` (`useDivisionLiveScores`, `divisionLiveScoresRefetchInterval`) over
  `GET /v1/teams/division/:id/live` (see the `division-live-scores` spec).
- Pending panel: `frontend/src/features/dashboard/PendingPanel.tsx` (data wiring, snooze buttons, phone
  "Show all (N)"), `features/dashboard/pendingItems.ts` (`buildPendingItems`, the severity order and the
  `FORECAST_WARNING_WEEKS` / `TICKET_PRICE_GAP` / `CONTRACT_EXPIRY_WARNING_YEARS` thresholds),
  `features/dashboard/pendingSnooze.ts` (localStorage snooze until local midnight, content key),
  `features/finances/cashFlowForecast.ts` (`computeCashFlowForecast`) and the `?ticketPrice=1` deep link
  that makes `StadiumCard` (`features/finances/FinancesPage.tsx`) open its price dialog.
- Live broadcast: `frontend/src/api/useLiveMatch.ts` (`useLiveMatch`, `mightStillBeLive`, `isPast`).
- Shared helpers: `frontend/src/utils/matchSummary.ts` (`liveMatchPath`), `utils/liveChip.ts` (pulsing LIVE
  style), `theme.ts` (`errorContainer` / `onErrorContainer` tokens).
- Strings: `dashboard.pending.*`, `dashboard.liveNow`, `dashboard.watchLive`, `dashboard.liveMatchMinute`, `match.halftime`,
  `liveScores.*` in `i18n/locales/{en,es}.json`.
