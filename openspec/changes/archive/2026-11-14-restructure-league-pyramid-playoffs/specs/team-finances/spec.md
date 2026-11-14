# Spec Delta

## MODIFIED Requirements

### Requirement: Prize money
At season rollover each league SHALL split its tier's pool (500,000 / 200,000 / 50,000 for tiers 1 / 2 / 3, deeper tiers as tier 3) by final position, linearly from 2× the per-team average for 1st down to 0 for last. Each cup tie MUST pay its winner 5,000 × the round number, and the cup champion a further 100,000. A promotion playoff tie pays no prize of its own.

#### Scenario: Ten-team tier-1 league
- **WHEN** a ten-team tier-1 league ends
- **THEN** the champion receives 100,000 and the last team 0

#### Scenario: Cup round 3 win
- **WHEN** a team wins a round-3 cup tie
- **THEN** it receives 15,000

#### Scenario: Promotion playoff win
- **WHEN** a team wins its promotion playoff tie
- **THEN** it receives that match's gate and sponsorship and no cup round prize

#### Scenario: Tier-5 league
- **WHEN** a ten-team tier-5 league ends
- **THEN** it splits the 50,000 tier-3 pool

### Requirement: Season tickets and merchandising
At season rollover each team SHALL receive season tickets of capacity × 5 and a merchandising share of its tier's pool (100,000 / 40,000 / 10,000 for tiers 1 / 2 / 3, deeper tiers as tier 3) from 1.5× the per-team average for 1st down to 0.5× for last, both × the Fanbase commercial factor.

#### Scenario: Last place still sells shirts
- **WHEN** a team finishes last
- **THEN** it still receives half the average merchandising share
