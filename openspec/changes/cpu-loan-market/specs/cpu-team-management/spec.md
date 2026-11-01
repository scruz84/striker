# Spec Delta

## MODIFIED Requirements

### Requirement: Manager roster upkeep
A Manager or Director SHALL, in order: renew any player with one season or less left at his expected wage and maximum acceptable term; withdraw listings that are no longer surplus and list at most one surplus player; withdraw loan listings that are no longer surplus and offer at most one player on loan; promote an academy prospect into a thin position; set the revenue-maximising ticket price.

#### Scenario: Surplus listing
- **WHEN** a core position's unlisted depth exceeds its minimum by more than one
- **THEN** the weakest owned, non-loan-listed player there is listed at his asking price, unless signed in the last 14 days, an international, or a youngster on a team that signs projects

#### Scenario: Listing no longer surplus
- **WHEN** a listed player's position would fall below minimum depth without him
- **THEN** the listing is withdrawn

#### Scenario: Academy promotion
- **WHEN** a core position is below its minimum depth
- **THEN** the best academy prospect for it is promoted, subject to the same once-a-week cap a human has

### Requirement: Same rules as humans
Every CPU action SHALL go through the same production functions a human uses, with the same validation (budget, transfer window, contract state, insolvency). This includes listing for loan, accepting a loan and exercising a buy option.

#### Scenario: Window closed
- **WHEN** the transfer window is closed
- **THEN** a CPU team cannot list, buy, make offers, list for loan or accept a loan either

## ADDED Requirements

### Requirement: Loan-out limits
A Manager or Director SHALL offer at most one new loan listing per management tick and MUST NOT have more than 2 open loan listings, nor more than 3 players out on loan counting its open listings as players about to go. A Caretaker never lists for loan.

#### Scenario: Three players out on loan
- **WHEN** a Director has 3 players out on loan
- **THEN** it offers nobody else on loan until one returns

### Requirement: Loan-out eligibility
A CPU team MUST only offer on loan an owned player who is not transfer-listed, not an international, not signed in the last 14 days, and whose contract has at least the loan's duration + 1 seasons left. His position MUST keep its minimum depth and the squad at least squad size − 2 owned, unlisted players without him.

#### Scenario: Contract too short
- **WHEN** a surplus player has 1 season left on his contract
- **THEN** he is not offered on a 1-season loan

#### Scenario: Thin position
- **WHEN** loaning a player out would leave his position under its minimum depth
- **THEN** he is not offered on loan

### Requirement: Development loans
A Director SHALL offer on loan a player aged 21 or younger who is outside its best XI, picking the youngest such player first (then the lowest player id). Terms: 1 season, no loan fee, the Director keeps paying 50% of his wage, no buy option.

#### Scenario: Benched teenager
- **WHEN** an eligible 19-year-old is outside a Director's best XI
- **THEN** it lists him for a 1-season loan with fee 0, wage contribution 50% and no buy option

#### Scenario: Manager keeps its youngsters
- **WHEN** a Manager has a benched 19-year-old
- **THEN** it does not offer him on a development loan

### Requirement: Surplus loans
A Manager or Director SHALL turn a transfer listing unsold for 7 days into a loan listing: withdraw the transfer listing and offer the player on a 1-season loan with a loan fee of 10% of his market value, no wage contribution, and a buy option at the asking price he was listed at.

#### Scenario: Unsold surplus
- **WHEN** a surplus player worth 200,000 has been transfer-listed for 7 days at 200,000 without a buyer
- **THEN** he is withdrawn from the transfer list and offered on a 1-season loan with fee 20,000, wage contribution 0% and a 200,000 buy option

#### Scenario: Listed for 6 days
- **WHEN** a surplus player has been transfer-listed for 6 days
- **THEN** he stays on the transfer list

### Requirement: Withdrawing loan listings
A CPU team SHALL withdraw any of its loan listings whose player stopped being eligible (his position would fall under its minimum depth, the squad under squad size − 2, or a development loanee entered the best XI).

#### Scenario: Injury crisis
- **WHEN** a loan-listed defender's position drops under its minimum depth after a sale
- **THEN** the loan listing is withdrawn
