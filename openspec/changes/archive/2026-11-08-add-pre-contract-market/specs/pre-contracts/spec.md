# Spec Delta

## ADDED Requirements

### Requirement: Pre-contract market
The transfer market SHALL list the best 100 players (by combined skills) of other clubs in the last season of their contract, not on loan and without a signed pre-contract, each with his club, best position, age, average skill, wage and clause, whether his league is far enough along for a pre-contract offer, and whether the viewer already made him one. His skills and contract years stay hidden as on any squad page.

#### Scenario: Not open yet
- **WHEN** a listed player's league has played 10 of its 30 rounds
- **THEN** his row says pre-contracts open at half the season and its Pre-contract action is disabled

#### Scenario: Own players
- **WHEN** a manager opens the list
- **THEN** none of his own players is in it

#### Scenario: Already offered
- **WHEN** the manager has a pending offer on a listed player
- **THEN** the row shows the offer as sent instead of a Pre-contract action
