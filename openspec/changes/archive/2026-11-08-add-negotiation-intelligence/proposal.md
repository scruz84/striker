# Proposal

## Why

Every player has a hidden negotiation intelligence (Naive, Casual, Aware, Shrewd, Mastermind;
`common/playerintelligence.go`), but it only scales one thing: how much of his latest progression
run he turns into a raise or a longer term. That run is a single match, so one good game moves his
demands, and nothing about contracts, pre-contracts or clauses depends on how shrewd he is. A
manager never learns whom he is dealing with either: the tier is invisible.

## What Changes

- **Steadier trend**: the skill trend a negotiation reacts to is the average of the player's last
  5 progression runs (rounded), not the latest one.
- **Last-season floor by intelligence**: in the last season of his contract a player keeps 75%
  (Naive), 80% (Casual), 85% (Aware), 90% (Shrewd) or 95% (Mastermind) of his wage when he renews,
  instead of a flat 85%.
- **Pre-contract premium by intelligence**: to leave his club a player asks 1.05, 1.075, 1.10, 1.15
  or 1.20 × his expected wage by tier, and his club keeps him for the best offer divided by his own
  premium.
- **Clause discount by intelligence**: for a clause of at most 2 × his value a player gives up 5%,
  7.5%, 10%, 12.5% or 15% of his wage by tier, half of that for one of at most 4 ×.
- **Negotiation insight**: once a club has negotiated with a player (renewal, free-agent signing,
  transfer offer, pre-contract or clause payment, accepted or not), it sees his tier on his page
  as "Negotiator: Naive … Mastermind".

The CPU needs no new rule: it already offers exactly what each player reports he wants.

Out of scope:
- A player waiting longer for more offers, or comparing clubs by division or playing time.
- Showing the tier before any negotiation, or to other clubs.

Open questions for the owner:
- The tier tables are first values chosen to keep Aware (the middle tier) at today's numbers; tune
  them after a few seasons.

## Capabilities

### New Capabilities

- `player-negotiation`: the trend players react to and the negotiation insight.

### Modified Capabilities

- `team-finances`: "Contract renewal and release" takes the last-season floor from the tier.
- `pre-contracts`: "Player demands" and "Right to counter-offer" take the premium from the tier.
- `release-clauses`: "Wage discount for a clause" takes the discount from the tier.

## Impact

- Schema: `resources/sql/init106_negotiation_insights.sql` (`negotiation_insights`).
- Backend: `common/playerintelligence.go` (tier tables), `common/playerprogression.go`
  (`PlayerSkillTrend`), `common/contracts.go` (`wageFloorShare`), `common/precontracts.go`,
  `common/releaseclauses.go`, the negotiation functions (record the insight), player details.
- Frontend: "Negotiator" row on the player page, en/es strings.
- No TRANSACTION_* reason, no notification.
