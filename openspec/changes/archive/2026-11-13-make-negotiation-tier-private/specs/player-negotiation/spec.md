# Spec Delta

## ADDED Requirements

### Requirement: Private negotiation tier
A player's negotiation tier (Naive, Casual, Aware, Shrewd, Mastermind) SHALL stay private: no manager MUST see it, on the player's page or in any API response, whether or not his club has negotiated with the player.

#### Scenario: After a refused offer
- **WHEN** a manager's offer is refused on personal terms by a Shrewd player
- **THEN** his page shows no negotiator row and the player details carry no negotiation tier

#### Scenario: Never negotiated
- **WHEN** a manager who never negotiated with him opens the same page
- **THEN** no negotiator row is shown

## REMOVED Requirements

### Requirement: Negotiation insight
**Reason**: The tier is meant to be hidden; showing it after a negotiation gave it away.
**Migration**: None for managers; the `negotiation_insights` table is dropped by `init109_drop_negotiation_insights.sql`.
