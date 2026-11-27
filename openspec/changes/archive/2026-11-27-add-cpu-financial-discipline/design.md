# Design

## Context

`RunAITeamManagement` (`engine/aiteams.go`) runs, in order, `renewExpiringContracts`,
`manageRosterIfManager` (surplus listings, loan listings, academy, ticket price, sponsor deal,
Manager trainer), `runAiTransferPass` and `manageMarketIfDirector`. Offers are answered by
`respondToReceivedOffers` through `common.EvaluateIncomingOffer`.

## Decisions

### One crisis predicate
`inFinancialCrisis(budget int64) bool` (budget < `aiCrisisBudget` = 0) is read once per pass from
`common.GetTeamBudget` into an `aiFinances` struct (`budget`, `crisis`) that the pass hands to
the steps that need it. Below 0 rather than at the insolvency line so a club stops spending
while it can still recover before the lockout.

### Crisis listing
`pickCrisisSale(owned, commitments, protected, keyPlayers)` is pure: highest wage, ties to the
lowest id, skipping listed/loan-listed/protected/key players (`common.IsKeyPlayer` over the owned
squad), and requiring `positionDepth(owned unlisted without him) >= MinPositionDepth` and owned
unlisted squad − 1 ≥ `MinSquadSizeAfterSale`. Price: `ListingAskingPrice × aiCrisisPriceShare`
(0.8). Listed through `common.CreateTransferListing`; the counter
`striker.ai.market_actions{action="listed_in_crisis"}` and the `ai-market` log line record it.
The existing surplus-listing withdrawal (`planListingWithdrawals`) can still pull a crisis
listing back if the position later turns thin, which is the right outcome.

### Offer discount
`OfferContext.Distressed` multiplies the threshold by `CrisisOfferShare` (0.8) after the key
player and international premiums, before the comparison, in `EvaluateIncomingOffer`. The
counter path (`cpuCounterAmount`) reads `decision.Threshold`, so it uses the discounted figure
without change. `respondToReceivedOffers` reads the budget once.

### Selective renewals
`contractsToRenew(aiLevel, players, starters, crisis)` gains the keep rule; `starters` (default
lineup) is now read for every level. Depth is judged on the owned squad (loanees excluded).
`renewContract(…, starter bool)` only retries on the player's terms when `starter` or
`offer.ExpectedWage <= aiMaxRenewalRaise × skill wage` (1.25).

### Releasing unsold veterans
`pickVeteranToRelease(owned, listedSince, now, budget)` is pure; `listedSince` merges
`common.GetTeamTransferListings` (`ListedAt`) with a new `common.GetTeamLoanListingTimes`
(`GetTeamLoanListingTimesStmtSql`). The listing is withdrawn with `CancelTransferListing` /
`CancelLoanListing`, then `common.ReleasePlayer` pays the severance under its own lock and budget
check. Action label `released_veteran`.

### Order of a pass
`renewExpiringContracts` → `manageRosterIfManager` (now also `listForFundsIfInCrisis` and
`releaseUnsoldVeteranIfManager`) → `runAiTransferPass` (returns at once in crisis) →
`manageMarketIfDirector`.

## Risks / Trade-offs

- Fewer renewals mean more free agents at season end; the free-agent market absorbs them and
  humans get the chance to sign them.
- Crisis sales at 80% make debt-ridden CPU clubs a bargain source for humans; that is intended.

## Migration

None. Releases reuse `TRANSACTION_CONTRACT_RELEASE`; no notification is added (the CPU club has
no owner to notify).
