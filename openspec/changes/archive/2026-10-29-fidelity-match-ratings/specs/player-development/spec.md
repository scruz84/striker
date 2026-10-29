# Spec Delta

## MODIFIED Requirements

### Requirement: Form drift
Every squad player's Form SHALL update after each match: a player who featured gets +2 plus round((match rating − 6.0) × 2); a squad player who did not feature loses 1. Form is clamped to [1, 99] and missing Form reads as 50.

The match rating used is the one stored for that match (see match-ratings), so a player who came on is adjusted by his own rating exactly like a starter. A player who featured but is unrated ("s.c.", under 10 minutes) gets only the +2.

#### Scenario: Poor performance
- **WHEN** a player features with a match rating of 4.5
- **THEN** his Form changes by +2 − 3 = −1

#### Scenario: Benched
- **WHEN** a squad player does not feature
- **THEN** his Form drops by 1

#### Scenario: Substitute with a good rating
- **WHEN** a player comes on in minute 60 and his stored match rating is 7.0
- **THEN** his Form changes by +2 + 2 = +4

#### Scenario: Unrated short appearance
- **WHEN** a player comes on in minute 85 and is unrated
- **THEN** his Form changes by +2
