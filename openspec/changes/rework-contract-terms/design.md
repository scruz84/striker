# Design

## Context

- Personal terms are judged in three places: `MakeContractRenewalOffer` and `SignFreeAgent`
  (`common/contracts.go`, with `ExpectedWageWithIntelligence` /
  `MaxAcceptableRenewalYearsWithIntelligence`) and `MakeTransferOffer` (`common/offers.go`, with
  the plain `PlayerWageFromSkills` / `MaxAcceptableRenewalYears`). None reads the current wage.
- `MarketValue(p)` (`common/transferscoring.go`) is skill wage × 20 × age factor and feeds
  `ListingAskingPrice`, `PerceivedValue` (thresholds and bids) and the surplus loan fee
  (`engine/ailoans.go`).
- CPU renewals: `renewExpiringContracts` → `contractsToRenew` → `renewContract`
  (`engine/aiteams.go`), with `shouldMeetRenewalTerms` capping the retry at 1.25 × the skill wage.
  CPU offers: `makeAiOffer` (`engine/aitransfers.go`) offers `PlayerWageFromSkills`.

## Decisions

### D1. One expectation function
`common.PersonalTermsExpectation(p Player, intelligence, trend int, kind TermsKind) (wage, years
int)` with `TermsKind` = `TermsRenewal`, `TermsTransfer`, `TermsFreeAgent`. It computes the
current intelligence/trend figures and applies the floor:
`wage = max(ExpectedWageWithIntelligence(...), ceil(p.Wage × floorShare))` where `floorShare` is
`RenewalWageFloorShare` (1.0) with `p.ContractYearsRemaining >= 2`, `LastSeasonWageFloorShare`
(0.85) with 1 or less, `TransferWageFloorShare` (1.0) for a transfer and 0 for a free agent. The
three call sites use it, so the rule lives in one pure, unit-tested function.

### D2. Contract factor
`contractValueFactor(years int) float64`: ≤1 → 0.6, 2 → 0.85, ≥3 → 1.0 (a free agent, years 0,
is never valued through `MarketValue` for a fee). `MarketValue` multiplies by it after the age
factor. `perceivePlayer` copies the player struct, so the factor reads the real contract length
(the club knows its own players' contracts; the CPU already reads other clubs' contracts today).

### D3. Selling unrenewed players
`pickUnrenewedToList(owned, renewed, commitments, protected)` is pure: owned players with
`ContractYearsRemaining <= 1` not returned by `contractsToRenew`, not listed or loan-listed, not
protected (`protectedFromListing`: recent signing or international), whose position keeps
`MinPositionDepth` and squad `MinSquadSizeAfterSale` without him; highest market value first,
ties to the lowest id. Listed through `common.CreateTransferListing` at `ListingAskingPrice`
(which now carries the 0.6 factor). Runs in `manageRosterIfManager` right after the surplus
listing, at most one per pass; action label `listed_expiring`. The listing withdrawal
(`planListingWithdrawals`) already pulls it back if the position later turns thin.

### D4. Director early renewals
`contractsToRenewEarly(owned, starters)` is pure: for a Director only, players with exactly 2
seasons left that are `common.IsKeyPlayer` over the owned squad or a starter under
`ProjectionMaxAge` (24). Not in crisis. `renewContract` is reused, with `starter = true` so the
retry on the player's own terms is always made.

### D5. CPU personal terms
`renewContract` and `makeAiOffer` start from `PersonalTermsExpectation` instead of
`PlayerWageFromSkills`, so the first offer is the accepted one; `shouldMeetRenewalTerms` keeps
comparing the expected wage against 1.25 × the skill wage, so an overpaid bench player whose
floor is far above his skill wage is let go (and listed by D3).

## Risks / Trade-offs

- Humans lose the free wage cut; a club with an overpaid squad must now sell or release (10 ×
  wage) to reduce wages. That is the intended cost of a bad contract.
- Cheaper last-season players move more often; the free-agent pool gets fewer good players.
- `MarketValue` changes shift CPU prices globally; the economy simulation must stay within its
  current bands.

## Migration

None. No TRANSACTION_* reason or notification is added; listings use the existing ledger path.
