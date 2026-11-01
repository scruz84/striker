# Proposal

## Why

CPU teams ignore the loan market entirely. No engine code calls `CreateLoanListing`, `AcceptLoan`,
`GetActiveLoanListings` or `ExerciseLoanBuyOption`; the AI only *avoids* loan state (it skips
loaned-in and loan-listed players). So a human's loan listing never finds a CPU borrower, CPU
youngsters rot on the bench instead of earning the minutes that drive development, and CPU surplus
can only leave by permanent sale. Loans should be a third way to reinforce and to shed players,
weighed with the same scoring as transfers.

## What Changes

- **Borrowing.** The transfer pass of a Manager and a Director also reads the loan market. A loan
  listing becomes a candidate next to free agents, listed and unlisted players, scored by the same
  `ScoreTarget` (best-XI points he adds now, thin-position bonus, value for money), with the loan fee
  as its price, only the borrower's share of the wage on the payroll, no growth projection and a
  bonus when it carries a fair buy option. The best candidate wins whatever the market it comes from.
- **Limits on borrowing.** At most 2 loaned-in players per CPU team; loaned-in players count towards
  the `MaxAiSquadSize` cap; accepting a loan uses one of the pass's moves.
- **Buy options.** In the last season of a loan, the transfer pass scores each loaned-in player that
  has a buy option as a permanent signing at the option fee (against the squad he would leave) and
  exercises it when he clears the bar.
- **Lending.** Manager and Director roster upkeep may loan players out, at most one new listing
  per pass, 2 open loan listings and 3 players out on loan at once:
  - *Development loan* (Director only): a 21-or-younger player outside the best XI whose position
    keeps its minimum depth without him — 1 season, no fee, lender keeps 50% of the wage, no option.
  - *Surplus loan* (Manager and Director): a surplus player transfer-listed for 7 days without a
    buyer is withdrawn from the transfer list and offered on loan — 1 season, fee 10% of his market
    value, no wage contribution, buy option at his listing asking price.
  - Only players whose contract outlives the loan by at least a season, never internationals, never
    signed in the last 14 days. Loan listings that stop being surplus are withdrawn.
- **Observability.** New `striker.ai.market_actions` actions: `loaned_in`, `listed_for_loan`,
  `loan_listing_withdrawn`, `exercised_buy_option`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cpu-transfer-market`: the pass searches the loan market, scores loan candidates and buy options,
  and counts loans against the squad cap and move limits; new observability actions.
- `cpu-team-management`: roster upkeep loans players out (development and surplus loans) and
  withdraws loan listings that stop being surplus; "Same rules as humans" covers loans.

## Impact

- Backend: `engine/aitransfers.go` (new candidate kinds `candidateLoan`, `candidateBuyOption`,
  gathering from `GetActiveLoanListings`, executing via `AcceptLoan` / `ExerciseLoanBuyOption`),
  `engine/aiteams.go` (loan-out planning next to `listSurplusPlayer`), `common/transferscoring.go`
  (`TargetContext` wage share and loan scoring), `common/aimanager.go` (profile flags), constants
  for the new limits. No migration, no new `TRANSACTION_*` reason, no new notification key:
  existing `AcceptLoan` / `ExerciseLoanBuyOption` ledger rows and notifications are reused.
- Humans: CPU loan listings appear in the existing loan market; a human's loan listing can be taken
  by a CPU team (the human gets the existing `loanedOut` notification). No frontend change.
- Economy: CPU loan fees and wage contributions are new money flows; checked with the economy sim.

## Out of scope

- Loan *offers* or negotiation: the loan market stays listing-based, a borrower accepts the listed
  terms or not.
- Caretaker teams: they still never shop or lend.
- Recalling a loaned-out player early, and loaning a player out to relieve an insolvent budget.
- Amortized transfer fees (`amortize-transfer-fees`) — buy options stay a single payment there too.

## Open questions for the owner

1. Should the development loan also apply to Managers (who do not sign projects), or stay a
   Director trait? Proposed: Director only.
2. Surplus loan fee of 10% of market value per season and the 7-day "unsold" wait — acceptable, or
   should CPU surplus loans be free to make them more attractive to humans?
3. Should a CPU team ever borrow from another CPU team, or only from humans' listings? Proposed:
   from anyone, like the transfer list.
