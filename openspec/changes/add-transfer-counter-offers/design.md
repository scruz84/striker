# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- `transfer_offers` holds pending offers only (deleted when resolved), unique per
  (player, buyer). `MakeTransferOffer` checks the personal terms with the player up front;
  `AcceptOffer` locks the offer, the player and the buyer's budget `FOR UPDATE`, checks the
  negotiation window, moves the money (`TRANSACTION_TRANSFER_IN/OUT` via `AdjustTeamBudget`) and
  the player, commits, then notifies (`NOTIFICATION_TRANSFER`). `RejectOffer` deletes and notifies
  `NOTIFICATION_OFFER_DECLINED`.
- CPU sellers answer in `engine/aiteams.go` `respondToReceivedOffers` with
  `common.EvaluateIncomingOffer`, which already returns the `Threshold` and a seeded decision.
- CPU buyers bid in `engine/aitransfers.go` (`makeAiOffer`, valuation × bump, cap 1.5 ×
  valuation) and record rejections in `offer_rejections` (`RecordOfferRejection`,
  `OfferRetryAllowed`, 72 h cooldown, +10% rebid).
- The offer-response job (`engine/tasks.go` `OfferResponseJob`) runs on every replica.

## Goals / Non-Goals

**Goals:**
- Accepting a counter is the same atomic transfer as accepting an offer, with one fee changed.
- Every state change is a single guarded statement or happens under the offer row lock, so two
  replicas or two clicks can never double a counter, an acceptance or an expiry.

**Non-Goals:**
- An offer history table; a resolved offer still disappears as today.

## Decisions

### D1. Storage
Migration `resources/sql/init98_offer_counters.sql` (renumber if needed): `counter_amount bigint
NULL`, `countered_at timestamptz NULL` on `transfer_offers`, plus an index on `countered_at`
where not null. `TransferOffer` gains `CounterAmount *int64`, `CounteredAt *time.Time` and
`CounterExpiresAt *time.Time` (`countered_at + CounterOfferLifetime`, `72 h`).

### D2. Operations (`common/offers.go`)
- `CounterOffer(offerId, toTeamId, amount)`: `UPDATE transfer_offers SET counter_amount = $3,
  countered_at = now() WHERE offer_id = $1 AND to_team_id = $2 AND counter_amount IS NULL AND
  $3 > amount AND $3 <= amount * 10` — zero rows → `ErrOfferNotCounterable`. Requires the window
  open. Notifies the buyer (`NOTIFICATION_COUNTER_OFFER`) after the update.
- `AcceptCounterOffer(offerId, fromTeamId)`: the body of `AcceptOffer` extracted into
  `executeOfferTx(tx, offer, fee)`; this entry point locks the offer, requires
  `counter_amount IS NOT NULL` and `from_team_id = caller`, skips the negotiation window, and
  passes `counter_amount` as the fee. Solvency is checked like `MakeTransferOffer`.
- `AcceptOffer` refuses a countered offer (`ErrOfferCountered`).
- `DeclineCounterOffer(offerId, fromTeamId)`: guarded `DELETE … RETURNING`, notifies the seller.
- `ExpireCounterOffers(now)`: `DELETE FROM transfer_offers WHERE countered_at < $1 - interval
  '72 hours' RETURNING …`; each returned row notifies both clubs. Called from the offer-response
  job; the `DELETE … RETURNING` hands each row to exactly one replica.
- New errors map to 400/409 in `handlers/offerHandlers.go`.

### D3. CPU seller
In `respondToReceivedOffers`, when `decision.Accept` is false, the reason is `below value` /
`key player` / `international` (never `squad too small` / `thin position`), the profile is Manager
or Director and `offer.Amount >= 0.70 × decision.Threshold`, call `CounterOffer` with the
threshold rounded up to the next 1,000 (`CpuCounterMinShare = 0.70`, `CpuCounterRounding =
1000`). Already-countered offers are skipped. Seeded noise keeps the threshold stable.

### D4. CPU buyer
New `answerCountersAsBuyer(teamId)` in `RespondToAllPendingOffers` for CPU teams with countered
sent offers: re-value the player with its profile (seeded by the offer id), accept when
`counter <= 1.5 × valuation` and the level's affordability check passes (`common.ScoreTarget`
with `Price = counter`), else decline and `RecordOfferRejection(player, team, originalAmount)`.

### D5. Notifications and frontend
`NOTIFICATION_COUNTER_OFFER = 15` (`common/fmconstants.go`), key
`notifications.messages.counterOffer` (params `player`, `playerId`, `team`, `amount`, `counter`);
seller-side keys `notifications.messages.counterDeclined` and `counterExpired` under
`NOTIFICATION_TRANSFER`; Spanish templates in `common/notificationtranslations.go`. Frontend: a
`CounterOfferDialog` (amount field with `TransferComparables`) on received offers; countered sent
offers show the counter, `counterExpiresAt` and Accept / Decline; hooks in `api/hooks.ts`. Keys
`transfers.offers.counter*` in en/es.

## Risks / Trade-offs

- [Humans counter every CPU offer to squeeze more] → a CPU buyer only goes to 1.5 × its valuation,
  the same ceiling as its own bids.
- [CPU counters flood a human's inbox] → one counter per offer, and CPU offers to humans are
  already capped at 3 per player.
- [Extracting `executeOfferTx` changes `AcceptOffer`] → covered by the existing offer tests, run
  unchanged before adding new ones.

## Migration Plan

Additive migration; existing offers have no counter and behave as today. Rollback: redeploy the
previous service and engine; countered offers then look like plain pending offers.
