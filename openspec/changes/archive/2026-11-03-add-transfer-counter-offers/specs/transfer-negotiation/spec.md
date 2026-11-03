# Spec Delta

## Purpose

Lets the selling club answer a transfer offer with one higher counter fee and lets the buying club
accept or decline it, so a near-miss offer becomes a negotiation instead of a blind rejection.

## ADDED Requirements

### Requirement: Making a counter-offer
The receiving club of a pending transfer offer SHALL be able to counter it once with a fee strictly above the offered fee and at most 10 times it, while the transfer window is open. A countered offer MUST NOT be accepted at its original fee; it waits for the buyer.

#### Scenario: Counter
- **WHEN** a seller counters a 800,000 offer with 1,000,000
- **THEN** the offer shows a counter of 1,000,000 waiting for the buyer

#### Scenario: Counter not above the offer
- **WHEN** a seller counters a 800,000 offer with 800,000
- **THEN** the counter is rejected with a validation error

#### Scenario: Second counter
- **WHEN** a seller tries to counter an offer it already countered
- **THEN** the request fails and the first counter stands

### Requirement: Answering a counter-offer
The buying club SHALL be able to accept a counter, which completes the transfer at the counter fee with the offer's personal terms in one atomic transaction, or decline it, which ends the offer. Accepting MUST fail without any change when the window is closed, the buyer is insolvent or cannot afford the counter fee, or the player no longer belongs to the seller.

#### Scenario: Counter accepted
- **WHEN** the buyer accepts a 1,000,000 counter
- **THEN** 1,000,000 moves from the buyer to the seller, the player joins the buyer on the offered wage and years, and the offer is gone

#### Scenario: Buyer cannot afford it
- **WHEN** the buyer accepts a counter larger than his budget
- **THEN** it fails with insufficient budget and the counter stays open

#### Scenario: Counter declined
- **WHEN** the buyer declines the counter
- **THEN** the offer ends and the seller is told it was declined

### Requirement: Counter-offer expiry
A counter-offer that the buyer has not answered within 72 hours SHALL expire: the offer ends and both clubs are told. The expiry MUST happen once even when several engine replicas run it at the same time.

#### Scenario: Ignored counter
- **WHEN** 72 hours pass after a counter without an answer
- **THEN** the offer is removed and both owners get a notification

### Requirement: Counter-offer notification
A human buyer SHALL receive a `NOTIFICATION_COUNTER_OFFER` naming the player, the selling club, the offered fee and the counter fee, linking to his sent offers.

#### Scenario: CPU counters a human
- **WHEN** a CPU club counters a human's offer
- **THEN** the human gets the counter-offer notification with both fees

### Requirement: Counter-offer display
The seller's received offers SHALL offer a "Counter" action on every pending offer, with the player's recent comparable transfers as a reference, and show a countered offer as waiting for the buyer. The buyer's sent offers SHALL show a countered offer with its counter fee, its expiry time and "Accept" and "Decline" actions.

#### Scenario: Buyer's view
- **WHEN** a buyer opens his sent offers and one has been countered
- **THEN** it shows the counter fee, when it expires, and Accept and Decline buttons
