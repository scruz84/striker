# cpu-transfer-market Specification

## Purpose

How CPU teams shop for players - a Caretaker only to fill a hole, a Manager and a Director for upgrades: which markets they look at, how a
candidate is scored, how they bid and the limits that keep them from flooding humans with offers,
plus how national-team status acts as a scouting radar.

## Requirements

### Requirement: Transfer pass
Every CPU team SHALL run one shopping pass per management tick, after roster upkeep, with no signing quota. A Caretaker or a Manager signs at most 1 player per pass; a Director up to 3, of which at most 2 new offers. Accepting a loan and exercising a buy option each count as one signing.

#### Scenario: Nothing worth buying
- **WHEN** no candidate clears the team's minimum score
- **THEN** it signs nobody

#### Scenario: Caretaker fills a hole
- **WHEN** a Caretaker's squad is under 18 players, or a position is under its minimum depth
- **THEN** it may sign one free agent or listed player that fills it (any position when the squad is short), with a score of at least 0 and at most 1.5 × its valuation, keeping half its budget in reserve

#### Scenario: Caretaker with no hole
- **WHEN** a Caretaker's squad is full and every position at its minimum depth
- **THEN** it signs nobody, however good the candidate

#### Scenario: Loan beats a transfer
- **WHEN** a Manager's best-scoring candidate is a loan listing and the next one a listed player
- **THEN** it accepts the loan and signs nobody else that pass

### Requirement: Markets searched
A Caretaker SHALL consider the best 60 free agents and the transfer list; a Manager also the loan market. A Director MUST additionally consider the best players at each core position anywhere (other humans' included), a random sample of 20 youngsters, and the national-team radar. Players older than 36 are ignored.

#### Scenario: Unlisted target
- **WHEN** a Director finds a strong player nobody listed
- **THEN** it may make an offer for him

#### Scenario: Human's loan listing
- **WHEN** a human lists a player for loan and he clears a Manager's minimum score
- **THEN** the Manager may accept the loan on the listed terms, and the human is notified as for any accepted loan

#### Scenario: Own loan listing
- **WHEN** a CPU team scans the loan market
- **THEN** its own loan listings are not candidates

### Requirement: Candidate score
A candidate SHALL score the best-XI points he adds as perceived through the level's noise, +4 for filling a thin position, a value-for-money adjustment of up to ±4, and for a Director a 3-season growth projection for under-24s and a reputation bonus for internationals.

#### Scenario: Minimum score
- **WHEN** a candidate is scored
- **THEN** a Manager needs at least 5, a Director at least 3 and a Caretaker (filling a hole) at least 0 to sign him

#### Scenario: Hidden potential
- **WHEN** a Director projects a youngster
- **THEN** it reads his ceiling with ±15% noise (a Manager ±60%), never the exact value

#### Scenario: Projects
- **WHEN** a Director signs a player for his future
- **THEN** it keeps at most 3 such projects and spends at most a quarter of its budget on one

### Requirement: Affordability
A signing MUST leave the budget above the level's reserve: 50% for a Caretaker or a Manager, 15% for a Director, who also needs 20 rounds of the new payroll (scaled by its division wage factor).

#### Scenario: Reserve breached
- **WHEN** a Manager's signing would leave less than half its budget
- **THEN** the candidate is unaffordable

### Requirement: Bidding
For an unlisted player a Director SHALL bid valuation × 1.1 × (1 + 0.02 per score point above its minimum score, bump capped at 0.25), never above 1.5 × its valuation nor its spendable budget. Listed players are bought at the asking price, and a loanee's buy option at its fee, but never when that price is above 1.5 × the team's valuation - the same ceiling as its bids; free agents cost the usual premium to their last club.

#### Scenario: Very wanted target
- **WHEN** a Director's target scores 20
- **THEN** the bump is capped at 0.25 and the bid is valuation × 1.375

#### Scenario: Overpriced listing
- **WHEN** a player the team values at 10,000 is listed at 30,000
- **THEN** it does not buy him, however much he would improve the XI

### Requirement: Offer limits
CPU offers MUST respect: at most 3 open on one human's player, from different teams (each notified to the human); at most 1 open on another CPU team's player; at most 4 open per CPU team; offers unanswered for 3 days are withdrawn.

#### Scenario: Fourth CPU bid on a human's player
- **WHEN** three CPU teams already have offers open on a human's player
- **THEN** no other CPU team bids for him

### Requirement: Rejection cooldown
After a rejection or withdrawal a CPU team SHALL leave that player alone for 3 days, then only return with a bid at least 10% higher, and only if its valuation allows it.

#### Scenario: Rebid
- **WHEN** a rejected 100,000 bid's cooldown expires
- **THEN** the next bid is at least 110,000 or none is made

### Requirement: Squad size cap and selling to buy
A CPU team MUST NOT grow beyond squad size + 4 players, counting unlisted owned players and players on loan to it. A Director that cannot pay for, or has no room for, a target SHALL list the player the target would replace (keeping minimum depth and size, with no other listing out) and buy later; it never lists to make room for a loan.

#### Scenario: Full squad
- **WHEN** a Director's owned squad is at the cap and it finds an upgrade
- **THEN** it lists the player he would replace instead of buying now

#### Scenario: Loaned-in players fill the cap
- **WHEN** a team has squad size + 2 unlisted owned players and 2 players on loan to it
- **THEN** it accepts no loan and signs nobody

### Requirement: National-team radar
A player in his country's national-team pool SHALL count as an international; the radar of every active country's pool is cached for 15 minutes. A Director scouts each country's top 5 who beat its weakest starter at their position (best 20), scores internationals +2 +0.1 per cap (up to 20 caps), and asks more for them.

#### Scenario: International never surplus
- **WHEN** a CPU team picks a surplus player or a player to sell for funds
- **THEN** internationals are never chosen

### Requirement: Observability
Every CPU market action SHALL increment `striker.ai.market_actions` (by action) and log `ai-market team=… action=…`. Loan actions use the actions `loaned_in`, `exercised_buy_option`, `listed_for_loan` and `loan_listing_withdrawn`.

#### Scenario: Signing
- **WHEN** a CPU team signs a free agent
- **THEN** the counter is incremented and an `ai-market` log line records team, level, action, player and price

#### Scenario: Loan accepted
- **WHEN** a CPU team accepts a loan listing with a 30,000 fee
- **THEN** the counter is incremented with action `loaned_in` and the log line records price 30,000

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

### Requirement: Answering counter-offers
On its next offer-response sweep a CPU buyer SHALL accept a counter-offer when the counter fee is at most 1.5 × its valuation of the player and still passes its affordability check, and decline it otherwise. A declined counter MUST start the usual rejection cooldown for that player, with the original bid as the base for the next one.

#### Scenario: Acceptable counter
- **WHEN** a Director valued a player at 1,000,000, bid 1,100,000 and is countered at 1,300,000 with enough budget
- **THEN** it accepts and the transfer completes at 1,300,000

#### Scenario: Counter too high
- **WHEN** the same Director is countered at 1,600,000
- **THEN** it declines, and it cannot bid for him again for 3 days, then only at 1,210,000 or more

### Requirement: Contract length in valuations
Every CPU valuation of a player (asking prices, offer thresholds, bids and surplus loan fees) SHALL multiply his market value by a contract factor: 0.6 with 1 season or less left, 0.85 with 2 seasons and 1.0 with 3.

#### Scenario: Last season
- **WHEN** a player's market value is 500,000 and he has 1 season left
- **THEN** a CPU club lists him at 300,000 and, as a Director, bids from 300,000 × 1.1

#### Scenario: Fresh contract
- **WHEN** the same player has 3 seasons left
- **THEN** his value stays 500,000

### Requirement: CPU personal terms
A CPU offer or renewal SHALL propose exactly the player's expected wage (with the wage floor) and his age's maximum term, so a CPU club never makes an offer the player refuses on personal terms.

#### Scenario: Overpaid target
- **WHEN** a Director bids for a player earning 3,000 whose market wage is 2,500
- **THEN** its offer carries a wage of 3,000

### Requirement: Pre-contract targets
A Manager or Director SHALL also scan the best players in their last season at each core position (a Director 10 per position, a Manager 5) and score each as a free signing at his offered wage. It MAY make one pre-contract offer per pass, at exactly 1.10 × his expected wage, to a candidate clearing its minimum score, with at most 2 signed pre-contracts, which count toward its squad size cap.

#### Scenario: Director plans next season
- **WHEN** a Director finds a striker in his last season whose expected wage is 3,000 and who scores above its minimum
- **THEN** it offers him a pre-contract at 3,300 and his club is notified

#### Scenario: Two signed already
- **WHEN** a Manager already has 2 signed pre-contracts
- **THEN** it makes no pre-contract offer

#### Scenario: Pile-up on a human's player
- **WHEN** three CPU clubs have pending pre-contract offers on a human's player
- **THEN** no other CPU club offers him one

### Requirement: Paying release clauses
A Director SHALL treat a scanned player whose release clause is at most 1.5 × its valuation as buyable at the clause, scored like a listed player, and pay at most one clause per pass, counted as a signing. No CPU club MAY pay a clause on a human club's player when a CPU club paid one on that human club in the last 7 days.

#### Scenario: Cheap clause
- **WHEN** a Director values a player at 1,000,000, his clause is 1,400,000 and he clears its minimum score
- **THEN** it pays the clause and signs him

#### Scenario: Clause above its ceiling
- **WHEN** the clause is 1,600,000
- **THEN** it does not pay it

#### Scenario: Rate limit
- **WHEN** a CPU club paid a clause on a human club 3 days ago
- **THEN** no CPU club pays another clause on that human club's players

## Code map

- The pass is skipped for a club in financial crisis by `RunAITeamManagement`
  (`engine/aiteams.go`, `inFinancialCrisis` in `engine/aifinances.go`).
- Market urgency: `common.AdjustProfileForObjective` (`common/seasonobjective.go`), applied at the
  top of `runAiTransferPass` with `common.TeamSeasonObjective`. Test:
  `common/seasonobjective_test.go` (`TestAdjustProfileForObjective`).
- `backend/engine/aitransfers.go` (`runAiTransferPass`, candidate sizes, bid constants; loans:
  `candidateLoan` / `candidateBuyOption`, `loanListingCandidates`, `buyOptionCandidates`,
  `transferState.playingSquad`, the loan branches of `scoreOne`, `pickNextTarget` and
  `executeTransferMove`).
- `common/transferscoring.go` (`ScoreTarget` and its loan context `TargetContext.Loan`,
  `loanFairFeeShare`, `loanBuyOptionBonus`, `MaxAiSquadSize`, `MaxProjects`,
  `ListingAskingPrice`), loan readers in `common/loans.go` (`GetActiveLoanListings`,
  `GetLoansToTeam`), limit `common.AiMaxLoanedIn` (`common/offerrejections.go`), `common/aimanager.go` (`AiTransferProfile`), `common/aimarket.go`,
  `common/offerrejections.go` (offer limits, cooldown, `player_offer_rejections`),
  `common/nationalradar.go` (`GetNationalRadar`), `common/telemetry.go`.
- Contract length in valuations: `contractValueFactor` in `common.MarketValue`
  (`LastSeasonContractValueFactor`, `TwoSeasonContractValueFactor`, `common/transferscoring.go`;
  test `TestMarketValue_ContractFactor`). CPU personal terms: `common.PlayerTermsExpectation` in
  `makeAiOffer` (`engine/aitransfers.go`) and `renewContract` (`engine/aiteams.go`).
- Pre-contract targets: `preContractCandidates`, `offerBestPreContract` (its own phase after
  `runAiSignings`, so it uses none of the pass's moves), `makeAiPreContractOffer`,
  `aiPreContractOffersPerPass`, `candidatePreContract` and `transferState.SignedPreContracts` in
  `engine/aitransfers.go`; search `common.GetExpiringCandidatesForPosition` (`common/aimarket.go`),
  `AiTransferProfile.PreContractCandidates`. Release clauses: `candidateReleaseClause`,
  `payableClauses`, `payAiReleaseClause`, `aiReleaseClausesPerPass`, `common.CpuMayPayClauseOn`.
  Tests: `engine/precontracts_integration_test.go`, `engine/releaseclauses_integration_test.go`,
  `engine/aicontractterms_test.go`, `TestAiTransferMarket_PreContractSeason`
  (`engine/ai_transfers_simulation_integration_test.go`).
- Counter-offers to a CPU buyer: `engine/aiteams.go` (`answerCountersAsBuyer`, `counterScore`,
  `acceptsCounter`), called from `RespondToAllPendingOffers`; it answers through
  `common.AcceptCounterOffer` / `DeclineCounterOffer` and records the declined bid with
  `common.RecordOfferRejection`. Tests: `engine/aiteams_test.go` (`TestAcceptsCounter`),
  `engine/counteroffers_integration_test.go`, `engine/ai_transfers_simulation_integration_test.go`.
