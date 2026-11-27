# Spec Delta

## MODIFIED Requirements

### Requirement: Lineup and formation
Every CPU team SHALL re-pick its best XI from the current squad whenever it acts and again at the kickoff of each of its matches, ranking the six formations by the projected expected goal difference of each formation's best XI against the same squad's best 4-4-2 XI: a Manager or Director takes the first, a Caretaker the third or fourth at random; a Director then re-picks against its opponent at competitive kickoffs (see "Director formation against the opponent"). A Manager or Director MUST pick only from players available for that match; a Caretaker picks from the whole squad.

#### Scenario: New signing
- **WHEN** a CPU team signs a better striker
- **THEN** its next lineup refresh can include him

#### Scenario: Injured starter, Manager
- **WHEN** a Manager's best striker is injured at kickoff of a league match
- **THEN** its formation and XI are chosen from the squad without him, so a different formation can win

#### Scenario: Injured starter, Caretaker
- **WHEN** a Caretaker's best striker is injured at kickoff
- **THEN** its XI still names him and the match-day backfill replaces him with the best available player for that position

#### Scenario: Suspended player in a friendly
- **WHEN** a Manager's player serving a suspension is picked for a friendly
- **THEN** he may play, because suspensions only bar league and cup matches

#### Scenario: Manager's shape
- **WHEN** a Manager's best 3-4-3 XI projects a higher expected goal difference against its own best 4-4-2 XI than any other formation's best XI
- **THEN** it plays 3-4-3

#### Scenario: Ranking ties
- **WHEN** two formations project the same expected goal difference within 0.000000001
- **THEN** the one whose XI has the higher summed slot value ranks first, and after that the earlier one in the order 4-4-2, 5-3-2, 3-4-3, 4-3-3, 4-5-1, 3-5-2


#### Scenario: Best XI by optimal assignment
- **WHEN** a squad's only flank players are a wide midfielder (Lateral, Midfield and Pass 75, the rest 25) and a full-back (Lateral and Defense 60, the rest 40), alongside a keeper, two centre-backs and weak reserves
- **THEN** the full-back plays full-back (slot value 57) and the wide midfielder wide midfield (60), because the XI is the assignment of players to slots with the highest total slot value
- **AND** the slot-by-slot pick, which reaches the back line first, would have put the wide midfielder at full-back (58) and a reserve in wide midfield

### Requirement: Answering transfer offers
Every CPU team SHALL accept an offer, highest first, when the fee reaches its threshold: its perceived value (market value with level noise: 30% / 15% / 5%), raised for key starters (×2 Manager, ×2.5 Director) and internationals (Director +25% up to +50% at 20 caps). Sales that leave the squad or the position too thin, or of a player it signed within the last 14 days, MUST be refused. A Manager or Director counters an offer of at least 70% of its threshold (see transfer-negotiation) instead of rejecting it.

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

#### Scenario: Recent signing
- **WHEN** a CPU team receives an offer of ten times his value for a player it signed 3 days ago
- **THEN** it rejects the offer at every AI level and does not counter it, the same 14-day hold that keeps him off its transfer list
