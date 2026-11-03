# Spec Delta

## MODIFIED Requirements

### Requirement: Answering transfer offers
Every CPU team SHALL accept an offer, highest first, when the fee reaches its threshold: its perceived value (market value with level noise: 30% / 15% / 5%), raised for key starters (×2 Manager, ×2.5 Director) and internationals (Director +25% up to +50% at 20 caps). Sales that leave the squad or the position too thin MUST be refused. A Manager or Director counters an offer of at least 70% of its threshold (see transfer-negotiation) instead of rejecting it.

#### Scenario: Thin position
- **WHEN** selling would leave the squad under its size minus 2 or the player's position under its minimum depth
- **THEN** the offer is rejected at every AI level

#### Scenario: Stable answer
- **WHEN** the same offer is evaluated on several sweeps
- **THEN** its noise is seeded from the offer id, so the answer does not change

#### Scenario: Negotiation window
- **WHEN** an acceptable offer is still inside its negotiation window
- **THEN** it is left pending

#### Scenario: Caretaker and potential
- **WHEN** a Caretaker values a teenager
- **THEN** it ignores his hidden potential

#### Scenario: Near-miss offer to a Director
- **WHEN** a Director's threshold for a player is 1,234,500 and it receives an offer of 1,000,000
- **THEN** it counters with 1,235,000 (the threshold rounded up to the next 1,000)

#### Scenario: Low offer
- **WHEN** a Manager receives an offer below 70% of its threshold
- **THEN** it rejects the offer

#### Scenario: Caretaker near miss
- **WHEN** a Caretaker receives an offer at 90% of its threshold
- **THEN** it rejects the offer

#### Scenario: Thin position near miss
- **WHEN** an offer at 95% of the threshold would leave the player's position too thin
- **THEN** it is rejected, never countered
