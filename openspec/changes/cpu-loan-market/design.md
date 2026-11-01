# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- The loan market is listing-based: `common.CreateLoanListing` (duration 1–3, fee, wage
  contribution %, buy-option fee), `common.AcceptLoan` (locks the listing and the borrower budget
  `FOR UPDATE`, writes `TRANSACTION_LOAN_FEE` on both sides, notifies `loanedIn` / `loanedOut`),
  `common.ExerciseLoanBuyOption` (`TRANSACTION_TRANSFER_OUT/IN`, notifies
  `loanBuyOptionExercised` / `loanBuyOptionUsed`), `common.GetActiveLoanListings`.
- The CPU transfer pass (`engine/aitransfers.go`, `runAiTransferPass`) gathers
  `transferCandidate`s of three kinds, ranks them with `common.ScoreTarget` and executes the best
  via `executeTransferMove`. Roster upkeep (`engine/aiteams.go`, `manageRosterIfManager` →
  `listSurplusPlayer`) lists surplus.
- `transferState.Squad` excludes loaned-in players; `available()` excludes transfer-listed only.
- A player on loan cannot be renewed (`ErrPlayerOnLoan`) and his contract expiry is deferred until
  the loan ends, so a lender can lose him at return if the contract ran out.
- `RunAITeamManagement` already runs inside the AI-activity job slot claim; per-row races are
  covered by the locks inside `AcceptLoan` / `ExerciseLoanBuyOption`.
- Development is driven by minutes played (`player-development`), so a benched youngster gains
  nothing; a loan is the only way to get him minutes.

## Goals / Non-Goals

**Goals:**
- One ranking: loans, buy options and transfers compete in the same `rankCandidates` list so the
  pass picks whichever is best, not a separate "loan quota".
- Reuse the human production functions only; no new SQL beyond what the existing helpers expose.

**Non-Goals:**
- Changing `AcceptLoan` / `CreateLoanListing` semantics, adding loan offers, or any frontend work.

## Decisions

### D1. Two new candidate kinds in the transfer pass
Add `candidateLoan` (carries `ListingId`, `SellerTeamId`, `LoanFee`, `WageContributionPercent`,
`BuyOptionFee`, `DurationSeasons`) and `candidateBuyOption` (a player currently on loan to the
team). `gatherCandidates` adds loan listings when `profile.SearchesLoanList` (new flag, true for
Manager and Director), skipping the team's own listings, players over 36 and players already seen.
Buy-option candidates come from `GetTeamMarketCommitments().LoanedIn` joined with the loan rows
(new `common.GetLoansToTeam(teamId)` returning `PlayerLoan` with `ReturnSeason` and
`BuyOptionFee`), kept only when `BuyOptionFee > 0` and `ReturnSeason - currentSeason <= 1`.
`executeTransferMove` calls `common.AcceptLoan` / `common.ExerciseLoanBuyOption`; their expected
errors (`ErrLoanListingNotFound`, `ErrTransferWindowClosed`, `ErrInsufficientBudget`,
`ErrTeamInsolvent`, `ErrCannotAcceptOwnLoan`, `ErrPlayerNotOnLoan`) are silent, like the others.
*Alternative:* a separate loan pass after the transfer pass. Rejected: it would sign a mediocre
permanent player first and then also take a better loanee, exactly the double-up the cap is for.

### D2. Scoring loans through `ScoreTarget`
Extend `TargetContext` with `Loan bool`, `LoanSeasons int`, `LoanWageContributionPercent int`
(the lender's share, 0 = the borrower pays it all) and `BuyOptionFee int64`. When `Loan`:
- payroll check uses `wage × (100 − LoanWageContributionPercent) / 100` (then the wage factor);
- `Projection` and `Reputation` are 0;
- value for money compares the fee with `Value × loanFairFeeShare (0.10) × LoanSeasons`
  (fee 0 → +4);
- `+loanBuyOptionBonus (2.0)`, reported as `TargetScore.BuyOption`, when `0 < BuyOptionFee <= Value`.
Buy-option candidates use the normal (non-loan) path with `Price = BuyOptionFee` and
`Squad = playing squad without him`, so the improvement is what the team loses if he leaves.
*Alternative:* a dedicated `ScoreLoan` function. Rejected: two scorers would drift; the shared one
keeps "same yardstick as transfers", which is the point of the change.

### D3. The squad a candidate is scored against
`transferState` gains `LoanedIn []*common.Player`. Loan candidates are scored against
`available() ∪ LoanedIn` (`playingSquad()`), so a team with a loaned-in striker does not borrow
another. Transfers keep scoring against owned players (unchanged spec). `available()` also drops
loan-listed players, since they are on their way out. The cap check in `pickNextTarget` becomes
`len(available()) + len(LoanedIn) >= MaxAiSquadSize`; loan candidates are additionally skipped at
`len(LoanedIn) >= AiMaxLoanedIn (2)`. `planSellToBuy` ignores loan candidates.

### D4. Loan-out planning in roster upkeep
New `manageLoanListings(dbCtx, team, players)` in `engine/ailoans.go`, called right after
`listSurplusPlayer` in `manageRosterIfManager`; new profile flags `LendsPlayers` (Manager,
Director) and `LoansOutYouth` (Director). It reads the commitments and the team's own transfer
listings fresh, then:
1. Withdraws loan listings that stopped being right (`planLoanListingWithdrawals`, pure): the
   player's position would be under its minimum depth, the unlisted squad under
   `minSquadSizeAfterSale`, or he is a starter of the best XI he would rejoin
   (`CancelLoanListing`, action `loan_listing_withdrawn`).
2. Returns if the window is closed or there is no room (`loanOutRoom`: fewer than 2 open loan
   listings and open listings + players already out below 3, so the cap of 3 holds even if every
   listing is taken).
3. Picks at most one player (`pickPlayerToLoanOut`, pure): first a surplus player transfer-listed
   for at least 7 days (longest-listed first), else - Director only - the youngest benched player
   aged <= 21 (lowest id breaks ties).
4. A surplus loan cancels the transfer listing, then `CreateLoanListing(1, round(MarketValue ×
   0.10), 0, oldAsking)`; if that fails the transfer listing is recreated at its old price so the
   player isn't lost from the market. A development loan is `CreateLoanListing(1, 0, 50, 0)`.
Eligibility (`loanOutEligible`): not recently signed (14 days) and not an international
(`recentlySignedOrInternational`, split out of `protectedFromListing`, which adds the youngster
protection of a project-signing team - that protection stops *selling*, not development loans),
contract >= duration + 1 seasons, and without him the unlisted squad keeps >= `minSquadSizeAfterSale`
players and his position its minimum depth. New SQL: `GetTeamLoanedOutPlayerIdsStmtSql`
(`TeamMarketCommitments.LoanedOut`), `GetTeamTransferListingsStmtSql` (price and age of the team's
own listings) and `GetLoansToTeamStmtSql`.

### D5. Constants live next to the existing ones
`AiMaxLoanedIn`, `AiMaxOpenLoanListings`, `AiMaxLoanedOut` in `common/offerrejections.go` beside
`AiMaxOpenOffers*`; `loanFairFeeShare`, `loanBuyOptionBonus` in `common/transferscoring.go`;
`aiSurplusLoanAfter`, `aiDevelopmentLoanMaxAge`, `aiSurplusLoanFeeShare (0.10)`,
`aiDevelopmentLoanWageContribution (50)`, `aiLoanDurationSeasons (1)` in `engine/ailoans.go`.

### D6. Persistence, money and notifications
- Migration: **none** (`loan_listings.listed_at`, `transfer_listings.listed_at` and all loan
  columns already exist).
- `TRANSACTION_*` reasons: **none new** — `TRANSACTION_LOAN_FEE`,
  `TRANSACTION_LOAN_WAGE_CONTRIBUTION`, `TRANSACTION_TRANSFER_OUT/IN` are reused via the existing
  functions.
- Notification keys: **none new** — `loanedIn`, `loanedOut`, `loanBuyOptionExercised`,
  `loanBuyOptionUsed`, `loanReturnedHome/Away` already have en/es templates; `NotifyTeamOwner` is a
  no-op for unowned teams.
- Replicas: no new engine job; the work runs inside the existing AI-activity slot claim, and
  `AcceptLoan` locks the listing row, so two CPU teams racing for one listing yield one loan.

## Risks / Trade-offs

- [CPU teams hoard each other's youngsters on free loans] → caps of 2 loaned-in / 3 loaned-out per
  team, and a loanee must clear the normal minimum score.
- [Surplus loans with a buy option undercut CPU transfer sales] → only after 7 unsold days, and
  the option price equals the original asking price.
- [Lender loses a player whose contract expires during the loan] → eligibility requires contract
  years >= duration + 1.
- [Development loanee is needed back after injuries] → no recall in scope; the position keeps its
  minimum depth by eligibility, and the academy promotion step still fills holes.
- [Economy drift from new fee/wage flows] → run `ECON_SIM=1 go test ./engine -run
  TestEconomySimulation -v` before and after and compare CPU budget medians; the sim does not model
  loans yet, so the AI transfer simulation integration test is extended to measure loan counts.

## Migration Plan

Pure engine behavior; deploy normally. Rollback = revert the commit: existing CPU loans simply run
their course and return through `ReturnExpiredLoans`.

## Verification notes

- **Economy check (task 4.2).** `ECON_SIM=1 go test ./engine -run TestEconomySimulation` models only
  gate, sponsorship, wages, stadium and trainer money; it contains no transfer or loan flow, so it
  cannot move with this change. Same seed, season-36 median team budget of the shipped scenario:
  `main` 638,769; this branch 629,709 / 638,581 / 638,212 / 634,278 across four runs - the sim
  itself varies ~1.5% between runs of identical code, more than any difference between the trees.
  Every division's income/expense ratios are unchanged within that noise and no median trends
  towards insolvency. The loan money flows are instead bounded by construction: a surplus loan fee
  is 10% of one player's market value (itself 20 × his wage × an age factor), a development loan
  is free, and each CPU team has at most 2 loanees in and 3 players out.
- **Simulation (task 4.1).** In the 12-day world of `TestAiTransferMarket_Simulation`, with two
  free 96-rated loan listings from the humans, CPU teams took 4 loans (including CPU-to-CPU) and
  every cap held on every day.
- **Not covered by a test:** the fallback that recreates a transfer listing when the loan listing
  fails right after the transfer listing was cancelled (the window is checked first, and the
  eligibility rules already exclude the other expected errors).
