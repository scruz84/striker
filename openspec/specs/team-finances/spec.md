# team-finances Specification

## Purpose

How a team's money moves: the budget and its ledger, every income stream (gate, sponsorship,
prize money, season tickets, merchandising), every expense (wages scaled by division, stadium
maintenance and expansion, transfers, loans, severance), the insolvency lockout, and the
finance page's planning tools.

## Requirements

### Requirement: Budget and ledger
Every team SHALL start with a budget of 400,000, and every change to it MUST be a signed ledger entry with a `TRANSACTION_*` reason. The running budget and the transaction history are the whole financial model.

#### Scenario: Any payment
- **WHEN** a team receives or pays money for any reason
- **THEN** its budget changes by the amount and a transaction row records the amount and reason

### Requirement: Matchday gate
After a league match or cup tie the home side alone SHALL receive attendance × ticket price. Attendance MUST follow the socios model (see the socios spec) and never exceed stadium capacity. Ticket prices are integers from 1 to 100, default 10.

#### Scenario: Away side
- **WHEN** a league match is played
- **THEN** the away side receives no gate money

### Requirement: Friendly gate split
A friendly SHALL draw its crowd from the home side's stadium like a league match and split the gate evenly between both sides. Friendlies MUST NOT pay sponsorship, drift Fanbase or trigger the CPU's per-match upkeep.

#### Scenario: Friendly at a 1,000-seat ground
- **WHEN** a friendly earns 8,000 at the gate
- **THEN** home and away each receive 4,000 and no sponsorship is paid

### Requirement: Sponsorship
After every league match and cup tie both sides SHALL receive sponsorship: a tier base (8,000 / 4,000 / 1,500 for tiers 1 / 2 / 3, deeper tiers as tier 3) shaped by the team's chosen deal and the result, then × the Fanbase commercial factor. Owners can change deal at any time.

#### Scenario: Balanced deal (default)
- **WHEN** a Balanced team wins / draws / loses
- **THEN** it receives 1.6 / 1.3 / 1.0 × base

#### Scenario: Steady deal
- **WHEN** a Steady team plays
- **THEN** it receives 1.3 × base whatever the result

#### Scenario: Performance deal
- **WHEN** a Performance team wins / draws / loses
- **THEN** it receives 1.8 / 1.0 / 0.6 × base

#### Scenario: Deal previews
- **WHEN** the finance page shows the three deals
- **THEN** each preview uses the team's current tier and Fanbase, matching what will be paid

### Requirement: Prize money
At season rollover each league SHALL split its tier's pool (500,000 / 200,000 / 50,000) by final position, linearly from 2× the per-team average for 1st down to 0 for last. Each cup tie MUST pay its winner 5,000 × the round number, and the cup champion a further 100,000.

#### Scenario: Ten-team tier-1 league
- **WHEN** a ten-team tier-1 league ends
- **THEN** the champion receives 100,000 and the last team 0

#### Scenario: Cup round 3 win
- **WHEN** a team wins a round-3 cup tie
- **THEN** it receives 15,000

### Requirement: Season tickets and merchandising
At season rollover each team SHALL receive season tickets of capacity × 5 and a merchandising share of its tier's pool (100,000 / 40,000 / 10,000) from 1.5× the per-team average for 1st down to 0.5× for last, both × the Fanbase commercial factor.

#### Scenario: Last place still sells shirts
- **WHEN** a team finishes last
- **THEN** it still receives half the average merchandising share

### Requirement: Matchday wages
After each league match (never a cup tie or friendly) each side SHALL pay its squad's wage bill plus its trainer's wage, × the division wage factor (4.0 / 2.1 / 1.1 for tiers 1 / 2 / 3, deeper tiers as tier 3). The factor MUST apply only when wages are paid; stored wages stay the base figure.

#### Scenario: Promotion
- **WHEN** a team is promoted from tier 2 to tier 1
- **THEN** its next wage payments use ×4.0 with no change to any stored wage

#### Scenario: Offers and valuations
- **WHEN** a renewal, offer or CPU valuation reads a player's wage
- **THEN** it uses the stored base wage

### Requirement: Wage formulas
A player's base wage SHALL be the sum of his seven core skills × 5 ÷ 7; a trainer's base wage is his skill × 8 ÷ 7. Negotiated wages are checked against the player's expected wage.

#### Scenario: Average player
- **WHEN** a player's seven skills sum to 315
- **THEN** his base wage is 225 per round

### Requirement: Stadium maintenance
Each team SHALL pay capacity × 1 ÷ 10 in stadium maintenance once per round, at the same point as wages.

#### Scenario: Default stadium
- **WHEN** a team with 1,000 seats plays a league round
- **THEN** it pays 100 in maintenance

### Requirement: Stadium expansion
An owner SHALL be able to add between 100 and 5,000 seats at a cost of current capacity × 10 × seats ÷ 500, paid upfront; the seats go live 7 days later. Only one expansion MAY be under way at a time.

#### Scenario: 500 seats at 1,000 capacity
- **WHEN** a 1,000-seat team adds 500 seats
- **THEN** it pays 10,000 and its capacity becomes 1,500 seven days later

#### Scenario: Second expansion
- **WHEN** an expansion is already under construction
- **THEN** another request fails with `ErrStadiumUpgradeInProgress`

### Requirement: Transfers
A fixed-price listing purchase or an accepted offer SHALL debit the buyer and credit the seller the agreed fee in one atomic transaction. A free-agent signing costs only the wage, plus a premium of 4× the agreed wage to his last club if signed within his first season of free agency.

#### Scenario: Competing offers
- **WHEN** several teams have pending offers on one player (at most one per team)
- **THEN** the seller's Offers tab groups them under that player, sorted by amount

### Requirement: Loan money
A loan listing SHALL accept three independent optional terms, each off by default: a one-time loan fee paid upfront, a 0–100% wage contribution kept by the lender, and a buy-option fee. Loans last 1–3 seasons and are exclusive with a transfer listing.

#### Scenario: 40% wage contribution
- **WHEN** a loaned player's wage contribution is 40%
- **THEN** the borrower's wage bill includes 60% of his wage and the lender pays the other 40% each matchday as `TRANSACTION_LOAN_WAGE_CONTRIBUTION`

#### Scenario: Buy option
- **WHEN** the borrower exercises a loan's buy option
- **THEN** it pays the option fee to the lender and the move becomes permanent, like a transfer without moving the player

### Requirement: Contract renewal and release
An owner SHALL be able to renew a player for 1–3 seasons; a renewal meeting the player's expected wage and acceptable term is accepted. Releasing a player early MUST cost a severance of 10 × his wage.

#### Scenario: Veteran renewal
- **WHEN** a 33-year-old is offered a renewal
- **THEN** he accepts at most 1 season (2 seasons from 30 to 32, 3 under 30, longer only for a declining player)

### Requirement: Insolvency lockout
A team whose budget is below −200,000 SHALL be insolvent: buying a listed player, making an offer, accepting a loan, exercising a buy option and signing a free agent MUST fail with `ErrTeamInsolvent`. Selling, releasing and cancelling listings stay allowed.

#### Scenario: Budget at −250,000
- **WHEN** an insolvent team tries to make a transfer offer
- **THEN** the request fails with `ErrTeamInsolvent`

#### Scenario: Selling to recover
- **WHEN** an insolvent team accepts an offer for one of its players
- **THEN** the sale goes through

#### Scenario: Finance page warning
- **WHEN** an insolvent owner opens the finance page
- **THEN** the response includes `insolvent: true` and `maxDebt`, and a standing warning is shown

### Requirement: Financial planning tools
The finance page SHALL compute, client-side from the transaction list, a cash-flow forecast (weekly averages of wages + trainer wages + maintenance against ticket revenue + sponsorship, excluding season lumps and transfers), a budget-over-time chart and an income-breakdown chart.

#### Scenario: Negative weekly net
- **WHEN** weekly expenses exceed weekly income and the team is not yet insolvent
- **THEN** the forecast shows how many weeks until the budget reaches −maxDebt

## Code map

- Ledger: `common.AdjustTeamBudget`, reasons `TRANSACTION_*` (`common/fmconstants.go`).
- Post-match money: `engine/database.go` (`UpdateTeamFinancesAfterMatch`), `engine/stadium.go`
  (gate, `creditFriendlyTicketRevenue`, maintenance), `engine/sponsorship.go`,
  `engine/wages.go` (`PayMatchdayWages`, `deductWages`, `deductTrainerWage`).
- Season money: `engine/prizemoney.go`, `engine/seasonincome.go`.
- Wages: `common.PlayerWageFromSkills` (`common/fmcommon.go`), `common/trainers.go`,
  `common/wagefactor.go` (`DivisionWageFactor`), `common.GetTeamWageBill`,
  `common.GetTeamLoanWageContribution`.
- Stadium: `common/stadium.go` (`UpgradeStadium`, `StadiumUpgradeDuration`).
- Market money: `common/transfers.go`, `common/offers.go`, `common/loans.go`,
  `common/contracts.go`; insolvency `common.MaxDebt`/`IsTeamSolvent`.
- Frontend: `frontend/src/features/finances/` (`cashFlowForecast.ts`, `BudgetOverTimeChart`,
  `IncomeBreakdownChart`); offer grouping `ReceivedOffersByPlayer` in `TransferMarketPage.tsx`.
- Balance check: `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v`
  (`engine/economysim_test.go`); background in
  `openspec/changes/archive/2026-10-19-rebalance-economy-socios/`.
