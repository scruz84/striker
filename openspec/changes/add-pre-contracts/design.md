# Design

## Context

- Contracts tick down at each country's rollover in `engine.ProgressPlayers` →
  `common.AdvancePlayerContract`, under the country's rollover advisory lock (one replica).
  Retirement is checked first (`ShouldRetire` → `RetirePlayer`).
- Offers (`common/offers.go`) are judged on personal terms at creation, wait for a negotiation
  window and are answered by CPU clubs from `OfferResponseJob` (`engine/tasks.go`), claimed per
  slot with `claimRun`, so one replica runs each sweep.
- League progress per club: `common.GetTeamSeasonSituation` (`Played`, `Remaining`).
- This change builds on rework-contract-terms: the player's expected wage comes from
  `common.PersonalTermsExpectation(…, TermsTransfer)`.

## Decisions

### D1. Storage
`resources/sql/init104_pre_contracts.sql` (renumber if taken):
`pre_contract_offers (offer_id bigserial PRIMARY KEY, player_id bigint NOT NULL REFERENCES players
ON DELETE CASCADE, from_team_id bigint NOT NULL REFERENCES teams ON DELETE CASCADE, wage int NOT
NULL, years int NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), signed_at timestamptz)`,
`UNIQUE (player_id, from_team_id)`, a partial unique index `ON (player_id) WHERE signed_at IS NOT
NULL` (at most one signed pre-contract per player, enforced by the database) and an index on
`from_team_id`. `CREATE … IF NOT EXISTS` throughout.

### D2. Making an offer
`common.MakePreContractOffer(dbCtx, playerId, fromTeamId, wage, years)`: window open, buyer
solvent and not a national team, player owned by another club, `ContractYearsRemaining == 1`,
not on loan, no signed pre-contract, his club's league at least `PreContractSeasonShare` (0.5)
played, the buyer below `MaxSignedPreContracts` (3; CPU `AiMaxSignedPreContracts` = 2) signed
pre-contracts. The terms must reach `PreContractMovePremium` (1.10) × the expected wage and at
most the age's maximum term; like `MakeTransferOffer`, a refusal creates nothing and reports the
expectation. The owning club is notified with the buyer, wage, years and decision time (first
pending offer + 72 h).

### D3. Deciding
`common.DecidePreContracts(dbCtx, now)` runs from `OfferResponseJob` after
`RespondToAllPendingOffers`. For each player whose oldest pending offer is at least
`PreContractDecisionDelay` (72 h) old it opens a transaction, locks the player row
(`GetPlayerOwnershipForUpdateStmtSql`), re-checks D2's player conditions (a renewal in the
meantime raised his years above 1), re-judges each pending offer against his current expectation,
signs the best one (`UPDATE … SET signed_at = now()`), deletes the others, commits and notifies
all parties. Renewing (`MakeContractRenewalOffer`) and selling (`executeOfferTx`,
`ExecuteTransfer`, `AcceptLoan`) delete pending pre-contract offers on the player inside their own
transaction. Every decision is under the player's row lock, so a concurrent renewal and a
decision serialise.

### D3b. Counter-offer by renewal
`common.PreContractCounterBar(tx, playerId) int` = `ceil(best pending wage ÷
PreContractMovePremium)`, 0 with no pending offer, read under the player lock in
`MakeContractRenewalOffer`; the renewal expectation becomes `max(PersonalTermsExpectation(…,
TermsRenewal), bar)`, so the declined-renewal response reports the wage that keeps him. The
decision time is fixed by the first pending offer; a later, better offer raises the bar and
notifies the owner again. `DecidePreContracts` also sends the owner, once per player, the
`preContractDecisionSoon` reminder when the decision is 24 h or less away
(`pre_contract_offers.reminded_at` on the first offer row, set in the same statement that selects
it, so two replicas cannot both send it).

### D4. A signed player is frozen
`MakeContractRenewalOffer`, `ReleasePlayer`, `CreateTransferListing`, `CreateLoanListing` and
`MakeTransferOffer` refuse a player with a signed pre-contract (`ErrPlayerPreContracted`).
`IsPlayerPreContracted(tx, playerId)` is read under the existing locks.

### D5. Completing
`AdvancePlayerContract` becomes transactional: it locks the player, decrements the contract and,
when it reaches 0 and a signed pre-contract exists, moves the player to the buyer
(`movePlayerToTeam`, `nextFreeShirtNumber`, lineup rows deleted, `ApplyRenewalOfferStmtSql` with
the agreed wage and years, `RecordPlayerClubHistory`, `RecordPlayerTransfer` with kind
`PLAYER_TRANSFER_PRE_CONTRACT` and fee 0), deletes the pre-contract row and, after the commit,
notifies both clubs and `NotifyWatchers(…, WatchEventMoved, …)`. With no pre-contract it releases
him as today. `RetirePlayer` deletes his pre-contract offers. A buyer reset or deleted cascades
away its offers.

### D6. CPU buyer
`common.GetExpiringCandidatesForPosition` mirrors `GetMarketCandidatesForPosition` with
`contract_years_remaining = 1`, not on loan, no signed pre-contract. A Director takes the best 10
per core position, a Manager the best 5. They become `candidatePreContract` in
`runAiTransferPass`, scored with `ScoreTarget` at price 0 and the offered wage as payroll, using
the usual minimum score; signed pre-contracts count toward `MaxAiSquadSize`. The wage offered is
`ceil(1.10 × expected)`. At most 1 pre-contract offer per pass (counted as one of a Director's 2
new offers); at most 3 CPU pre-contract offers pending on one human's player, 1 on a CPU club's.
Action labels `pre_contract_offered`, `pre_contract_signed`.

### D7. CPU seller
`answerPreContractOffers(team)` in `RespondToAllPendingOffers`: for each own player with a pending
offer that `contractsToRenew` keeps, a Manager or Director renews him at his raised expectation
(read from the declined first offer, as `renewContract` already does) when
`acceptsCounterBar(expected, skillWage, starter, key, level)` holds: ≤ `aiCounterStarterCap`
(1.5) × skill wage for a starter, ≤ `aiCounterKeyPlayerCap` (2.0) for a Director's key player,
≤ `aiMaxRenewalRaise` (1.25) otherwise. Otherwise, or for a player it would not keep, it lists him
through rework-contract-terms' `pickUnrenewedToList` rule if it applies. A Caretaker does nothing
beyond its normal pass. Action labels `pre_contract_countered`, `pre_contract_let_go`.

### D8. Notifications
`NOTIFICATION_PRE_CONTRACT` (17). Keys, each with a Spanish template in
`common/notificationtranslations.go`: `notifications.messages.preContractOffered` (to the owning
club: buyer, player, wage, years, decision time and the wage that keeps him),
`preContractDecisionSoon` (to the owning club, 24 h before the decision), `preContractSigned` (to the buyer), `preContractSignedAway` (to the
owning club), `preContractRejected` (to a losing buyer), `preContractCancelled` (to every pending
buyer when the player is renewed or sold), `preContractJoined` (to both clubs at the move).

### D9. API and frontend
`POST /v1/players/:id/pre-contract` `{wage, years}`, `DELETE /v1/pre-contracts/:id` (pending only),
`GET /v1/user/team/pre-contracts` (sent and received). `Player.ContractExpiring` (`json:
"contractExpiring"`, true when owned and in his last season) is never redacted. Frontend:
"Last season" chip on player rows, an "Expiring contracts" filter on the Transfers tab search,
`PreContractDialog` (wage and years, the 1.10 hint), a Pre-contracts section in the Offers tab,
and a "Signed for <club>" chip in the squad list.

## Risks / Trade-offs

- Humans who forget renewals lose players for nothing; the dashboard already warns about
  contracts with 1 season left, the offer notification arrives 72 hours before he signs with the
  wage that keeps him, and a reminder follows 24 hours before.
- A CPU Director can take a human's star in his last season. That is the counter-play the human
  controls by renewing in time.

## Migration

`init104_pre_contracts.sql`. No TRANSACTION_* reason: no money moves.
