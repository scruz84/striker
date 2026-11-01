# Spec Delta

## MODIFIED Requirements

### Requirement: Transfer pass
Every Manager and Director SHALL run one shopping pass per management tick, after roster upkeep, with no signing quota. A Manager signs at most 1 player per pass; a Director up to 3, of which at most 2 new offers. Accepting a loan and exercising a buy option each count as one signing.

#### Scenario: Nothing worth buying
- **WHEN** no candidate clears the team's minimum score
- **THEN** it signs nobody

#### Scenario: Loan beats a transfer
- **WHEN** a Manager's best-scoring candidate is a loan listing and the next one a listed player
- **THEN** it accepts the loan and signs nobody else that pass

### Requirement: Markets searched
A Manager SHALL consider the best 60 free agents, the transfer list and the loan market. A Director MUST additionally consider the best players at each core position anywhere (other humans' included), a random sample of 20 youngsters, and the national-team radar. Players older than 36 are ignored.

#### Scenario: Unlisted target
- **WHEN** a Director finds a strong player nobody listed
- **THEN** it may make an offer for him

#### Scenario: Human's loan listing
- **WHEN** a human lists a player for loan and he clears a Manager's minimum score
- **THEN** the Manager may accept the loan on the listed terms, and the human is notified as for any accepted loan

#### Scenario: Own loan listing
- **WHEN** a CPU team scans the loan market
- **THEN** its own loan listings are not candidates

### Requirement: Squad size cap and selling to buy
A CPU team MUST NOT grow beyond squad size + 4 players, counting unlisted owned players and players on loan to it. A Director that cannot pay for, or has no room for, a target SHALL list the player the target would replace (keeping minimum depth and size, with no other listing out) and buy later; it never lists to make room for a loan.

#### Scenario: Full squad
- **WHEN** a Director's owned squad is at the cap and it finds an upgrade
- **THEN** it lists the player he would replace instead of buying now

#### Scenario: Loaned-in players fill the cap
- **WHEN** a team has squad size + 2 unlisted owned players and 2 players on loan to it
- **THEN** it accepts no loan and signs nobody

### Requirement: Observability
Every CPU market action SHALL increment `striker.ai.market_actions` (by action) and log `ai-market team=… action=…`. Loan actions use the actions `loaned_in`, `exercised_buy_option`, `listed_for_loan` and `loan_listing_withdrawn`.

#### Scenario: Signing
- **WHEN** a CPU team signs a free agent
- **THEN** the counter is incremented and an `ai-market` log line records team, level, action, player and price

#### Scenario: Loan accepted
- **WHEN** a CPU team accepts a loan listing with a 30,000 fee
- **THEN** the counter is incremented with action `loaned_in` and the log line records price 30,000

## ADDED Requirements

### Requirement: Loan candidate score
A loan listing SHALL be scored like a signing, against the team's playing squad (unlisted, non-loan-listed owned players plus players already on loan to it), from best-XI points he adds now, the thin-position bonus, a loan value-for-money adjustment of up to ±4 and a +2 bonus for a fair buy option. A loan gets no growth projection and no reputation bonus. The minimum score is the level's usual one.

#### Scenario: Fair loan fee
- **WHEN** a team values a loan-listed player at 400,000 and the loan lasts 2 seasons
- **THEN** a fair fee is 400,000 × 0.10 × 2 = 80,000, and value for money is clamp((1 − fee ÷ 80,000) × 4, −4, +4)

#### Scenario: Free loan
- **WHEN** a loan listing has no fee
- **THEN** its value-for-money adjustment is +4

#### Scenario: Fair buy option
- **WHEN** a loan listing's buy-option fee is greater than 0 and at most the team's valuation of the player
- **THEN** the loan scores +2 more

#### Scenario: Already loaned in at his position
- **WHEN** a team already has a better striker on loan to it
- **THEN** a loan-listed striker adds only the best-XI points he gains over the squad including that loanee

### Requirement: Loan affordability
A loan MUST pass the level's usual affordability check, with the loan fee as the price and the borrower's share of the wage, wage × (100 − wage contribution %) ÷ 100 scaled by its division wage factor, as the payroll it adds.

#### Scenario: Lender pays half the wage
- **WHEN** a Director borrows a player on a 10,000 wage with a 50% wage contribution
- **THEN** its payroll check adds 5,000 (times its wage factor) per round

### Requirement: Loan limits
A CPU team MUST have at most 2 players on loan to it at once and MUST NOT accept a loan listing it cannot accept under the human rules (window, insolvency, budget for the fee).

#### Scenario: Two loanees already
- **WHEN** a Manager has 2 players on loan to it
- **THEN** loan listings are not candidates in its pass

### Requirement: Exercising buy options
In the last season of a loan (the loan returns at the next rollover), a Manager or Director SHALL score each player on loan to it whose loan carries a buy option as a permanent signing at the option fee, against its squad without him, and exercise the option when he clears its minimum score and the signing is affordable with his full wage.

#### Scenario: Starter worth keeping
- **WHEN** a loaned-in starter's buy option costs less than the team's valuation and his departure would cost the XI 6 points
- **THEN** the team exercises the option and he becomes its player

#### Scenario: Not the last season
- **WHEN** a 2-season loan has more than one season left
- **THEN** its buy option is not considered

#### Scenario: Too expensive
- **WHEN** exercising the option would leave a Manager with less than half its budget
- **THEN** the option is not exercised and the player returns at the end of the loan
