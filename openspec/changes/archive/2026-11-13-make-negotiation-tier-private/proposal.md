# Proposal

## Why

A player's negotiation tier (Naive .. Mastermind) is meant to be hidden: a manager should judge
how shrewd a player is from how he negotiates, not read it off his page. Today the page shows a
"Negotiator" row once the manager's club has negotiated with him, which gives the tier away.

## What Changes

- **The tier is private**: no manager ever sees a player's negotiation tier, before or after
  negotiating with him. The "Negotiator" row leaves the player page and `negotiationStyle`
  leaves `GET /v1/teams/player/:id/`.
- **Nothing records negotiations any more**: `negotiation_insights` and the code that filled
  and read it go; `init109_drop_negotiation_insights.sql` drops the table (`init106` is
  removed, so a new database never creates it).
- **FAQ**: the negotiation-intelligence answer says the tier is never revealed and no longer
  names the tiers.
- The tiers keep shaping what a player asks exactly as before; only their visibility changes.

## Impact

- Specs: `player-negotiation` (the "Negotiation insight" requirement is replaced).
- Backend: `common/playerintelligence.go`, `common/playerdetails.go`, `common/accounts.go`,
  the contract, offer, pre-contract and clause handlers, `handlers/playerHandlers.go`, SQL.
- Frontend: `components/PlayerProfileRail.tsx`, `api/types.ts`, the en/es locales.
