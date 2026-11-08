# Spec Delta

## ADDED Requirements

### Requirement: Contract length in valuations
Every CPU valuation of a player (asking prices, offer thresholds, bids and surplus loan fees) SHALL multiply his market value by a contract factor: 0.6 with 1 season or less left, 0.85 with 2 seasons and 1.0 with 3.

#### Scenario: Last season
- **WHEN** a player's market value is 500,000 and he has 1 season left
- **THEN** a CPU club lists him at 300,000 and, as a Director, bids from 300,000 × 1.1

#### Scenario: Fresh contract
- **WHEN** the same player has 3 seasons left
- **THEN** his value stays 500,000

### Requirement: CPU personal terms
A CPU offer or renewal SHALL propose exactly the player's expected wage (with the wage floor) and his age's maximum term, so a CPU club never makes an offer the player refuses on personal terms.

#### Scenario: Overpaid target
- **WHEN** a Director bids for a player earning 3,000 whose market wage is 2,500
- **THEN** its offer carries a wage of 3,000
