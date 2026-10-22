# Tasks

## 1. Decide

- [ ] 1.1 Resolve the open questions in `proposal.md` (seller side, early exit, insolvency) and write `design.md`

## 2. Schema and ledger

- [ ] 2.1 Add the `transfer_amortization` table in a new re-runnable `initN_*.sql` migration
- [ ] 2.2 Add the installment `TRANSACTION_*` reason(s) in `common/fmconstants.go` and the frontend reason labels (en/es)

## 3. Transfers

- [ ] 3.1 Make `ExecuteTransfer` and `AcceptOffer` create a schedule instead of a single debit/credit, inside the same transaction
- [ ] 3.2 Handle a player leaving before the schedule ends, per the decision in 1.1

## 4. Charging

- [ ] 4.1 Charge one installment per active schedule from `PayMatchdayWages`
- [ ] 4.2 Make CPU affordability (`common.ScoreTarget`) and the cash-flow forecast account for installments

## 5. Verify

- [ ] 5.1 Unit tests for the schedule maths and the transfer/offer paths
- [ ] 5.2 Re-run the economy simulation and record the result
