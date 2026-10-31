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

#### Scenario: Released club becomes CPU
- **WHEN** a club released from a human owner finishes its reset in a tier-2 league
- **THEN** it draws a new AI level with the tier-2 weights (30/50/20%) and CPU routines start managing it

#### Scenario: Released club before its reset
- **WHEN** a released club is still waiting for its reset
- **THEN** it has no AI level and no CPU routine acts on it

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
Every CPU team SHALL re-pick its best XI from the current squad whenever it acts and again at the kickoff of each of its matches, choosing among the six formations ranked by best-XI strength: a Director takes the strongest (rank 0), a Manager the second, a Caretaker the third or fourth at random. A Manager or Director MUST pick only from players available for that match; a Caretaker picks from the whole squad.

#### Scenario: New signing
- **WHEN** a CPU team signs a better striker
- **THEN** its next lineup refresh can include him

#### Scenario: Injured starter, Manager
- **WHEN** a Manager's best striker is injured at kickoff of a league match
- **THEN** its formation and XI are chosen from the squad without him, so a different formation can win

#### Scenario: Injured starter, Caretaker
- **WHEN** a Caretaker's best striker is injured at kickoff
- **THEN** its XI still names him and the match-day backfill replaces him with the best available player for that position

#### Scenario: Suspended player in a friendly
- **WHEN** a Manager's player serving a suspension is picked for a friendly
- **THEN** he may play, because suspensions only bar league and cup matches

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

### Requirement: CPU kickoff lineup obeys a human's lineup rules
The XI, formation and scheduled substitutions a CPU team picks at kickoff MUST pass the same validation a human's saved lineup and substitutions pass, except that a Caretaker's XI may name unavailable players. A pick that fails validation MUST be dropped in favour of the team's stored lineup for that match.

#### Scenario: Valid pick
- **WHEN** a Director's kickoff XI and 3 substitutions are validated
- **THEN** they match the formation's position counts, name only squad players, never repeat a player, and every substitution takes off a starter for a fit bench player at a minute from 1 to 90

#### Scenario: Invalid pick
- **WHEN** a CPU kickoff pick fails validation
- **THEN** the match is played with the team's stored lineup and the failure is logged

### Requirement: CPU planned substitutions
At kickoff of a league match or cup tie a CPU team SHALL schedule substitutions by its AI level, scoring each starter/bench pair at a slot by fatigue gain: the bench player's average effective value over the remaining minutes minus the starter's, times those minutes (see scenarios). A Caretaker schedules at most 1, a Manager at most 2, a Director at most 3. No player is used in two pairs.

#### Scenario: Gain formula
- **WHEN** a starter worth V with Stamina Ss would be replaced at minute m by a bench player worth W with Stamina Sb
- **THEN** the starter's average factor is 1 − 0.30 × ((m + 90) / 180) × (99 − Ss) / 99, the bench player's is 1 − 0.30 × ((90 − m) / 180) × (99 − Sb) / 99
- **AND** gain = (90 − m) × (W × bench factor − V × starter factor), with W and V each player's value at the starter's slot

#### Scenario: Caretaker
- **WHEN** a Caretaker's lowest-Stamina outfield starter has Stamina below 50 and the best bench player for his slot is worth at least 80% of him
- **THEN** it schedules that one change at minute 70; otherwise it schedules none

#### Scenario: Manager
- **WHEN** a Manager evaluates every outfield pair at minute 70
- **THEN** it schedules the top 2 pairs by gain whose gain is at least 5% of (90 − 70) × V, so one substitution always stays free for injuries

#### Scenario: Director
- **WHEN** a Director evaluates every pair at minutes 55, 60, …, 85
- **THEN** each pair takes its best minute, and the top 3 pairs by gain whose gain is at least 2% of (90 − m) × V are scheduled
- **AND** the third of them is moved to minute 75 if it was due earlier, so an early injury can still be covered

#### Scenario: Nothing worth changing
- **WHEN** no pair reaches its level's threshold
- **THEN** the team schedules no substitutions

### Requirement: CPU friendly rotation
At kickoff of a friendly a CPU team SHALL start an XI and schedule 3 substitutions at minute 60 so its reserves play, by AI level. Regulars are the XI it would field in a league match that day. Bench players for the substitutions are picked by highest value at their best open slot, and each replaces the starter in that slot. Fewer than 3 are scheduled if the squad has no eligible reserves left.

#### Scenario: Caretaker friendly
- **WHEN** a Caretaker plays a friendly
- **THEN** it starts its regulars and at minute 60 brings on its 3 best bench players

#### Scenario: Manager friendly
- **WHEN** a Manager plays a friendly
- **THEN** its 5 outfield regulars with the highest value at their slot do not start or come on, it starts the best XI from the rest in its league formation, and its 3 substitutions use only players who did not start

#### Scenario: Director friendly
- **WHEN** a Director plays a friendly
- **THEN** it starts the best XI from players who are not regulars, including a backup goalkeeper if it has one
- **AND** a position no reserve can fill takes the lowest-valued regular for it, and key players start only if needed to field 11
- **AND** its 3 substitutions at minute 60 prefer players aged 21 or under, then the highest value

#### Scenario: Thin squad
- **WHEN** a CPU team has only 13 fit players for a friendly
- **THEN** it fields 11 and schedules at most 2 substitutions

## Code map

- `backend/engine/aiteams.go` (`isAiManagedTeam`, `RunAITeamMatchReaction`,
  `RunAITeamManagement`, `RespondToAllPendingOffers`, `manageRosterIfManager`,
  `listSurplusPlayer`, `promoteFromAcademyForWeakestPosition`, `maximizeTicketPrice`,
  `manageMarketIfDirector`, `refreshAiTeamLineup`, `RunAiTeamDailyActivity`).
- Kickoff lineups: `common/ailineup.go` (`PlanAiMatchLineup`, `AiMatchPlan`,
  `planCompetitiveSubstitutions`, `aiSubstitutionGain`) and `common/ailineupfriendly.go`
  (`planFriendlyRotation`), validated with `common.ValidateLineup` / `ValidateSubstitutions`
  (`common/lineupvalidation.go`); `engine/aikickoff.go` (`planAiKickoff`, `aiKickoffSeed`) runs in
  `simulateMatch`, `simulateCupTie` and `simulateFriendly` before `runMatchSimulation`. The fatigue
  numbers come from `common.FatigueMultiplierAt` (`common/fatigue.go`).
- Released clubs: `common/accounts.go` (`ReleaseTeam`, `ResetReleasedTeam` draws the new AI level
  with `RandomAiLevel(tier)`); `BackfillAiLevelsStmtSql` and `GetFreeTeamStmtSql` skip a club
  while `teams.reset_pending_since` is set, so a released club has no AI level and is not
  claimable until its reset.
- `common/aimanager.go` (`RandomAiLevel`, `AiTransferProfileFor`), `common/lineup.go`
  (`SelectBestFormationAndLineup`, `formationRankForAiLevel`), `common/timezones.go`
  (`IsWithinLocalActivityWindow`), `common/transferscoring.go` (`EvaluateIncomingOffer`,
  `MarketValue`, `IsKeyPlayer`), `common/squadassessment.go` (`MinPositionDepth`).
