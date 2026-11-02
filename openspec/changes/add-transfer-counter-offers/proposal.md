# Proposal

## Why

A transfer offer between two clubs can only be accepted or rejected (`common/offers.go`:
`AcceptOffer` / `RejectOffer`). A seller who would sell for a bit more has no way to say so, and a
CPU club that rejects an offer just below its price gives the buyer no hint: the buyer has to guess,
wait out the cooldown and bid again. A single counter-offer round ("not for 800,000, but for
1,000,000") makes the market a negotiation, saves round trips and lets human managers haggle
with each other and with CPU clubs.

## What Changes

- The receiving club of a pending transfer offer can answer with a **counter-offer**: a higher
  fee. The offer then waits for the buyer, who can **accept** (the transfer completes at the
  counter fee, with the personal terms already agreed), **decline** (the offer ends) or let it
  **expire after 72 hours**.
- One counter per offer; the counter fee must be above the offered fee.
- CPU sellers (Manager and Director) counter instead of rejecting when the offer reaches at least
  70% of their threshold: the counter is their threshold rounded up to the next 1,000. A
  Caretaker and offers below 70% are rejected as today.
- CPU buyers answer a human's counter on their next offer-response sweep: they accept when the
  counter is within their bidding limit (≤ 1.5 × their valuation and affordable) and decline
  otherwise; a decline starts their usual rejection cooldown.
- New notification `NOTIFICATION_COUNTER_OFFER` to a human buyer when a counter arrives; existing
  transfer / declined notifications cover the rest.
- The Offers tab shows a "Counter" action on received offers (with the transfer comparables as a
  reference) and "Accept / Decline" on countered sent offers.

Out of scope:
- Contract renewals and free-agent signings: the player already reveals his expected wage and
  maximum term on every attempt (`ContractRenewalOffer`), so a counter step adds nothing there.
- Loans and multi-round haggling (counter to a counter).
- Counter-offers on personal terms (wage, years).

Open questions for the owner:
- Is one round enough, or should the buyer be able to counter back once more?
- Should the 72-hour expiry follow the admin-configurable negotiation window instead of a constant?

## Capabilities

### New Capabilities
- `transfer-negotiation`: the counter-offer round between a selling and a buying club, its
  timing, outcomes, notifications and display.

### Modified Capabilities
- `cpu-team-management`: "Answering transfer offers" — a CPU seller counters near-miss offers.
- `cpu-transfer-market`: new "Answering counter-offers" requirement for CPU buyers.

## Impact

- Backend: new migration (counter columns on `transfer_offers`), `common/offers.go`
  (`CounterOffer`, `AcceptCounterOffer`, `DeclineCounterOffer`, expiry), `engine/aiteams.go`
  (CPU seller counters, CPU buyer answers), a periodic expiry in the existing offer-response job.
- API: `POST /v1/offers/:id/counter`, `POST /v1/offers/:id/accept-counter`,
  `POST /v1/offers/:id/decline-counter`; `TransferOffer` gains `counterAmount`, `counteredAt`,
  `counterExpiresAt`.
- Frontend: Offers tab in `TransferMarketPage`, `TransferOfferDialog` reuse; en/es strings;
  Spanish notification template.
