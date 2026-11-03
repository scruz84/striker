# Proposal

## Why

The dashboard already has a row of notices for things that need the owner's action
(`AttentionAlerts` in `features/dashboard/DashboardPage.tsx`: insolvency, expiring contracts,
unavailable players, an academy prospect, received offers and friendly challenges), but it has
gaps and is not specified anywhere. It counts unavailable players in the whole squad instead of
the ones in the lineup that will actually be used next, it says nothing when there is no trainer,
when the budget is heading for insolvency or when the ticket price is far from the recommended one,
and on a phone six stacked alerts push the next match below the fold. This change turns the
notices into one specified "Pending" panel: fewer, more precise, actionable items, ordered by cost.

## What Changes

- The notices become a **Pending** panel: one row per item with an icon, a short text and a
  single action button, ordered by severity (error, warning, info, success). Nothing pending → no
  panel (as today).
- More precise item: **unavailable starters in the next fixture's lineup** (with the projected
  replacements already computed for the lineup editor) replaces the squad-wide unavailable count.
- New items:
  - **No trainer hired** → link to the coaching tab.
  - **Insolvency in sight**: the existing cash-flow forecast reaches −maxDebt within 4 weeks →
    link to finances.
  - **Ticket price far from the recommendation** (3 or more away) → opens the price dialog.
  - **Countered offers waiting for your answer** and **followed players newly available**, shown
    only once `add-transfer-counter-offers` / `add-player-watchlist` have shipped.
- Kept items: insolvent, expiring contracts, academy prospect, offers received, friendly
  challenges received.
- On phones the panel shows the 3 most severe items and a "Show all (N)" toggle.
- Each non-error item can be snoozed until the next day; the snooze is remembered per browser.

Out of scope:
- Server-side "to-do" storage or notifications for these items (the notification feed already
  covers events).
- New dashboard cards; the hero, KPIs and the other cards are unchanged.

Open questions for the owner:
- Are 4 weeks (forecast) and 3 (ticket price) the right thresholds?
- Should the squad-wide unavailable count stay as a low-priority item, or is the lineup-based one
  enough (the default here)?

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `team-dashboard`: new requirements "Pending panel", "Pending items" and "Pending panel on phones".

## Impact

- Frontend only: `features/dashboard/DashboardPage.tsx` (replace `AttentionAlerts` with a
  `PendingPanel` component in its own file), reuse of `useLineup` / projected replacements,
  `useTeamTrainer`, `computeCashFlowForecast`, the finance response's `recommendedTicketPrice`;
  en/es strings.
- No backend, migration or notification change.
