# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- Scheduled substitutions are stored per lineup kind in `default_lineup_substitutions`,
  `lineup_substitutions`, `cup_tie_lineup_substitutions` and the friendly table, read by
  `GetSubstitutionsForMatch` / `GetCupTieLineupSubstitutions` / `GetFriendlyLineupSubstitutions`
  and applied in `SubstitutionSimulation` (`engine/substitution_simulation.go`,
  `applyScheduledSubstitutions`, `bestEligibleBenchPlayer`, `teamSide.substitutionsUsed`).
- Injury replacements share the same counter (`bringOnInjuryReplacement`).
- Events are `common.Event` with types 0–7 (`common/fmconstants.go`); the frontend renders them in
  `utils/matchCommentary.tsx` and `features/match/MatchEventsReport.tsx`, live through the
  WebSocket broadcast, which already paces events by minute.
- `add-team-mentality` gives `MatchSimulation.HomeMentality` / `AwayMentality`, read every minute
  by qualification, chance probability and fatigue.

## Goals / Non-Goals

**Goals:**
- Orders behave exactly like scheduled substitutions wherever a substitution is involved (same
  validation, same counter, same fallback), so there is one set of rules.
- Firing is deterministic given the match's random stream (no extra random draws), so re-running
  a simulation with the same seed gives the same result.

**Non-Goals:**
- Showing a team's orders to anyone but its owner (the stored snapshot keeps them, the viewer only
  shows their effects through events).

## Decisions

### D1. Storage mirrors substitutions
Migration `resources/sql/init96_lineup_orders.sql` (renumber if needed): tables
`default_lineup_orders (team_id, slot, …)`, `lineup_orders (match_id, team_id, slot, …)`,
`cup_tie_lineup_orders (tie_id, team_id, slot, …)` and `friendly_lineup_orders (match_id, team_id,
slot, …)`, each with `from_minute smallint`, `condition smallint`, `action smallint`,
`mentality smallint NULL`, `player_out_id bigint NULL`, `player_in_id bigint NULL`, primary key
on the owner columns + `slot`, `CHECK (slot BETWEEN 1 AND 3)`. Players reference `players` with
`ON DELETE CASCADE`, like the substitution tables (a deleted player takes his orders with him). Constants: `ORDER_CONDITION_LOSING = 0`,
`…_LOSING_BY_2 = 1`, `…_DRAWING = 2`, `…_WINNING = 3`, `…_WINNING_BY_2 = 4`;
`ORDER_ACTION_MENTALITY = 0`, `ORDER_ACTION_SUBSTITUTION = 1`; `MaxConditionalOrders = 3`.
`common.LineupOrder{FromMinute, Condition, Action, Mentality, PlayerOutId, PlayerInId}` and
`Lineup.Orders`. Each lineup kind's get/set reads and writes its orders in the same transaction
as its substitutions; the kickoff snapshot stores them too.
*Alternative:* one polymorphic `lineup_orders` table keyed by kind. Rejected: it breaks the
existing one-table-per-kind pattern and its foreign keys.

### D2. Validation
`common/lineupvalidation.go` gets `ValidateOrders(lineup, squad)`: at most 3, from-minute 1–89,
known condition/action, mentality −1..1 for mentality actions, and for substitution actions the
existing scheduled-substitution player checks (outgoing in the XI, incoming on the bench, both in
the squad). The same player may be the outgoing player of several orders with different
conditions; at runtime the first to fire wins and the others are spent.

### D3. Firing
`SubstitutionSimulation.Apply` calls `applyConditionalOrders(match, side)` after
`applyScheduledSubstitutions`. `teamSide` gains `orders []common.LineupOrder`, `ordersFired
[]bool` and a pointer to the side's mentality. The condition uses the goals before the current
minute's scorer stage (the stage order guarantees it). A mentality action sets the side's
mentality and appends a `MENTALITY_CHANGE` event (`Event.Mentality` new field, persisted in a new
`mentality` column of `match_events`, `cup_tie_events` and `friendly_events` in the same
migration); a substitution action reuses the scheduled-substitution helper.

### D4. CPU orders
`PlanAiMatchLineup` returns `Orders` for Manager and Director in competitive matches
(constants `aiManagerChaseMinute = 70`, `aiDirectorChaseMinute = 65`,
`aiDirectorProtectMinute = 75`); `planAiKickoffs` stores them with the CPU kickoff lineup and they
go through `ValidateOrders` like a human's.

### D5. Frontend
`OrdersEditor` (in `features/lineup/`, next to `SubstitutionsEditor`) with three rows: minute,
condition select, action select (mentality or substitution with the two player pickers reused
from `SubstitutionsEditor`). `useSubstitutionDrafts` pattern for unsaved changes. Chronicle text in
`utils/matchCommentary.tsx`. Keys `lineup.orders.*` and `match.commentary.mentalityChange` in
en/es.

### D6. Visibility and the friendly default (as implemented)
`Lineup.Orders` is filled only by the lineup GET handlers a team owner opens (`withOwnerOrders`),
like `ProjectedReplacements`; every other payload - the played match's lineups, the live
"lineups" message, the opponent preview - carries none. The friendly default lineup has no orders
table (proposal: out of scope), so `MatchLineupForm` shows the orders section through its own
`ordersEnabled` prop, set everywhere `substitutionsEnabled` is except the friendly-default tab, and
`SetFriendlyDefaultLineupHandler` ignores any orders sent. The editor shows the orders expanded,
like the substitutions planner, rather than collapsed.

## Risks / Trade-offs

- [Orders make the editor heavy on phones] → the section is collapsed by default and shows a
  one-line summary per order.
- [A human can't see why the CPU changed approach] → the chronicle event explains it.
- [Ordering bugs between scheduled substitutions and orders] → spec fixes the order (scheduled
  first, then orders by slot) and the tests pin it.

## Migration Plan

Additive migration; lineups without orders behave as today. Rollback: redeploy the previous
engine; the new tables and the event column are ignored (the frontend shows an unknown event type
as nothing, which is the current behavior for unknown types — verify in task 4.2).
