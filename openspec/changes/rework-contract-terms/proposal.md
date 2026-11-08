# Proposal

## Why

A review of transfers and renewals found that contracts carry almost no weight in the game:

- **Free wage cuts.** A renewal is accepted whenever the wage reaches the player's market wage, at
  any time and whatever he earns today. A human can renew an overpaid or declining player with 3
  seasons left at his lower market wage, and the player accepts. A transfer offer works the same
  way: a player earning 20,000 joins a new club for 12,000 if that is his market wage.
- **Inconsistent expectations.** Renewals and free-agent signings judge the wage with the player's
  intelligence and skill trend (`ExpectedWageWithIntelligence`,
  `MaxAcceptableRenewalYearsWithIntelligence`); a transfer offer's personal terms
  (`MakeTransferOffer`) use the plain skill wage and age term, so an improving player asks a raise
  from his own club but not from a buyer.
- **Contract length has no price.** `MarketValue` ignores `contract_years_remaining`: a player in
  the last season of his contract costs a CPU buyer, and is sold by a CPU club, at the same price
  as one with 3 seasons left, although he can leave for nothing a few weeks later.
- **The CPU loses players for nothing.** A Manager or Director that decides not to renew a bench
  player (see cpu-team-management "Contract renewals") keeps him until he leaves as a free agent,
  instead of trying to sell him while he still has a value. A Director never renews early, so its
  improving key players reach their last season and ask for the raise their trend gives them.

## What Changes

- **Wage floor**: a player's expected wage in a renewal is the higher of his market wage and
  100% of his current wage while he has 2 or more seasons left, 85% of it in his last season. In
  a transfer offer it is the higher of his market wage and 100% of his current wage. Free agents
  have no floor (they have no current wage to protect).
- **One rule for personal terms**: a transfer offer's personal terms are judged with the same
  intelligence- and trend-aware expectations as a renewal, plus the floor above.
- **Contract length in valuations**: every CPU valuation (`MarketValue`, so listings, thresholds,
  bids, loan fees) is multiplied by 0.6 with 1 season left, 0.85 with 2 and 1.0 with 3.
- **CPU sells what it will not renew**: a Manager or Director lists, at most one per pass, a
  player in his last season it does not renew, at his (already discounted) asking price, while
  the position keeps its minimum depth.
- **Director early renewals**: a Director renews its key players and its starters under 24 when
  they have 2 seasons left, at the player's own terms if he declines its first offer.
- **CPU offers meet the floor**: CPU offers and renewals start from the player's expected wage
  including the floor, so a CPU club never makes an offer the player is certain to refuse.

Out of scope:
- Contract lengths beyond 3 seasons, signing bonuses or loyalty bonuses.
- Playing-time or club-stature demands (a player asking more to stay at a small club).
- Any change to Caretakers beyond the valuation factor (they stay inattentive by design).

Open questions for the owner:
- Are 0.6 / 0.85 / 1.0 the right contract factors? They make a last-season player a bargain on
  purpose, so the market moves before contracts expire. Tune with `ECON_SIM=1`.
- Should the floor in the last season be 85% (a veteran accepts a small cut to stay) or 100%?
- Contract years are owner-only today (`Player.RedactFinancials`). The CPU already reads them for
  its own valuations; should humans see at least "last season" on other clubs' players? This
  change keeps them hidden; add-pre-contracts proposes showing a "last season" marker.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `team-finances`: "Contract renewal and release" gains the wage floor; new "Personal terms of a
  transfer offer" requirement.
- `cpu-transfer-market`: new "Contract length in valuations" and "CPU personal terms"
  requirements.
- `cpu-team-management`: new "Selling contracts it will not renew" and "Director early renewals"
  requirements.

## Impact

- Backend: `common/contracts.go` (`RenewalExpectedWage`, floor in `MakeContractRenewalOffer`),
  `common/offers.go` (`MakeTransferOffer` personal terms), `common/transferscoring.go`
  (`contractValueFactor` in `MarketValue`), `engine/aiteams.go` (`renewContract`, early
  renewals, selling unrenewed players), `engine/aitransfers.go` (`makeAiOffer` wage).
- Frontend: the renewal and offer dialogs already show the expected wage the backend returns; one
  new hint string explains the floor.
- No schema change, no new TRANSACTION_* reason, no new notification.
- Economy: CPU sale prices of expiring players drop; check with `ECON_SIM=1`.
