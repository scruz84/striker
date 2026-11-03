# Spec Delta

## MODIFIED Requirements

### Requirement: Discipline
Each minute SHALL have a chance of a card for a random outfield player of either side: 3%, or 3.6% in a Rival match and 4.2% in a Clásico (league and cup only, see rivalries). 6% of first cards are straight reds, and a second yellow becomes a red. A sent-off player MUST leave the pitch unreplaced. Goalkeepers are never booked.

#### Scenario: Straight red in a league match
- **WHEN** a player gets a straight red in a league or cup match
- **THEN** he is removed for the rest of the match and banned for 2 matches

#### Scenario: Second yellow
- **WHEN** a player already booked is booked again in a league or cup match
- **THEN** he is sent off and banned for 1 match

#### Scenario: Red card in a friendly
- **WHEN** a player is sent off in a friendly
- **THEN** he leaves the match but carries no suspension

#### Scenario: Clásico
- **WHEN** a league match between Clásico rivals is simulated
- **THEN** each minute has a 4.2% chance of a card, about 3.8 cards per match instead of 2.7
