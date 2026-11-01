# cpu-transfer-market Specification

## Purpose

How Manager- and Director-level CPU teams shop for players: which markets they look at, how a
candidate is scored, how they bid and the limits that keep them from flooding humans with offers,
plus how national-team status acts as a scouting radar.

## Requirements

### Requirement: Transfer pass
Every Manager and Director SHALL run one shopping pass per management tick, after roster upkeep, with no signing quota. A Manager signs at most 1 player per pass; a Director up to 3, of which at most 2 new offers.

#### Scenario: Nothing worth buying
- **WHEN** no candidate clears the team's minimum score
- **THEN** it signs nobody

### Requirement: Markets searched
A Manager SHALL consider the best 60 free agents and the transfer list. A Director MUST additionally consider the best players at each core position anywhere (other humans' included), a random sample of 20 youngsters, and the national-team radar. Players older than 36 are ignored.

#### Scenario: Unlisted target
- **WHEN** a Director finds a strong player nobody listed
- **THEN** it may make an offer for him

### Requirement: Candidate score
A candidate SHALL score the best-XI points he adds as perceived through the level's noise, +4 for filling a thin position, a value-for-money adjustment of up to ±4, and for a Director a 3-season growth projection for under-24s and a reputation bonus for internationals.

#### Scenario: Minimum score
- **WHEN** a candidate is scored
- **THEN** a Manager needs at least 5 and a Director at least 3 to sign him

#### Scenario: Hidden potential
- **WHEN** a Director projects a youngster
- **THEN** it reads his ceiling with ±15% noise (a Manager ±60%), never the exact value

#### Scenario: Projects
- **WHEN** a Director signs a player for his future
- **THEN** it keeps at most 3 such projects and spends at most a quarter of its budget on one

### Requirement: Affordability
A signing MUST leave the budget above the level's reserve: 50% for a Manager, 15% for a Director, who also needs 20 rounds of the new payroll (scaled by its division wage factor).

#### Scenario: Reserve breached
- **WHEN** a Manager's signing would leave less than half its budget
- **THEN** the candidate is unaffordable

### Requirement: Bidding
For an unlisted player a Director SHALL bid valuation × 1.1 × (1 + 0.02 per score point above its minimum score, bump capped at 0.25), never above 1.5 × its valuation nor its spendable budget. Listed players are bought at the asking price; free agents cost the usual premium to their last club.

#### Scenario: Very wanted target
- **WHEN** a Director's target scores 20
- **THEN** the bump is capped at 0.25 and the bid is valuation × 1.375

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
A CPU team MUST NOT grow beyond squad size + 4 unlisted owned players. A Director that cannot pay for, or has no room for, a target SHALL list the player the target would replace (keeping minimum depth and size, with no other listing out) and buy later.

#### Scenario: Full squad
- **WHEN** a Director's owned squad is at the cap and it finds an upgrade
- **THEN** it lists the player he would replace instead of buying now

### Requirement: National-team radar
A player in his country's national-team pool SHALL count as an international; the radar of every active country's pool is cached for 15 minutes. A Director scouts each country's top 5 who beat its weakest starter at their position (best 20), scores internationals +2 +0.1 per cap (up to 20 caps), and asks more for them.

#### Scenario: International never surplus
- **WHEN** a CPU team picks a surplus player or a player to sell for funds
- **THEN** internationals are never chosen

### Requirement: Observability
Every CPU market action SHALL increment `striker.ai.market_actions` (by action) and log `ai-market team=… action=…`.

#### Scenario: Signing
- **WHEN** a CPU team signs a free agent
- **THEN** the counter is incremented and an `ai-market` log line records team, level, action, player and price

## Code map

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
