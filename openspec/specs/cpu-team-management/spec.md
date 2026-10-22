# cpu-team-management Specification

## Purpose

How an unowned (CPU) club runs itself: its AI level, when it acts, how it picks its lineup and
formation, answers transfer offers, and manages contracts, squad depth, academy, ticket price,
stadium and trainer. Shopping for players is covered by cpu-transfer-market.

## Requirements

### Requirement: AI levels
Each unowned club SHALL have an AI level, Caretaker (1), Manager (2) or Director (3), each a strict superset of the one below. New CPU teams draw it by tier: tier 1 10/30/60%, tier 2 30/50/20%, others 60/30/10% (Caretaker/Manager/Director). Admins can override it.

#### Scenario: Tier-1 team created
- **WHEN** a CPU team is created in a tier-1 league
- **THEN** it is a Director with 60% probability

#### Scenario: AI-managed gate
- **WHEN** any CPU routine checks a team
- **THEN** it only acts if the team has no owner and an AI level of at least Caretaker (national teams have level 0 and are never managed)

### Requirement: When CPU teams act
After each of its league matches or cup ties a CPU team SHALL refresh its lineup and answer pending offers. Its proactive management (roster upkeep, transfer pass, Director actions) MUST run only from the daily activity tick, inside its country's local 09:00–23:00 window.

#### Scenario: Activity roll
- **WHEN** the 5-minute activity tick runs and a CPU team's local time is inside its window
- **THEN** it runs its management pass with a probability that averages once per day

#### Scenario: Night time
- **WHEN** a CPU team's local time is 03:00
- **THEN** it starts no market activity and no friendly challenge

#### Scenario: Offer sweep
- **WHEN** the offer-response job runs at 00:00 or 12:00 UTC
- **THEN** every CPU team answers its pending offers, even if it has not played or acted recently

### Requirement: CPU friendlies
Inside its local window each CPU team SHALL roll for a friendly challenge (about 10 attempts per day, at most one challenge per day) against another AI-managed team that is also awake. The challenge MUST be auto-accepted and respect both teams' weekly friendly cap; humans are never challenged by the CPU.

#### Scenario: Challenge issued
- **WHEN** a CPU team's challenge roll succeeds
- **THEN** a friendly against a random awake CPU opponent is accepted at once and dated 5 minutes later

### Requirement: Lineup and formation
Every CPU team SHALL re-pick its best XI from the current squad whenever it acts, choosing among the six formations ranked by best-XI strength: a Director takes the strongest (rank 0), a Manager the second, a Caretaker the third or fourth at random.

#### Scenario: New signing
- **WHEN** a CPU team signs a better striker
- **THEN** its next lineup refresh can include him

### Requirement: Answering transfer offers
Every CPU team SHALL accept an offer, highest first, when the fee reaches its threshold: its perceived value (market value with level noise: 30% / 15% / 5%), raised for key starters (×2 Manager, ×2.5 Director) and internationals (Director +25% up to +50% at 20 caps). Sales that leave the squad or the position too thin MUST be refused.

#### Scenario: Thin position
- **WHEN** selling would leave the squad under its size minus 2 or the player's position under its minimum depth
- **THEN** the offer is rejected at every AI level

#### Scenario: Stable answer
- **WHEN** the same offer is evaluated on several sweeps
- **THEN** its noise is seeded from the offer id, so the answer does not change

#### Scenario: Negotiation window
- **WHEN** an acceptable offer is still inside its negotiation window
- **THEN** it is left pending

#### Scenario: Caretaker and potential
- **WHEN** a Caretaker values a teenager
- **THEN** it ignores his hidden potential

### Requirement: Manager roster upkeep
A Manager or Director SHALL, in order: renew any player with one season or less left at his expected wage and maximum acceptable term; withdraw listings that are no longer surplus and list at most one surplus player; promote an academy prospect into a thin position; set the revenue-maximising ticket price.

#### Scenario: Surplus listing
- **WHEN** a core position's unlisted depth exceeds its minimum by more than one
- **THEN** the weakest owned, non-loan-listed player there is listed at his asking price, unless signed in the last 14 days, an international, or a youngster on a team that signs projects

#### Scenario: Listing no longer surplus
- **WHEN** a listed player's position would fall below minimum depth without him
- **THEN** the listing is withdrawn

#### Scenario: Academy promotion
- **WHEN** a core position is below its minimum depth
- **THEN** the best academy prospect for it is promoted, subject to the same once-a-week cap a human has

### Requirement: Director stadium and trainer
A Director SHALL expand its stadium by 500 seats only when its budget is at least 3 × the cost and at least 40% of its latest 10 home league matches (5 or more on record) sold out. It MUST hire a trainer if it has none, or replace its trainer with one at least 10 skill points better.

#### Scenario: Rich but half-empty
- **WHEN** a Director could afford an expansion but its ground rarely sells out
- **THEN** it does not expand

### Requirement: Same rules as humans
Every CPU action SHALL go through the same production functions a human uses, with the same validation (budget, transfer window, contract state, insolvency).

#### Scenario: Window closed
- **WHEN** the transfer window is closed
- **THEN** a CPU team cannot list, buy or make offers either

## Code map

- `backend/engine/aiteams.go` (`isAiManagedTeam`, `RunAITeamMatchReaction`,
  `RunAITeamManagement`, `RespondToAllPendingOffers`, `manageRosterIfManager`,
  `listSurplusPlayer`, `promoteFromAcademyForWeakestPosition`, `maximizeTicketPrice`,
  `manageMarketIfDirector`, `refreshAiTeamLineup`, `RunAiTeamDailyActivity`).
- `common/aimanager.go` (`RandomAiLevel`, `AiTransferProfileFor`), `common/lineup.go`
  (`SelectBestFormationAndLineup`, `formationRankForAiLevel`), `common/timezones.go`
  (`IsWithinLocalActivityWindow`), `common/transferscoring.go` (`EvaluateIncomingOffer`,
  `MarketValue`, `IsKeyPlayer`), `common/squadassessment.go` (`MinPositionDepth`).
