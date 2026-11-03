# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- `AttentionAlerts` (`features/dashboard/DashboardPage.tsx`) builds a list of MUI `Alert`s from
  `finances.insolvent`, `players` (contract years), `useSquadUnavailability()`, `useAcademy()`,
  `useOffersReceived()` and `useFriendlyChallengesReceived()`, rendered in a `Stack`.
- Data the new items need already exists client-side: `useLineup()` returns the default lineup with
  `projectedReplacements`, and per-fixture lineups come from the match lineup hooks;
  `computeCashFlowForecast` (`features/finances/cashFlowForecast.ts`) gives the weeks until
  −maxDebt; `FinancesResponse.recommendedTicketPrice` and the stadium ticket price;
  `/v1/trainers/mine` gives the trainer (hook `useTeamTrainer`).
- The dashboard already has a phone breakpoint (`isDesktop` / `useMediaQuery`).

## Goals / Non-Goals

**Goals:**
- Item rules as pure functions (`pendingItems(input) → PendingItem[]`) so every scenario is a unit
  test, and the component only renders.

**Non-Goals:**
- Backend endpoints: everything is derived from data the dashboard already loads or that existing
  hooks expose.

## Decisions

### D1. Pure item builder
`features/dashboard/pendingItems.ts` exports `buildPendingItems(input)` returning
`{id, severity, textKey, params, action, contentKey}`; `contentKey` (e.g. `offers:2`) changes
whenever the item's content changes, which is what un-snoozes it. Thresholds as constants:
`FORECAST_WARNING_WEEKS = 4`, `TICKET_PRICE_GAP = 3`, `CONTRACT_EXPIRY_WARNING_YEARS = 1`
(moved from the page).

### D2. Next fixture's lineup
The next fixture comes from `useTeamMatches` (as `LineupEditorPage` does). If it has a per-fixture
override, its lineup hook is used, otherwise the default (or friendly default for a friendly);
the unavailable starters are the lineup's players who are in the squad's unavailability list
(suspensions ignored for a friendly), each paired with his entry in `projectedReplacements` when
the refill has one. The per-fixture lineup endpoints already fall back to the default lineup, so
one hook per kind (`useMatchLineup` / `useCupTieLineup` / `useFriendlyLineup`) is enough.

### D3. Snooze
`localStorage` key `striker.pending.snoozed` → `{[id]: {contentKey, until}}`, read and written in
`try/catch` (a failure just means no snooze). An item is hidden while `now < until` and the stored
`contentKey` equals the current one.

### D4. Rendering
`PendingPanel.tsx` renders a `Card` with a `List` (icon by severity, text, one `Button`); on phones
it slices to 3 with a "Show all (N)" toggle. Optional items for counter-offers and the watchlist are
added by those changes through the same builder (a feature flag is not needed: their hooks simply
do not exist before they ship). The ticket price item links to `/dashboard/finances?ticketPrice=1`, which makes `StadiumCard` open its existing price dialog (the dashboard has no dialog of its own). Keys `dashboard.pending.*` in en/es; existing `dashboard.*` keys
used by the old alerts are reused where the text is unchanged.

## Risks / Trade-offs

- [More requests on the dashboard (trainer, next-fixture lineup)] → both are small and cached by
  TanStack Query; the lineup is already fetched when the owner opens the editor.
- [Snooze hides something important] → errors cannot be snoozed, and any content change brings an
  item back.

## Migration Plan

Frontend-only; rollback is a redeploy of the previous frontend.
