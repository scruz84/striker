# Proposal

## Why

A match is simulated in one go and then replayed as if live (match-scheduling), so a manager can
never react while it is being played: his only in-match tool is a substitution at a fixed minute,
whatever the score. Real managers say "if we are losing at 60, go for it" or "if we are winning
at 75, close the game". Conditional orders let a manager write that plan before kickoff; the
engine follows it minute by minute. It gives real tactical depth without changing the
simulate-then-replay architecture, and it builds on team mentality (`add-team-mentality`).

## What Changes

- Every lineup with substitutions (the default, per league match, per cup tie, per friendly) can
  carry up to 3 **conditional orders**. Each order has:
  - a **from minute** (1–89);
  - a **score condition**: losing, losing by 2+, drawing, winning, winning by 2+;
  - an **action**: change mentality (to Defensive, Balanced or Attacking), or a substitution
    (player off → player on).
- An order is armed from its minute and fires once, at the first minute whose starting score
  meets its condition. A substitution order counts against the 3-substitution cap like any other
  and follows the same eligibility rules as a scheduled substitution.
- A fired mentality order is logged as a new match event, shown in the live chronicle and the
  report ("Team X switches to an attacking approach").
- CPU teams use orders by AI level: a Manager goes Attacking when losing from minute 70; a Director
  goes Attacking when losing from 65 and Defensive when winning from 75.
- The lineup editor gets an "Orders" section next to the scheduled substitutions.

Depends on: `add-team-mentality` (mentality actions and the mentality stored with the lineup).

Out of scope:
- Conditions other than the score (red cards, injuries, opponent's mentality, fatigue).
- Formation changes as an action.
- Orders for the friendly default lineup (it has no substitutions today; it keeps using none).

Open questions for the owner:
- Is 3 orders enough? (3 keeps the editor simple and matches the substitution cap.)
- Should an order be allowed to fire more than once (e.g. back to Balanced after equalising)? The
  default here is once; a second order with the opposite condition covers that.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-simulation`: new "Conditional match orders" and "Order firing" requirements; the
  "Per-minute simulation pipeline" evaluates orders in the substitution stage.
- `cpu-team-management`: new "CPU conditional orders" requirement.
- `match-viewer`: new "Mentality change in the chronicle" requirement.

## Impact

- Backend: new migration (orders tables), `common.Lineup.Orders`, getters/setters next to the
  substitution ones (`common/lineupsubstitutions.go`, `matchlineupsubstitutions.go`,
  `cuptielineupsubstitutions.go`, `friendlylineupsubstitutions.go`), validation, kickoff snapshot,
  `engine/substitution_simulation.go`, new event type `MENTALITY_CHANGE = 8`, `common/ailineup.go`.
- API: optional `orders` array in lineup GET/PUT payloads.
- Frontend: `OrdersEditor` in the lineup form, chronicle/report phrasing; en/es strings.
