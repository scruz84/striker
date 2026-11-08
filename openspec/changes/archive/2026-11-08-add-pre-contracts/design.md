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
him as today. `RetirePlayer` deletes his pre-contract offers (the buyers are told first by
`LapsePreContractsOnRetirement`). A deleted buyer cascades away its offers; `ResetReleasedTeam`
deletes a reset club's.

### D6. CPU buyer
`common.GetExpiringCandidatesForPosition` mirrors `GetMarketCandidatesForPosition` with
`contract_years_remaining = 1`, not on loan, no signed pre-contract. A Director takes the best 10
per core position, a Manager the best 5. They become `candidatePreContract` in
`runAiTransferPass`, scored with `ScoreTarget` at price 0 and the offered wage as payroll, using
the usual minimum score; signed pre-contracts count toward `MaxAiSquadSize`. The wage offered is
`ceil(1.10 × expected)`. At most 1 pre-contract offer per pass (`aiPreContractOffersPerPass`;
it is not a signing, so it uses none of the pass's moves, and a failed one ends pre-contracts for
that pass), only to players whose league is past half its season; at most 3 CPU pre-contract offers pending on one human's
player, 1 on a CPU club's. Action label `pre_contract_offered`.

### D7. CPU seller
`answerPreContractOffers(team)` in `RespondToAllPendingOffers`: for each own player with a pending
offer, a Manager or Director runs `renewExpiringContracts` at once: `renewContract` reads the
raised expectation with `common.PlayerRenewalExpectation` and, for a contested player, renews
when
`acceptsCounterBar(expected, skillWage, starter, key, level)` holds: ≤ `aiCounterStarterCap`
(1.5) × skill wage for a starter, ≤ `aiCounterKeyPlayerCap` (2.0) for a Director's key player,
≤ `aiMaxRenewalRaise` (1.25) otherwise. Otherwise, or for a player it would not keep, it lists him
through rework-contract-terms' `pickUnrenewedToList` rule if it applies. A Caretaker does nothing
beyond its normal pass. Action label `pre_contract_countered`.

### D8. Notifications
`NOTIFICATION_PRE_CONTRACT` (17). Keys, each with a Spanish template in
`common/notificationtranslations.go`: `notifications.messages.preContractOffered` (to the owning
club: buyer, player, wage, years, decision time and the wage that keeps him),
`preContractDecisionSoon` (to the owning club, 24 h before the decision), `preContractSigned` (to the buyer), `preContractSignedAway` (to the
owning club), `preContractRejected` (to a losing buyer), `preContractCancelled` (to every pending
buyer when the player is renewed, sold, loaned, released or his contract runs out),
`preContractJoined` (to both clubs at the move), `preContractRetired` (to every buyer when he
retires).

### D9. API and frontend
`POST /v1/pre-contracts` `{playerId, wage, years}`, `DELETE /v1/pre-contracts/:id` (pending only),
`GET /v1/pre-contracts/received` and `/sent`. `Player.ContractExpiring` (`json:
"contractExpiring"`) is set by `RedactFinancials` when it hides another club's player's contract,
so the last season stays public. Frontend:
a "Last season" chip on another club's squad rows and player page, a "Pre-contract" button there
opening `PreContractDialog` (wage and years, the 1.10 hint), a Pre-contracts section in the Offers
tab (`features/transfers/PreContractCards.tsx`) and a "Leaving: <club>" chip in the own squad
list. A market-wide search of last-season players is left for later: managers find them on rival
squads, their watchlist and player pages.

## Risks / Trade-offs

- Humans who forget renewals lose players for nothing; the dashboard already warns about
  contracts with 1 season left, the offer notification arrives 72 hours before he signs with the
  wage that keeps him, and a reminder follows 24 hours before.
- A CPU Director can take a human's star in his last season. That is the counter-play the human
  controls by renewing in time.

## Migration

`init104_pre_contracts.sql`. No TRANSACTION_* reason: no money moves.

## Code map (for the new `pre-contracts` spec when the change is archived)

- Core: `backend/common/precontracts.go` (`MakePreContractOffer` / `MakePreContractOfferWithClause`,
  `WithdrawPreContractOffer`, `DecidePreContracts` with `remindPreContractDecisions` /
  `decidePreContract`, `PreContractCounterBar`, `PreContractMinimumWage`, `PreContractKeepWage`,
  `completePreContractTx`, `LapsePreContractsOnRetirement`, `cancelPendingPreContracts`, the
  `PreContract*` constants); storage `resources/sql/init104_pre_contracts.sql`.
- Wiring: `common.MakeContractRenewalOffer` (counter bar, cancellation), `ReleasePlayer`,
  `CreateTransferListing`, `CreateLoanListing`, `MakeTransferOffer` (refuse a signed player),
  `ExecuteTransfer`, `executeOfferTx`, `AcceptLoan` (cancel pending offers),
  `AdvancePlayerContract` (completion), `ResetReleasedTeam` (clears a reset club's offers),
  `Player.ContractExpiring` set by `RedactFinancials`; `engine/tasks.go` `OfferResponseJob`;
  `engine/progression.go` (retirement).
- API: `handlers/preContractHandlers.go`, routes `/v1/pre-contracts` in `service/fmservice.go`.
- Notifications: `NOTIFICATION_PRE_CONTRACT`, keys `notifications.messages.preContract*` with
  Spanish templates in `common/notificationtranslations.go`.
- Frontend: `components/PreContractDialog.tsx`, `features/transfers/PreContractCards.tsx` (Offers
  tab), `utils/contractTerms.ts` (`isLastSeason`), chips in `features/team/SquadList.tsx`, button in
  `components/PlayerProfileRail.tsx`; hooks `usePreContractsReceived`, `usePreContractsSent`,
  `useMakePreContractOffer`, `useWithdrawPreContractOffer`; strings `transfers.preContract.*`,
  `team.lastSeason*`, `team.signedFor*`.
- Tests: `common/precontracts_test.go`, `engine/precontracts_integration_test.go`,
  `handlers/preContractHandlers_test.go`, `features/transfers/PreContractCards.test.tsx`,
  `utils/contractTerms.test.ts`.
