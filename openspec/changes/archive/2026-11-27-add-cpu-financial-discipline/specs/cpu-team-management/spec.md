# Spec Delta

## MODIFIED Requirements

### Requirement: Contract renewals
Every time it acts, a CPU team SHALL renew the contract of each player with one season or less left whom it keeps: a Caretaker, or a club in financial crisis, keeps only its current default XI; otherwise a Manager or Director keeps its XI, every player under 30 and anyone whose position would fall under minimum depth without him. The offer MUST be the skill-based wage at the age's maximum term; if declined, it is repeated once on the player's own terms, for a starter or when his wage is at most 1.25 × the skill-based one. A loanee is never renewed by the borrower.

#### Scenario: Improving player asks for a raise
- **WHEN** a Manager's 21-year-old with one season left has just improved and declines his skill-based wage, asking for 1.2 × that wage
- **THEN** the team renews him at the higher wage he asked for instead of letting him leave as a free agent

#### Scenario: Caretaker bench player
- **WHEN** a Caretaker's player with one season left is not in its current default XI
- **THEN** his contract is not renewed and he leaves as a free agent at season end

#### Scenario: Caretaker starter
- **WHEN** a Caretaker's player with one season left is in its current default XI
- **THEN** his contract is renewed

#### Scenario: Bench veteran
- **WHEN** a Director's 31-year-old bench midfielder with one season left plays a position that keeps its minimum depth without him
- **THEN** his contract is not renewed

#### Scenario: Veteran keeping the depth
- **WHEN** a Manager's 33-year-old bench goalkeeper is one of its only 2 goalkeepers
- **THEN** he is renewed, since goalkeepers need a depth of 2

#### Scenario: Greedy bench player
- **WHEN** a Manager's 24-year-old bench player declines his skill-based wage and asks for 1.4 × it
- **THEN** no second offer is made and he leaves at season end

## ADDED Requirements

### Requirement: CPU financial crisis
A CPU club SHALL be in financial crisis while its budget is below 0. A Manager or Director in crisis MUST NOT sign, borrow or exercise a buy option (its transfer pass is skipped) and, once per management pass while it has fewer than 2 open transfer listings, SHALL list its highest-wage owned player who is not a key player, an international, signed in the last 14 days, transfer- or loan-listed, and whose sale keeps minimum depth and squad size − 2, at 80% of his usual asking price.

#### Scenario: In debt
- **WHEN** a Director's budget is −50,000 and the transfer window is open
- **THEN** it makes no signing or offer that pass and lists its best-paid sellable player at 80% of his asking price

#### Scenario: Two listings already
- **WHEN** a Manager in crisis already has 2 players on the transfer list
- **THEN** it lists nobody else that pass

#### Scenario: Back in the black
- **WHEN** the same club's budget is 10,000
- **THEN** its transfer pass runs as usual

#### Scenario: Caretaker in debt
- **WHEN** a Caretaker's budget is below 0
- **THEN** it lists nobody (it only answers offers)

### Requirement: Crisis offer discount
A CPU club of any level in financial crisis SHALL accept an offer that reaches 80% of the threshold it would otherwise ask (see "Answering transfer offers"); refusals for a thin squad or position are unchanged and a Manager's or Director's counter-offer is computed from the discounted threshold.

#### Scenario: Cheaper sale in debt
- **WHEN** a club in crisis values a player at a threshold of 500,000 and receives 410,000
- **THEN** it accepts, since 410,000 ≥ 400,000

#### Scenario: Not in debt
- **WHEN** the same club with a positive budget receives 410,000
- **THEN** a Manager or Director counters and a Caretaker rejects

### Requirement: Releasing unsold veterans
A Manager or Director not in financial crisis, owning more players than the squad size (18), SHALL release through the human release rules at most one player per management pass: the one with the highest wage among owned players aged 30 or more, with 2 or more seasons left, transfer- or loan-listed for at least 14 days, and only when its budget is at least 3 × his severance (10 × wage). His listing is withdrawn first.

#### Scenario: Nobody wanted him
- **WHEN** a Manager owning 20 players has had a 32-year-old with 2 seasons left on the loan list for 15 days, with a wage of 3,000 and a budget of 200,000
- **THEN** it withdraws the listing and releases him, paying a severance of 30,000

#### Scenario: Squad at its size
- **WHEN** the same Manager owns 18 players
- **THEN** it releases nobody

#### Scenario: Listed for 10 days
- **WHEN** the veteran has been listed for 10 days
- **THEN** he stays
