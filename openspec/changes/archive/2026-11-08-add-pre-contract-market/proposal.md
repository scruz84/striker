# Proposal

## Why

Pre-contracts and release clauses shipped (archived add-pre-contracts and add-release-clauses)
with two gaps: a manager could only find players in their last season by browsing rival squads,
and a release clause only showed on the player page, not where managers compare players.

## What Changes

- **Pre-contract market**: a "Last year" tab on the transfer market lists the best 100 players of
  other clubs in the last season of their contract, not on loan and not already signed, with their
  club, position, age, rating, wage, clause, whether a pre-contract is open yet and whether the
  viewer already made one, and the "Pre-contract" and "Make offer" actions.
- **Contract facts on market rows**: every market card and last-season row shows the player's
  release clause and the "Last season" marker when they apply.

Out of scope:
- Filters or sorting on the Last year tab (it is ordered by skill).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pre-contracts`: new "Pre-contract market" requirement.
- `release-clauses`: new "Clause on market rows" requirement.

## Impact

- Backend: `common.GetPreContractMarket` (`common/precontracts.go`), `PreContractMarketHandler`,
  route `GET /v1/pre-contracts/market`. No schema change.
- Frontend: `features/transfers/LastSeasonTab.tsx`, `ContractMeta` in `marketUi.tsx`, hook
  `usePreContractMarket`, strings `transfers.lastSeason.*`, `transfers.tabsShort.lastSeason`,
  `transfers.releaseClause.meta`.
