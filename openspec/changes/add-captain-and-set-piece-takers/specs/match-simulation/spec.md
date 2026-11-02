# Spec Delta

## ADDED Requirements

### Requirement: Lineup roles
Every lineup SHALL be able to name a captain, a penalty taker and a free-kick taker, each optional ("automatic" when empty). Each named player MUST be in that lineup's XI or bench; the two takers MUST NOT be goalkeepers. A named player who later leaves the squad is treated as automatic.

#### Scenario: Roles saved
- **WHEN** a manager names his number 10 as free-kick taker and his centre-back as captain
- **THEN** both are stored with the lineup and the penalty taker stays automatic

#### Scenario: Goalkeeper as taker
- **WHEN** a manager names his goalkeeper as penalty taker
- **THEN** the lineup is rejected with a validation error

#### Scenario: Taker sold
- **WHEN** the named penalty taker is sold after the lineup was saved
- **THEN** the lineup's penalty taker is automatic

### Requirement: Captain's calming effect
When a card is drawn for an outfield player whose side has its captain on the pitch, and that player is not the captain, the card SHALL be cancelled with probability 10% and nothing is recorded. A side without a captain on the pitch MUST be booked exactly as today.

#### Scenario: Captain on the pitch
- **WHEN** a card is drawn for a midfielder whose captain is playing
- **THEN** with probability 10% no card is shown, and otherwise the card is shown as usual

#### Scenario: Captain booked
- **WHEN** the card is drawn for the captain himself
- **THEN** it is never cancelled

#### Scenario: Captain sent off
- **WHEN** the captain was sent off earlier in the match
- **THEN** his team-mates' cards are no longer cancelled

## MODIFIED Requirements

### Requirement: Penalty resolution
A penalty SHALL be taken by the attacking side's named penalty taker when he is on the pitch; otherwise by its on-pitch outfield player with the highest fatigue-adjusted Shot, ties going to the lowest player id. It is scored with probability `0.78 + 0.30 × (taker Shot − keeper GoalKeeper) / 99`, bounded to 60%–92%. It uses neither the open-play clear-cut roll nor the flank logic.

#### Scenario: Even taker and keeper
- **WHEN** the taker's Shot equals the opposing keeper's GoalKeeper
- **THEN** the penalty is scored with probability 78%

#### Scenario: Elite taker against a weak keeper
- **WHEN** the taker has Shot 80 and the keeper has GoalKeeper 30
- **THEN** the probability is 0.78 + 0.30 × 50/99 ≈ 0.93, bounded to 92%

#### Scenario: Weak taker against an elite keeper
- **WHEN** the taker has Shot 25 and the keeper has GoalKeeper 85
- **THEN** the probability is 0.78 − 0.30 × 60/99 ≈ 0.598, bounded to 60%

#### Scenario: Best finisher already sent off
- **WHEN** the side has no named taker and its highest-Shot player has been sent off, injured off or substituted
- **THEN** the best remaining on-pitch outfield player takes the penalty

#### Scenario: No outfield player left
- **WHEN** the attacking side has no outfield player on the pitch
- **THEN** the penalty is discarded, the same as an open-play chance

#### Scenario: Named taker on the pitch
- **WHEN** the named penalty taker has Shot 60 and a team-mate on the pitch has Shot 75
- **THEN** the named taker takes the penalty and it is scored with the probability for Shot 60

#### Scenario: Named taker on the bench
- **WHEN** the named penalty taker has not come on yet
- **THEN** the automatic rule picks the taker

### Requirement: Direct free-kick resolution
A direct free kick SHALL be taken by the attacking side's named free-kick taker when he is on the pitch; otherwise by its on-pitch outfield player with the highest `0.6 × Shot + 0.4 × Pass` (fatigue-adjusted), ties going to the lowest player id. It is scored with probability `0.06 + 0.10 × (taker value − keeper GoalKeeper) / 99`, bounded to 2%–15%.

#### Scenario: Even taker and keeper
- **WHEN** the taker's value equals the keeper's GoalKeeper
- **THEN** the free kick is scored with probability 6%, matching the real-world ~5–6% conversion

#### Scenario: Specialist taker
- **WHEN** the taker has Shot 75 and Pass 70 (value 73) and the keeper has GoalKeeper 40
- **THEN** the probability is 0.06 + 0.10 × 33/99 ≈ 0.093

#### Scenario: Bounds
- **WHEN** the taker value exceeds the keeper's GoalKeeper by 99, or trails it by 99
- **THEN** the probability is 15% and 2% respectively

#### Scenario: Named free-kick taker substituted off
- **WHEN** the named free-kick taker was substituted off in minute 60 and a free kick comes in minute 70
- **THEN** the automatic rule picks the taker
