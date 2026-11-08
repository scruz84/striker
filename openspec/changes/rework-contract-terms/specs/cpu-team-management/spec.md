# Spec Delta

## ADDED Requirements

### Requirement: Selling contracts it will not renew
A Manager or Director not in financial crisis SHALL, at most once per management pass, transfer-list one owned player in his last season whom it does not renew, at his asking price: the most valuable one who is not listed, loan-listed, an international or signed in the last 14 days, and whose sale keeps his position's minimum depth and the squad at size − 2.

#### Scenario: Bench veteran with one season left
- **WHEN** a Director does not renew a 31-year-old bench midfielder worth 200,000 with 1 season left
- **THEN** it lists him at 120,000 (contract factor 0.6) instead of waiting for him to leave free

#### Scenario: Nobody buys him
- **WHEN** the listing is still open at the season rollover
- **THEN** his contract expires and he leaves as a free agent, as before

#### Scenario: Caretaker
- **WHEN** a Caretaker does not renew a bench player
- **THEN** it does not list him

### Requirement: Director early renewals
A Director not in financial crisis SHALL renew, on its management pass, every owned key player and every starter under 24 who has exactly 2 seasons left, offering his expected wage at his age's maximum term and, if he declines, once more on his own terms.

#### Scenario: Rising star
- **WHEN** a Director's 21-year-old starter has 2 seasons left
- **THEN** it renews him to 3 seasons now, before his last season

#### Scenario: Bench player with two seasons
- **WHEN** a Director's 26-year-old bench player has 2 seasons left
- **THEN** it does not renew him yet

#### Scenario: Manager
- **WHEN** a Manager's key player has 2 seasons left
- **THEN** it waits until his last season, as before
