# Spec Delta

## ADDED Requirements

### Requirement: Answering counter-offers
On its next offer-response sweep a CPU buyer SHALL accept a counter-offer when the counter fee is at most 1.5 × its valuation of the player and still passes its affordability check, and decline it otherwise. A declined counter MUST start the usual rejection cooldown for that player, with the original bid as the base for the next one.

#### Scenario: Acceptable counter
- **WHEN** a Director valued a player at 1,000,000, bid 1,100,000 and is countered at 1,300,000 with enough budget
- **THEN** it accepts and the transfer completes at 1,300,000

#### Scenario: Counter too high
- **WHEN** the same Director is countered at 1,600,000
- **THEN** it declines, and it cannot bid for him again for 3 days, then only at 1,210,000 or more
