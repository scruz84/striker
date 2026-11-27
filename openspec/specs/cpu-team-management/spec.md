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
Every CPU team SHALL re-pick its best XI from the current squad whenever it acts and again at the kickoff of each of its matches, ranking the six formations by the projected expected goal difference of each formation's best XI against the same squad's best 4-4-2 XI: a Manager or Director takes the first, a Caretaker the third or fourth at random; a Director then re-picks against its opponent at competitive kickoffs (see "Director formation against the opponent"). A Manager or Director MUST pick only from players available for that match; a Caretaker picks from the whole squad.

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

#### Scenario: Manager's shape
- **WHEN** a Manager's best 3-4-3 XI projects a higher expected goal difference against its own best 4-4-2 XI than any other formation's best XI
- **THEN** it plays 3-4-3

#### Scenario: Ranking ties
- **WHEN** two formations project the same expected goal difference within 0.000000001
- **THEN** the one whose XI has the higher summed slot value ranks first, and after that the earlier one in the order 4-4-2, 5-3-2, 3-4-3, 4-3-3, 4-5-1, 3-5-2

#### Scenario: Best XI by optimal assignment
- **WHEN** a squad's only flank players are a wide midfielder (Lateral, Midfield and Pass 75, the rest 25) and a full-back (Lateral and Defense 60, the rest 40), alongside a keeper, two centre-backs and weak reserves
- **THEN** the full-back plays full-back (slot value 57) and the wide midfielder wide midfield (60), because the XI is the assignment of players to slots with the highest total slot value
- **AND** the slot-by-slot pick, which reaches the back line first, would have put the wide midfielder at full-back (58) and a reserve in wide midfield

### Requirement: Director formation against the opponent
At the kickoff of a league match or cup tie a Director SHALL rate the best XI it can field in each of the six formations against its opponent's XI by expected goal difference, computed from the minute-1 zone ratings with the engine's chance-rate, chance-share and conversion formulas and no randomness, and play the highest; ties go to the stronger XI. Its substitutions, orders, captain and mentality are then planned for that XI.

#### Scenario: Exploiting a weak flank
- **WHEN** an opponent's XI has a far weaker right side than left and a Director's 4-3-3 expects a goal difference of +0.4 against it while its strongest-by-value 4-4-2 expects +0.2
- **THEN** the Director plays 4-3-3

#### Scenario: Friendly
- **WHEN** a Director plays a friendly
- **THEN** it keeps its friendly rotation and does not re-pick its formation

#### Scenario: Projection matches the engine
- **WHEN** two identical XIs are projected on a neutral ground
- **THEN** both expect the same goals, about 1.2 open-play goals each per match

### Requirement: Answering transfer offers
Every CPU team SHALL accept an offer, highest first, when the fee reaches its threshold: its perceived value (market value with level noise: 30% / 15% / 5%), raised for key starters (×2 Manager, ×2.5 Director) and internationals (Director +25% up to +50% at 20 caps). Sales that leave the squad or the position too thin, or of a player it signed within the last 14 days, MUST be refused. A Manager or Director counters an offer of at least 70% of its threshold (see transfer-negotiation) instead of rejecting it.

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

#### Scenario: Near-miss offer to a Director
- **WHEN** a Director's threshold for a player is 1,234,500 and it receives an offer of 1,000,000
- **THEN** it counters with 1,235,000 (the threshold rounded up to the next 1,000)

#### Scenario: Low offer
- **WHEN** a Manager receives an offer below 70% of its threshold
- **THEN** it rejects the offer

#### Scenario: Caretaker near miss
- **WHEN** a Caretaker receives an offer at 90% of its threshold
- **THEN** it rejects the offer

#### Scenario: Thin position near miss
- **WHEN** an offer at 95% of the threshold would leave the player's position too thin
- **THEN** it is rejected, never countered

#### Scenario: Recent signing
- **WHEN** a CPU team receives an offer of ten times his value for a player it signed 3 days ago
- **THEN** it rejects the offer at every AI level and does not counter it, the same 14-day hold that keeps him off its transfer list

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

### Requirement: Manager roster upkeep
A Manager or Director SHALL, in order: withdraw listings that are no longer surplus and list at most one surplus player; withdraw loan listings that are no longer surplus and offer at most one player on loan; promote an academy prospect into a thin position; set the revenue-maximising ticket price.

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
Every CPU action SHALL go through the same production functions a human uses, with the same validation (budget, transfer window, contract state, insolvency). This includes listing for loan, accepting a loan and exercising a buy option.

#### Scenario: Window closed
- **WHEN** the transfer window is closed
- **THEN** a CPU team cannot list, buy, make offers, list for loan or accept a loan either

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

### Requirement: CPU mentality
At the kickoff of a league match or cup tie a CPU team SHALL pick its mentality from the ratio R of its kickoff XI's strength to the opponent's kickoff XI's strength (sum of each player's value at his slot). A Caretaker is always Balanced. A Manager is Attacking when R ≥ 1.15, Defensive when R ≤ 0.87, Balanced otherwise. A Director first multiplies R by 1.07 at home and by 1/1.07 away, then uses 1.10 and 0.90. In friendlies every CPU team is Balanced.

#### Scenario: Manager favourite
- **WHEN** a Manager's XI is worth 1,150 against an opponent's 1,000
- **THEN** it plays Attacking

#### Scenario: Director away underdog
- **WHEN** a Director's XI is worth 950 against 1,000 away from home
- **THEN** R becomes 0.95 / 1.07 ≈ 0.888 and it plays Defensive

#### Scenario: Caretaker
- **WHEN** a Caretaker plays any match
- **THEN** it plays Balanced

### Requirement: CPU conditional orders
At the kickoff of a league match or cup tie a CPU team SHALL add conditional orders by AI level. A Caretaker adds none. A Manager adds "from 70, if losing, go Attacking". A Director adds "from 65, if losing, go Attacking" and "from 75, if winning, go Defensive". In friendlies CPU teams add no orders.

#### Scenario: Director protecting a lead
- **WHEN** a Director leads 1–0 at the start of minute 75
- **THEN** it switches to Defensive in minute 75

#### Scenario: Manager chasing the game
- **WHEN** a Manager is losing 0–1 at the start of minute 80 and was drawing until then
- **THEN** it switches to Attacking in minute 80

### Requirement: CPU captain
At kickoff of every match a CPU team SHALL name as captain its oldest outfield starter who is not the outgoing player of one of its scheduled substitutions, ties going to the lowest player id, and leave both set-piece takers automatic. If every outfield starter is due off, the oldest outfield starter is captain.

#### Scenario: Veteran captain
- **WHEN** a CPU team's outfield starters are aged 24, 31 and 33 among others, with 33 the oldest, and none of them is scheduled off
- **THEN** the 33-year-old is its captain

#### Scenario: Veteran due off
- **WHEN** the 33-year-old is the outgoing player of one of the team's scheduled substitutions
- **THEN** the 31-year-old is its captain

#### Scenario: Takers
- **WHEN** a CPU team wins a penalty
- **THEN** its taker is chosen by the automatic rule

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

### Requirement: Answering pre-contract offers
On its offer sweep a Manager or Director with a pending pre-contract offer on a player it would keep (see "Contract renewals") SHALL counter by renewing him at his raised expectation when that is at most 1.5 × his skill-based wage for a starter (2 × for a Director's key player) or 1.25 × for anyone else; otherwise it lists him under "Selling contracts it will not renew" when that rule allows it. A Caretaker MUST NOT react beyond its usual renewals.

#### Scenario: Manager keeps its starter
- **WHEN** a Manager's starter with a skill-based wage of 2,000 receives a pre-contract offer that raises his expectation to 2,800
- **THEN** it renews him at 2,800 on its next sweep and the offer is cancelled

#### Scenario: Rival outbids the ceiling
- **WHEN** the same starter's expectation rises to 3,200 (1.6 × his skill-based wage)
- **THEN** the Manager does not renew him and, if the rule allows it, lists him for transfer

#### Scenario: Director key player
- **WHEN** a Director's key player with a skill-based wage of 2,000 has an expectation of 3,800
- **THEN** it renews him at 3,800

#### Scenario: Director lets a bench veteran go
- **WHEN** a Director's 32-year-old bench player it does not renew receives an offer
- **THEN** it lists him for transfer instead of renewing him

#### Scenario: Caretaker bench player
- **WHEN** a Caretaker's bench player in his last season receives an offer
- **THEN** it does nothing, and after 72 hours he signs the pre-contract

### Requirement: Release clauses in CPU renewals
When it renews a player, a Manager SHALL set a release clause of 2 × its valuation for a non-starter and none for a starter; a Director 4 × its valuation unless he is a key player or an international; a Caretaker none. A clause below the minimum is raised to it, and the wage offered is the discounted expected wage.

#### Scenario: Manager bench renewal
- **WHEN** a Manager renews a bench player it values at 300,000, his market value on a 3-season contract, with an expected wage of 1,500
- **THEN** it sets a 600,000 clause and offers 1,350

#### Scenario: Director key player
- **WHEN** a Director renews its key striker
- **THEN** it sets no clause and pays his full expected wage

### Requirement: AI level after a division change
At the season rollover a CPU club (unowned, AI level 1 or more) that is promoted or relegated SHALL draw a level with its new tier's weights (see "AI levels") and move one step towards it: up one level if the draw is higher, down one if lower, unchanged if equal. A club with a human owner or no AI level is untouched.

#### Scenario: Caretaker reaches the top flight
- **WHEN** a Caretaker is promoted to tier 1 and the draw gives Director
- **THEN** it becomes a Manager

#### Scenario: Director relegated
- **WHEN** a Director is relegated to tier 3 and the draw gives Caretaker
- **THEN** it becomes a Manager

#### Scenario: Same draw
- **WHEN** a promoted Manager draws Manager
- **THEN** it stays a Manager

### Requirement: Club style
Every CPU club SHALL have a style, assigned once from its team id and stored for its whole life (a club with none reads as balanced): of every 5 consecutive ids, 2 are balanced, 1 youth, 1 frugal and 1 ambitious. A Manager's or Director's transfer profile is adjusted by it; a Caretaker ignores it.

#### Scenario: Youth club
- **WHEN** a Manager's style is youth
- **THEN** it signs players for their projected growth and lends out benched under-22s, as a Director does

#### Scenario: Frugal club
- **WHEN** a Director's style is frugal
- **THEN** its budget reserve is 30% instead of 15% and its key-player price multiplier is 2.0 instead of 2.5

#### Scenario: Ambitious club
- **WHEN** a Manager's style is ambitious
- **THEN** its budget reserve is 40%, its minimum score 4 and its key-player multiplier 2.5

#### Scenario: Stable
- **WHEN** the same club's style is read twice, before and after a restart
- **THEN** it is the same

### Requirement: Prospects by potential
On its management pass a Director, or a Manager whose style is youth, that has not promoted an academy prospect into a thin position SHALL promote the prospect with the highest projected value at his best core position, read through its potential noise 3 seasons on, when that value is at least its weakest starter's value at that position, within the weekly promotion cap.

#### Scenario: Gifted prospect
- **WHEN** a Director's 17-year-old academy striker projects to 64 and its weakest starting striker is worth 60
- **THEN** it promotes him to the senior squad

#### Scenario: Ordinary prospect
- **WHEN** its best prospect projects to 50 against a weakest starter of 60
- **THEN** nobody is promoted

#### Scenario: Balanced Manager
- **WHEN** a balanced Manager has the same gifted prospect
- **THEN** it only promotes for a thin position

### Requirement: CPU season objective
A CPU club SHALL derive a season objective from its league table once it has played a third of its league matches (`open` before): `relegation_fight` within 3 points of the relegation zone, else `promotion_push` within 3 points of the last promotion place (the title in a top division), else `nothing_at_stake` when it can neither reach that place nor be caught by the zone at 3 points per remaining match, else `open`.

#### Scenario: Early season
- **WHEN** a club has played 9 of its 30 league matches
- **THEN** its objective is `open` whatever its position

#### Scenario: Just above the drop
- **WHEN** with 2 relegation places a club has 20 points and the best side in the zone has 18, after 12 of 30 matches
- **THEN** its objective is `relegation_fight`

#### Scenario: Chasing promotion
- **WHEN** with 2 promotion places a club in 4th has 40 points and 2nd place has 42
- **THEN** its objective is `promotion_push`

#### Scenario: Dead rubber
- **WHEN** with 3 league matches left a club has 45 points, 2nd place has 60 and the best side in the relegation zone 30
- **THEN** its objective is `nothing_at_stake`, since 45 + 9 < 60 and 30 + 9 < 45

#### Scenario: Points compared
- **WHEN** the objective is computed
- **THEN** "within 3 points of the zone" means points ≤ those of the best-placed side in the relegation zone + 3 (only with a tier below), and "within 3 points of the place" means points ≥ that place's points − 3

#### Scenario: Top division
- **WHEN** a tier-1 club with no tier above is 2 points behind the leader after 20 of 30 matches
- **THEN** its objective is `promotion_push` (the title race)

#### Scenario: Playoff places count
- **WHEN** a club plays in a middle tier of a pyramid that follows the league-pyramid ratio rule
- **THEN** its promotion places are 1st and 2nd (direct and playoff) and its relegation zone is the bottom 4 (playoff and direct), so 2 promotion and 4 relegation places

### Requirement: Youth minutes when nothing is at stake
At the kickoff of a league match a Director whose objective is `nothing_at_stake` SHALL start up to 2 fit players aged 21 or under from its bench: for each of its outfield starters from the lowest-valued up, the best such player at his slot replaces him when worth at least 80% of him. Its substitutions, orders and captain are then planned for that XI.

#### Scenario: Two prospects get a start
- **WHEN** a Director with nothing at stake has bench players aged 19 and 20 worth 85% and 90% of its two weakest outfield starters at their slots
- **THEN** both start in their place

#### Scenario: Prospect too raw
- **WHEN** its only bench player under 22 is worth 60% of the weakest starter at his slot
- **THEN** the XI is unchanged

#### Scenario: Still something to play for
- **WHEN** the same Director's objective is `relegation_fight`
- **THEN** it plays its usual XI

### Requirement: CPU sponsor deal
On each management pass a Manager or Director that has played at least 6 league matches this season and has not yet chosen a deal this season SHALL compute each deal's expected payout per match from its season win rate w and draw rate d, and switch to the best one when it is not its current deal. Ties keep the current deal. A Caretaker never changes its deal.

#### Scenario: Payout formulas
- **WHEN** the expected payouts are computed, in units of the tier's base sponsorship
- **THEN** Steady is 1.3, Balanced is 1 + 0.6 × w + 0.3 × d and Performance is 0.6 + 1.2 × w + 0.4 × d

#### Scenario: Strugglers take the guarantee
- **WHEN** a Manager has 1 win and 2 draws in 8 league matches (w = 0.125, d = 0.25)
- **THEN** Steady (1.3) beats Balanced (1.15) and Performance (0.85), and it switches to Steady, locked for the season

#### Scenario: Runaway leader
- **WHEN** a Director has 7 wins and 1 draw in 9 league matches (w ≈ 0.778, d ≈ 0.111)
- **THEN** Performance (≈ 1.578) beats Balanced (≈ 1.50) and it switches to Performance

#### Scenario: Too early
- **WHEN** a Director has played 5 league matches this season
- **THEN** it keeps its current deal

### Requirement: Manager trainer
A Manager with no trainer SHALL hire, on its management pass, the most skilled available trainer whose skill is below 80; it MUST NOT fire a trainer it has. A Director keeps its rule (hire the best, or swap for one at least 10 better). A Caretaker never hires.

#### Scenario: Elite coaches left alone
- **WHEN** a Manager with no trainer finds trainers of skill 88, 79 and 45 on the market
- **THEN** it hires the 79

#### Scenario: Only elite coaches
- **WHEN** every available trainer has skill 80 or more
- **THEN** the Manager hires nobody

#### Scenario: Better coach appears
- **WHEN** a Manager already has a trainer of skill 40 and one of skill 75 becomes available
- **THEN** it keeps its trainer

### Requirement: Director cup orders
At kickoff of a cup tie a Director SHALL add, after its other conditional orders, "from 75, if drawing" with mentality Attacking when its kickoff XI's strength is at least the opponent's, and Defensive otherwise. A draw goes to a penalty shootout won by either side with probability 50%, so a weaker side settles for it.

#### Scenario: Favourite level late
- **WHEN** a Director whose XI is worth 1,050 against 1,000 is drawing at the start of minute 75 of a cup tie
- **THEN** it switches to Attacking in minute 75

#### Scenario: Underdog level late
- **WHEN** a Director whose XI is worth 950 against 1,000 is drawing at the start of minute 80 of a cup tie
- **THEN** it switches to Defensive

#### Scenario: League match
- **WHEN** a Director plays a league match
- **THEN** it adds no drawing order

### Requirement: AI level calibration
An opt-in simulation (`AI_LEVEL_SIM=1`) SHALL play 4,000 matches of each level against a Caretaker on a neutral ground, both sides with the same squad, each side planning its XI, formation, substitutions, mentality and orders as at a league kickoff, and report each side's win, draw and loss rates and points per match. A Manager MUST score more points per match than the Caretaker it plays, and a Director more than a Manager does.

#### Scenario: Running it
- **WHEN** an operator runs `AI_LEVEL_SIM=1 go test ./engine -run TestAiLevelCalibration -v`
- **THEN** the log lists, for Manager vs Caretaker and Director vs Caretaker, both sides' win/draw/loss rates and points per match

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

## Code map

- `backend/engine/aiteams.go` (`isAiManagedTeam`, `RunAITeamMatchReaction`,
  `RunAITeamManagement`, `RespondToAllPendingOffers`, `manageRosterIfManager`,
  `listSurplusPlayer`, `promoteFromAcademyForWeakestPosition`, `maximizeTicketPrice`,
  `manageMarketIfDirector`, `refreshAiTeamLineup`, `RunAiTeamDailyActivity`,
  `recentlySignedOrInternational`, `protectedFromListing`); near-miss counters in
  `respondToReceivedOffers` via `cpuCounterAmount` (`common.CpuCounterMinShare` /
  `CpuCounterRounding` in `common/offercounters.go`, `common.CounterOffer`). Tests:
  `engine/aiteams_test.go` (`TestCpuCounterAmount`), `engine/counteroffers_integration_test.go`.
- Lending: `backend/engine/ailoans.go` (`manageLoanListings`, `pickPlayerToLoanOut`,
  `loanOutEligible`, `loanOutRoom`, `planLoanListingWithdrawals`, the `ai*Loan*` constants);
  limits `common.AiMaxOpenLoanListings` / `AiMaxLoanedOut` (`common/offerrejections.go`); readers
  `common.GetTeamMarketCommitments` (`LoanedOut`), `common.GetTeamTransferListings`
  (`common/transfers.go`); profile flags `LendsPlayers` / `LoansOutYouth`
  (`common/aimanager.go`). Tests: `engine/ailoans_test.go`,
  `engine/cpu_loan_market_integration_test.go`.
- Kickoff mentality: `engine/aikickoff.go` (`planAiKickoffs` sets the planned side's
  `Lineup.Mentality` with `kickoffStrength` and `aiKickoffMentality`), `common.PickAiMentality` and
  `common.LineupStrength` with the `AiManager*Ratio` / `AiDirector*Ratio` / `AiDirectorVenueFactor`
  constants (`common/mentality.go`). Tests: `common/ailineup_test.go` (`TestPickAiMentality`),
  `engine/aimentality_integration_test.go`.
- Kickoff orders: `common.AiConditionalOrders` (`aiManagerChaseMinute`, `aiDirectorChaseMinute`,
  `aiDirectorProtectMinute` in `common/ailineup.go`), set as `AiMatchPlan.Orders` by
  `PlanAiMatchLineup` for competitive matches only and checked with `common.ValidateOrders` in
  `planAiKickoff`. Tests: `common/ailineup_test.go` (`TestAiConditionalOrders`),
  `engine/conditional_orders_test.go` (`TestCpuConditionalOrders_*`).
- Kickoff captain: `common.AiCaptain` (`common/ailineup.go`), set as `AiMatchPlan.Roles` by
  `PlanAiMatchLineup` in every match and checked with `common.ValidateRoles` in `planAiKickoff`.
  Tests: `common/ailineup_test.go` (`TestPlanAiMatchLineup_VeteranCaptain`, `TestAiCaptain_*`).
  `AiCaptain` skips the plan's outgoing players (`oldestOutfieldStarter`); tests
  `TestAiCaptain_SkipsScheduledOutgoingPlayer`, `TestPlanAiMatchLineup_CaptainStaysOnThePitch`.
- Director cup orders: `common.AiCupDrawOrder` (`aiCupDrawOrderMinute`), appended for cup ties by
  `setAiKickoffTactics` / `applyAiKickoffTactics` in `engine/aikickoff.go`. Tests:
  `common/ailineup_test.go` (`TestAiCupDrawOrder`), `engine/aikickoff_tactics_test.go`.
- Sponsor deal and Manager trainer: `engine/aibusiness.go` (`chooseSponsorDealIfManager`,
  `bestSponsorDeal`, `expectedSponsorPayout` over `sponsorshipAmount`, `aiSponsorMinMatches`;
  `hireTrainerIfManager`, `pickManagerTrainer`, `aiManagerTrainerMaxSkill`), called from
  `manageRosterIfManager`; season record from `common.GetTeamSeasonRecord`
  (`common/sponsordeal.go`). Tests: `engine/aibusiness_test.go`,
  `engine/ai_business_integration_test.go`.
- Financial discipline: `engine/aifinances.go` (`inFinancialCrisis`, `teamInCrisis`,
  `pickCrisisSale`, `listForFundsIfInCrisis`, `pickVeteranToRelease`,
  `releaseUnsoldVeteranIfManager`, the `aiCrisis*` / `aiVeteran*` / `aiMaxRenewalRaise`
  constants); `RunAITeamManagement` reads the budget once and skips `runAiTransferPass` in a
  crisis; selective renewals in `contractsToRenew` / `shouldMeetRenewalTerms`
  (`engine/aiteams.go`); offer discount `common.OfferContext.Distressed` / `CrisisOfferShare`
  (`common/transferscoring.go`); loan listing ages `common.GetTeamLoanListingTimes`
  (`common/loans.go`). Tests: `engine/aifinances_test.go`, `engine/aiteams_test.go`
  (`TestContractsToRenew_*`, `TestShouldMeetRenewalTerms`), `common/transferscoring_test.go`
  (`TestEvaluateIncomingOffer_CrisisDiscount`), `engine/ai_finances_integration_test.go`.
- Selling contracts it will not renew: `listUnrenewedPlayer` / `pickUnrenewedToList`
  (`engine/aiteams.go`, action `listed_expiring`), called from `manageRosterIfManager` outside a
  crisis. Director early renewals: `contractsToRenewEarly` (`earlyRenewalYears`) in
  `renewExpiringContracts`. Renewals offer the player's own terms at once (`renewContract` with
  `common.PlayerTermsExpectation`), capped for non-starters by `shouldMeetRenewalTerms`. Tests:
  `engine/aicontractterms_test.go`, `engine/contract_terms_integration_test.go`.
- Answering pre-contract offers: `answerPreContractOffers` (from `RespondToAllPendingOffers`),
  `acceptsCounterBar` (`aiCounterStarterCap`, `aiCounterKeyPlayerCap`) with
  `common.PlayerRenewalExpectation` in `renewContract`, `listUnrenewedPlayerAmong`. Release clauses
  in renewals: `aiRenewalClause`, `renewalClauseAmount` (`aiManagerClauseMultiple`,
  `aiDirectorClauseMultiple`) in `engine/aiteams.go`.
- Season objective: `common/seasonobjective.go` (`SeasonSituation`, `SeasonObjectiveFor`,
  `SeasonObjectiveStartShare`, `SeasonObjectiveMargin`, `GetTeamSeasonSituation` with
  `CountRemainingLeagueMatchesStmtSql`, `TeamSeasonObjective`). Youth starts:
  `common.AiPlanOptions`, `PlanAiMatchLineupWithOptions`, `giveYouthStarts` (`aiYouthMaxAge`,
  `aiYouthMinValueRatio`) in `common/ailineup.go`, set by `aiPlanOptions`
  (`aiDirectorYouthStarts`) in `engine/aikickoff.go`. Tests: `common/seasonobjective_test.go`,
  `common/seasonobjective_integration_test.go`, `engine/ai_season_context_integration_test.go`.
- Director formation against the opponent: `engine/expectedgoals.go` (`ProjectExpectedGoals`,
  `expectedConversion`, `shooterMix`, over `ProjectZoneDuel`), `engine/aiformation.go`
  (`chooseFormationAgainst`), the second pass in `planAiKickoffs` (`directorFormationAgainst`,
  `planAiKickoffWithFormation`, `common.AiPlanOptions.Formation`) in `engine/aikickoff.go`.
  Manager rank 0 in `common.formationRankForAiLevel`. Tests: `engine/expectedgoals_test.go`,
  `engine/aiformation_test.go`, `common/lineup_test.go`.
- Formation ranking and best XI: `common.SelectBestFormationAndLineup` ranks by the registered
  `common.FormationScorer` (`SetFormationScorer`; the engine registers `projectedFormationScore`
  from `init()` in `engine/aiformation.go`) against the squad's own best 4-4-2 XI;
  `common.SelectBestLineupForFormation` picks each XI by optimal slot assignment
  (`bestAssignmentLineup`, `optimalAssignment` in `common/assignment.go`). Tests:
  `common/formationscorer_test.go`, `common/assignment_test.go`, `engine/aiformation_test.go`.
- Club identity: `common/aistyle.go` (`AiClubStyleFor`, `AiStyleCode` / `AiStyleFromCode`,
  `AssignAiStyle`, `BackfillAiStyles`, `ApplyClubStyle`, `AiTransferProfileForTeam`), column
  `teams.ai_style` (`resources/sql/history/init102_team_ai_style.sql`) read into `Team.AiStyle` by
  `GetTeamById`, assigned in `handlers.createTeam` and at startup (`service.StartSvc`); the engine
  reads every CPU profile through `aiProfile` (`engine/aiteams.go`). Level after a division change:
  `common.RandomAiLevelFrom` / `StepAiLevelToward` (`common/aimanager.go`) and
  `redrawAiLevelAfterMove` (`engine/aibusiness.go`), called per move in `engine/season.go`.
  Prospects by potential: `pickProspectByPotential` / `promoteProspectByPotential`
  (`engine/aibusiness.go`), after `promoteFromAcademyForWeakestPosition` in
  `manageRosterIfManager`. Tests: `common/aistyle_test.go`, `engine/aibusiness_test.go`,
  `engine/ai_club_identity_integration_test.go`.
- Calibration: `engine/ai_level_calibration_test.go` (`TestAiLevelCalibration`,
  `calibrationSquad`, `measureAiLevel`; `AI_LEVEL_SIM=1`).
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
  (`IsWithinLocalActivityWindow`), `common/transferscoring.go` (`EvaluateIncomingOffer`, whose `OfferContext.RecentlySigned` comes from
  `GetPlayerIdsAcquiredSince` in `engine/aiteams.go` `respondToReceivedOffers`,
  `MarketValue`, `IsKeyPlayer`), `common/squadassessment.go` (`MinPositionDepth`).
