# Spec Delta

## ADDED Requirements

### Requirement: Added minutes count as minutes played
A player's minutes in a match SHALL include every added minute he was on the pitch, and his per-minute zone contributions SHALL include those minutes. The result component keeps its 90-minute cap, and a player on the pitch for the whole match counts as playing the full match.

#### Scenario: Full match with added time
- **WHEN** a defender plays the whole of a 2-1 win with 2 and 5 added minutes
- **THEN** his record shows 97 minutes and his result component is +0.3, the same as for 90 minutes

#### Scenario: Substitute in added time
- **WHEN** a striker comes on through a substitution order at 90+1 of a match with 5 added minutes
- **THEN** he has 5 minutes and is unrated ("s.c.")

#### Scenario: Ratings as of an added minute
- **WHEN** a viewer scrubs a finished match to 45+2
- **THEN** each rating counts minutes, contributions, events and score up to and including 45+2
