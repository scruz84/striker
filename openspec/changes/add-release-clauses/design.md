# Design

## Context

- Contract terms are set by `MakeContractRenewalOffer`, `SignFreeAgent` (`common/contracts.go`),
  `executeOfferTx` with the offer's personal terms (`common/offers.go`) and, with
  add-pre-contracts, the pre-contract completion. rework-contract-terms puts the expectation in
  `common.PersonalTermsExpectation`.
- Transfers follow "lock FOR UPDATE, validate inside the lock, both ledger rows, commit, notify",
  e.g. `ExecuteTransfer` (`common/transfers.go`).
- CPU candidates: `runAiTransferPass` builds `candidateListed`, `candidateFreeAgent`,
  `candidateLoan`, `candidateOffer` … and executes the best through `executeTransferMove`
  (`engine/aitransfers.go`).

## Decisions

### D1. Storage
`init105_release_clauses.sql` (renumber if taken): `ALTER TABLE players ADD COLUMN IF NOT EXISTS
release_clause bigint`, `release_clause_set_at timestamptz`; `ALTER TABLE transfer_offers ADD
COLUMN IF NOT EXISTS offered_release_clause bigint`; `ALTER TABLE IF EXISTS pre_contract_offers
ADD COLUMN IF NOT EXISTS release_clause bigint`. NULL means no clause. Every statement that sets a
contract (`SignFreeAgentStmtSql`, `ApplyRenewalOfferStmtSql`) also sets or clears both columns;
`ReleaseFromTeamStmtSql` clears them.

### D2. Minimum and discount
`ClauseBaseValue(p)` = `MarketValue` with the 3-season contract factor (1.0).
`MinReleaseClauseMultiple` = 1.5. `ReleaseClauseDiscount(clause, base)`: ≤ 2 × base → 0.10, ≤ 4 ×
→ 0.05, else 0. `PersonalTermsExpectation` gains an optional clause and returns
`ceil(expected × (1 − discount))`. A clause below the minimum is a validation error.

### D3. Paying
`common.PayReleaseClause(dbCtx, playerId, buyerTeamId, wage, years, newClause)`: window open,
buyer solvent; in one transaction locks the player (`GetPlayerOwnershipForUpdateStmtSql` extended
with the clause columns), checks a clause is set, `release_clause_set_at` ≥ 14 days
(`ReleaseClauseLockIn`) ago, not on loan, not pre-contracted, owner ≠ buyer, personal terms meet
`PersonalTermsExpectation(TermsTransfer, newClause)`; locks the buyer's budget, debits the
clause, credits the seller, moves the player like `executeOfferTx` (shirt number, lineup rows,
club history, `RecordPlayerTransfer` kind `PLAYER_TRANSFER_RELEASE_CLAUSE`), deletes his offers,
listings and pre-contract offers, writes `TRANSACTION_TRANSFER_OUT` / `TRANSACTION_TRANSFER_IN`,
commits, then notifies (`notifications.messages.releaseClausePaid` to the seller,
`releaseClauseSigned` to the buyer, Spanish templates added) and the watchers (`WatchEventMoved`).

### D4. CPU seller
`aiRenewalClause(profile, valuation, starter, key, international) *int64`: Caretaker nil; Manager
`2 × valuation` for a non-starter, nil otherwise; Director `4 × valuation` unless key or
international. Raised to the minimum when below it. `renewContract` passes it, and the expected
wage it offers is the discounted one.

### D5. CPU buyer
`candidateReleaseClause` for a Director over the players it already scans (core positions,
youngsters, radar) whose clause is past its lock-in (`payableClauses`), added next to the usual
bid candidate for the same player and ruled out when the clause is above `aiMaxBidValueMultiple`
(1.5) × its valuation, scored with `ScoreTarget` at price = clause. Counts as one of its 3 signings, not as a new offer; at most one
per pass. Rate limit: `common.LastCpuClausePaymentOn(sellerTeamId)` from `player_transfers` (kind
release clause, buyer unowned) must be older than 7 days for a human seller. Action label
`paid_release_clause`.

### D6. API and frontend
Renewal, free-agent, offer and pre-contract bodies accept `releaseClause` (optional; the common
functions gain `...WithClause` variants and the old names pass nil); `POST /v1/release-clauses`
`{playerId, wage, years, releaseClause?}`. `Player.ReleaseClause` and `ReleaseClausePayableAt` are
public, read through `playerColumnsSql`. Frontend: `ContractTermsFields` gains the optional
clause field (with the discount bands as a hint) in the renewal, free-agent, offer and
pre-contract dialogs; the player page shows the clause and, once payable, a "Pay release clause"
button opening `PayReleaseClauseDialog`.

## Risks / Trade-offs

- A cheap clause is a real loss risk for a human; the 14-day lock-in and the 7-day CPU rate limit
  keep it from becoming a raid.
- Directors gain a market humans cannot block; that is the point of the clause.

## Migration

`init105_release_clauses.sql`. No new TRANSACTION_* reason or notification type.

## Code map (for the new `release-clauses` spec when the change is archived)

- Core: `backend/common/releaseclauses.go` (`ClauseBaseValue`, `MinReleaseClause`,
  `ReleaseClauseDiscount`, `DiscountedWage`, `ValidateReleaseClause`, `PayReleaseClause`,
  `CpuMayPayClauseOn`, `ReleaseClauseLockIn`); storage `resources/sql/init105_release_clauses.sql`
  (`players.release_clause`, `release_clause_set_at`, `transfer_offers.offered_release_clause`,
  `pre_contract_offers.release_clause`).
- Contracts: `ApplyRenewalOfferStmtSql` / `SignFreeAgentStmtSql` set the clause,
  `ReleaseFromTeamStmtSql` clears it; `MakeContractRenewalOfferWithClause`,
  `SignFreeAgentWithClause`, `MakeTransferOfferWithClause`, `MakePreContractOfferWithClause`;
  `Player.ReleaseClause` / `ReleaseClausePayableAt` read by `scanPlayer` (`playerColumnsSql`).
- API: `handlers/releaseClauseHandlers.go` (`POST /v1/release-clauses`), `releaseClause` in the
  renewal, free-agent, offer and pre-contract bodies.
- Notifications: `notifications.messages.releaseClausePaid` / `releaseClauseSigned`.
- Frontend: clause field in `components/ContractTermsFields.tsx` (`clauseValue` in
  `utils/contractTerms.ts`), `components/PayReleaseClauseDialog.tsx`, clause row and button in
  `components/PlayerProfileRail.tsx`, hook `usePayReleaseClause`, strings
  `transfers.releaseClause.*`, `team.contractDialog.releaseClause*`.
- Tests: `common/releaseclauses_test.go`, `engine/releaseclauses_integration_test.go`,
  `handlers/releaseClauseHandlers_test.go`, `components/PayReleaseClauseDialog.test.tsx`.
