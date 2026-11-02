# Proposal

## Why

Today the only tactical decision a manager makes is the formation (`common/formation.go`); the same
XI plays the same way whether it is defending a lead away at the leader or chasing a win at home
against the bottom side. A team mentality (defensive, balanced, attacking) is the simplest lever
that turns "who plays" into "how we play", gives a real trade-off (more goals for, more against,
more fatigue) and is the foundation for conditional match orders (`add-conditional-match-orders`),
which switch mentality mid-match.

## What Changes

- Every lineup (default, friendly default, per league match, per cup tie, per friendly) carries a
  **mentality**: Defensive, Balanced (default) or Attacking.
- Effects every minute, after the zone blend and the home advantage:
  - Attacking: the side's three attack ratings ×1.08 and its three defense ratings ×0.92;
    Defensive: the opposite. Midfield is unchanged.
  - Each Attacking side adds 1 point to the per-minute chance probability and each Defensive side
    removes 1 (an open match between two attacking sides, a dull one between two defensive sides).
  - Fatigue: the maximum fatigue penalty is 0.33 for an Attacking side and 0.27 for a Defensive
    one instead of 0.30.
- A Balanced side behaves exactly as today, so two Balanced sides play exactly today's match.
- CPU teams pick a mentality at kickoff by AI level from the strength gap with the opponent.
- The lineup editor gets a three-way selector; the match viewer's lineups show each side's
  mentality once the match has kicked off.

Out of scope:
- More than three levels, pressing, tempo, width or individual player instructions.
- Changing mentality during a match (that is `add-conditional-match-orders`).

Open questions for the owner:
- Are ±8% and ±1 point of chance probability the right size? Task 2.3 checks the effect against the
  spec's ranges; the owner may want a stronger lever.
- Should the opponent's mentality stay hidden until kickoff (the default: yes, like the lineup)?

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-simulation`: new "Team mentality" requirement; "Chance generation" and "In-match fatigue"
  depend on mentality.
- `cpu-team-management`: new "CPU mentality" requirement.
- `match-viewer`: new "Mentality in the lineups" requirement.

## Impact

- Backend: new migration (mentality columns), `common.Lineup.Mentality`, lineup getters/setters in
  `common/database.go`, `common/cups.go`, `common/friendlies.go`, kickoff snapshots,
  `engine/initial_simulation.go`, `engine/scorer_simulation.go`, `engine/stamina_simulation.go`,
  `common/ailineup.go` (CPU pick).
- API: optional `mentality` (−1, 0, 1; default 0) in every lineup GET/PUT payload, backwards
  compatible.
- Frontend: selector in `MatchLineupForm`, label in the viewer's `Lineups`; en/es strings.
