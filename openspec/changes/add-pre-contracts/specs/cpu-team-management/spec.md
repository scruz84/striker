# Spec Delta

## ADDED Requirements

### Requirement: Answering pre-contract offers
On its offer sweep a Manager or Director with a pending pre-contract offer on a player it would keep (see "Contract renewals") SHALL counter by renewing him at his raised expectation when that is at most 1.5 × his skill-based wage for a starter (2 × for a Director's key player) or 1.25 × for anyone else; otherwise it lists him under "Selling contracts it will not renew" when that rule allows it. A Caretaker MUST NOT react beyond its usual renewals.

#### Scenario: Manager keeps its starter
- **WHEN** a Manager's starter with a skill-based wage of 2,000 receives a pre-contract offer that raises his expectation to 2,800
- **THEN** it renews him at 2,800 on its next sweep and the offer is cancelled

#### Scenario: Rival outbids the ceiling
- **WHEN** the same starter's expectation rises to 3,200 (1.6 × his skill-based wage)
- **THEN** the Manager does not renew him and, if the rule allows it, lists him for transfer

#### Scenario: Director key player
- **WHEN** a Director's key player with a skill-based wage of 2,000 has an expectation of 3,800
- **THEN** it renews him at 3,800

#### Scenario: Director lets a bench veteran go
- **WHEN** a Director's 32-year-old bench player it does not renew receives an offer
- **THEN** it lists him for transfer instead of renewing him

#### Scenario: Caretaker bench player
- **WHEN** a Caretaker's bench player in his last season receives an offer
- **THEN** it does nothing, and after 72 hours he signs the pre-contract
