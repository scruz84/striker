# Proposal

## Why

A transfer fee leaves the buyer's budget in a single ledger line, so one big signing can push a
club straight towards the insolvency threshold, and the finance page's cash-flow forecast never
sees the cost again. Now that every player has a contract length, the fee can follow it: the
buyer pays it over the seasons the player is actually under contract, which smooths budgets and
makes long contracts a real financial commitment. This is the last open item of the old economy
roadmap's Phase 4 ("transfer economics depth"); loan money and competing-offer grouping already
shipped. Status: proposed, not started.

## What Changes

- Buying a listed player (`ExecuteTransfer`) or accepting an offer (`AcceptOffer`) no longer
  debits the whole fee at once. The fee becomes a schedule of installments spread over the
  player's new contract length.
- A new table (e.g. `transfer_amortization`) records each schedule: buyer, seller, player, total
  fee, remaining amount and remaining installments.
- Installments are charged on the same per-round cadence as wages (`PayMatchdayWages`), each as
  its own ledger entry with a new `TRANSACTION_*` reason, so the cash-flow forecast picks them up
  as a recurring expense.
- Free-agent signings, loans and loan buy options are unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `team-finances`: the "Transfers" requirement changes from a single upfront payment to an
  installment schedule over the buyer's contract length.

## Impact

- Backend: `common/transfers.go`, `common/offers.go`, `engine/wages.go`, a new migration
  (`resources/sql/initN_transfer_amortization.sql`), new `TRANSACTION_*` reason in
  `common/fmconstants.go`; CPU affordability checks (`common.ScoreTarget`) should consider the
  installment rather than the full fee.
- Frontend: finance page labels for the new reason; the forecast's recurring-expense bucket.
- Balance: re-run `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v`.

## Open questions

1. **Seller side**: does the seller also receive the fee in installments (keeps transfers
   zero-sum at every moment), or in full upfront (money is created until the buyer finishes
   paying)? Default proposal: installments on both sides.
2. **Early exit**: what happens to the remaining installments if the player is sold on, released
   or retires before the schedule ends? Default proposal: the remainder falls due immediately.
3. **Insolvency**: should a schedule's remaining amount count against `MaxDebt` when deciding
   whether a club may buy again?
