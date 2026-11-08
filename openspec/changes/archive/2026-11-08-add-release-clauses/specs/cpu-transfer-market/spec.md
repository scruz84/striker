# Spec Delta

## ADDED Requirements

### Requirement: Paying release clauses
A Director SHALL treat a scanned player whose release clause is at most 1.5 × its valuation as buyable at the clause, scored like a listed player, and pay at most one clause per pass, counted as a signing. No CPU club MAY pay a clause on a human club's player when a CPU club paid one on that human club in the last 7 days.

#### Scenario: Cheap clause
- **WHEN** a Director values a player at 1,000,000, his clause is 1,400,000 and he clears its minimum score
- **THEN** it pays the clause and signs him

#### Scenario: Clause above its ceiling
- **WHEN** the clause is 1,600,000
- **THEN** it does not pay it

#### Scenario: Rate limit
- **WHEN** a CPU club paid a clause on a human club 3 days ago
- **THEN** no CPU club pays another clause on that human club's players
