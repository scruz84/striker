# match-simulation Specification

## Purpose

How a single match (league, cup tie or friendly) is turned into a result: which players take the
field, how team strength is rated each minute, and how chances, goals, cards, injuries, fatigue and
forfeits are decided. All three competitions share the same engine.

## Requirements

### Requirement: Per-minute simulation pipeline
The simulator SHALL run, for every minute from 1 to 45, then each first-half added minute (45+1 … 45+A1), then 46 to 90, then each second-half added minute (90+1 … 90+A2), the stages in this order: substitutions (scheduled substitutions, then conditional orders), stamina (fatigue), team qualification, scorer, discipline, injury. A1 and A2 are fixed as described in stoppage-time.

#### Scenario: A minute is simulated
- **WHEN** minute N is simulated
- **THEN** due scheduled substitutions are applied first, then due conditional orders
- **AND** fatigue-adjusted player copies are rebuilt before qualification and scoring read them
- **AND** cards and injuries are rolled after the scoring stage

#### Scenario: Added minutes play like any other
- **WHEN** a match has 2 and 4 added minutes
- **THEN** 96 minutes are simulated in the order 1…45, 45+1, 45+2, 46…90, 90+1…90+4
- **AND** chances, set pieces, cards, injuries, injury replacements and the forfeit check apply in every added minute

#### Scenario: Scheduled substitutions in added time
- **WHEN** a first half is in its added minutes
- **THEN** no scheduled substitution fires, because scheduled substitutions only have minutes 1 to 90; one scheduled for minute 46 fires at 46

### Requirement: Unavailable players never take the field
Before kickoff the simulator MUST remove from the saved lineup every outfield player serving an injury or suspension, then refill each thinned position back to its original headcount with the best remaining eligible squad player for that position. Positions are refilled in the formation's fixed slot order (goalkeeper, defence, midfield, attack). If the squad runs out, the position stays short.

#### Scenario: Suspended striker in the saved lineup
- **WHEN** a suspended striker is still in a team's saved lineup
- **THEN** he does not play and the best available squad player for that position takes his slot

#### Scenario: Squad crisis
- **WHEN** no eligible player is left for a position
- **THEN** that position is left short rather than fielding anyone twice or anyone unavailable

#### Scenario: Suspensions in friendlies
- **WHEN** a player serving a red-card suspension is in a friendly lineup
- **THEN** he may play, because suspensions only bar league and cup matches (injuries still apply)

#### Scenario: Two positions need the same reserve
- **WHEN** an injured defender and an injured midfielder are both best replaced by the same bench player
- **THEN** he fills the defence slot, because defence comes first in slot order, and the midfield slot takes the next best player

### Requirement: In-match fatigue
Each minute, every player's seven core skills SHALL be scaled by `1 - P × (min(minutes played, 90) / 90) × ((99 - Stamina) / 99)`, where P is 0.30 for a Balanced side, 0.33 for an Attacking side and 0.27 for a Defensive side at that minute, and every later stage MUST read these effective skills. Added minutes count as minutes played.

#### Scenario: Stamina 99
- **WHEN** a Stamina-99 player has played 90 minutes
- **THEN** his effective skills equal his raw skills

#### Scenario: Stamina 0 at full time
- **WHEN** a Stamina-0 player of a Balanced side has played 90 minutes
- **THEN** each of his core skills is reduced by 30%

#### Scenario: Not yet on the pitch
- **WHEN** a player has played 0 minutes this match
- **THEN** he is unaffected whatever his Stamina

#### Scenario: Attacking side at full time
- **WHEN** a Stamina-0 player of an Attacking side has played 90 minutes
- **THEN** each of his core skills is reduced by 33%

#### Scenario: Deep into added time
- **WHEN** a Stamina-0 player of a Balanced side has played 95 minutes, added minutes included
- **THEN** each of his core skills is reduced by 30%, the same as at 90

### Requirement: Team qualification recomputed every minute
The simulator SHALL recompute each team's attack, midfield and defense ratings (center, left and right) every minute from whoever is on the pitch, so red cards, injuries and substitutions take effect immediately.

#### Scenario: Core ratings
- **WHEN** a central rating is computed
- **THEN** it is the sum of the relevant skill over the players in that line (e.g. every striker's Shot)
- **AND** wide forwards also count toward the central attack

#### Scenario: Flank ratings
- **WHEN** a flank rating is computed
- **THEN** it blends the flank player's own skill 50/50 with his Lateral rating
- **AND** in a back three (one DEFENSE slot) the flank defenders weigh Lateral 35% and Defense 65%

#### Scenario: Zone spillover
- **WHEN** a line's zones are blended
- **THEN** each zone takes 15% of its neighbors' per-player average strength
- **AND** the center's blended value is scaled back up by its player count

### Requirement: Chance generation
Each minute SHALL produce an attacking chance with probability 14% plus up to 10 points scaled by midfield dominance, plus 1 point for each Attacking side and minus 1 point for each Defensive side. The attacking side MUST be chosen by a midfield-weighted share bounded to 32%–68%.

#### Scenario: Even midfields
- **WHEN** both midfields are equal and both sides are Balanced
- **THEN** a chance occurs with 14% probability and each side gets it half of the time

#### Scenario: Total midfield dominance
- **WHEN** one side holds almost all the midfield strength and both sides are Balanced
- **THEN** the chance probability approaches 24% and that side gets at most 68% of chances

#### Scenario: Two Attacking sides
- **WHEN** both midfields are equal and both sides are Attacking
- **THEN** a chance occurs with 16% probability

#### Scenario: Attacking against Defensive
- **WHEN** one side is Attacking and the other Defensive
- **THEN** the mentality adjustments cancel and the probability is the same as for two Balanced sides

### Requirement: Chance resolution
An open-play chance SHALL be central 50%, left 25% or right 25%, taken by a striker 70%, midfielder 20% or defender 10% (falling back to any remaining line), and converted with a logistic probability bounded to 5%–65% and centered on 28% for an even matchup. Set-piece chances are resolved separately (see "Set-piece chances").

#### Scenario: Flank chance
- **WHEN** a chance goes down the left
- **THEN** the attack value is the left attack plus 0.2 × (own left midfield − opponent's right midfield)
- **AND** it is compared with the opponent's right defense, because the two teams attack in opposite directions

#### Scenario: Finisher and goalkeeper
- **WHEN** a chance is resolved
- **THEN** the attack value includes the shooter's own scoring value and the defense value includes the goalkeeper's skill

#### Scenario: Clear-cut versus half-chance
- **WHEN** a chance is resolved
- **THEN** it is clear-cut (attack ×1.6) with probability 20% ± up to 15 points by attack-vs-defense edge, bounded to 5%–50%, and otherwise a half-chance (attack ×0.7)

#### Scenario: No outfield player left
- **WHEN** the attacking team has no outfield players left
- **THEN** the chance is discarded

#### Scenario: Set piece instead of open play
- **WHEN** a chance has been classified as a penalty or a direct free kick
- **THEN** none of the open-play rules above (flank, shooter weights, clear-cut roll, logistic curve) apply to it

### Requirement: Set-piece chances
Each attacking chance SHALL first be classified: a penalty with probability 2.2%, a direct free kick with probability 6.5%, and otherwise an open-play chance. The set piece goes to the side that won the chance, and it MUST NOT add to or remove from the number of chances a match produces.

#### Scenario: Chance classification
- **WHEN** a chance is generated
- **THEN** it is a penalty if a uniform roll r < 0.022, a direct free kick if 0.022 ≤ r < 0.087, and open play otherwise

#### Scenario: Expected volume in an even match
- **WHEN** two teams with equal midfields play (about 12.6 chances per match at 14% per minute)
- **THEN** the match averages about 0.28 penalties and 0.82 direct free-kick attempts
- **AND** these match the real-world 0.24–0.41 penalties and ~0.85 direct free-kick shots per match

#### Scenario: Dominant side wins more penalties
- **WHEN** one side gets 68% of the chances
- **THEN** it also gets on average 68% of the penalties and direct free kicks, because they are a fixed share of its chances

#### Scenario: Total goals unchanged
- **WHEN** a large sample of even matches is simulated before and after this change
- **THEN** the average goals per match differ by no more than 0.05
- **AND** that holds because each set piece replaces an open-play chance worth ~0.24 goals with a penalty worth 0.78 or a free kick worth 0.06

### Requirement: Penalty resolution
A penalty SHALL be taken by the attacking side's named penalty taker when he is on the pitch; otherwise by its on-pitch outfield player with the highest fatigue-adjusted Shot, ties going to the lowest player id. It is scored with probability `0.78 + 0.30 × (taker Shot − keeper GoalKeeper) / 99`, bounded to 60%–92%. It uses neither the open-play clear-cut roll nor the flank logic.

#### Scenario: Even taker and keeper
- **WHEN** the taker's Shot equals the opposing keeper's GoalKeeper
- **THEN** the penalty is scored with probability 78%

#### Scenario: Elite taker against a weak keeper
- **WHEN** the taker has Shot 80 and the keeper has GoalKeeper 30
- **THEN** the probability is 0.78 + 0.30 × 50/99 ≈ 0.93, bounded to 92%

#### Scenario: Weak taker against an elite keeper
- **WHEN** the taker has Shot 25 and the keeper has GoalKeeper 85
- **THEN** the probability is 0.78 − 0.30 × 60/99 ≈ 0.598, bounded to 60%

#### Scenario: Best finisher already sent off
- **WHEN** the side has no named taker and its highest-Shot player has been sent off, injured off or substituted
- **THEN** the best remaining on-pitch outfield player takes the penalty

#### Scenario: No outfield player left
- **WHEN** the attacking side has no outfield player on the pitch
- **THEN** the penalty is discarded, the same as an open-play chance

#### Scenario: Named taker on the pitch
- **WHEN** the named penalty taker has Shot 60 and a team-mate on the pitch has Shot 75
- **THEN** the named taker takes the penalty and it is scored with the probability for Shot 60

#### Scenario: Named taker on the bench
- **WHEN** the named penalty taker has not come on yet
- **THEN** the automatic rule picks the taker

### Requirement: Direct free-kick resolution
A direct free kick SHALL be taken by the attacking side's named free-kick taker when he is on the pitch; otherwise by its on-pitch outfield player with the highest `0.6 × Shot + 0.4 × Pass` (fatigue-adjusted), ties going to the lowest player id. It is scored with probability `0.06 + 0.10 × (taker value − keeper GoalKeeper) / 99`, bounded to 2%–15%.

#### Scenario: Even taker and keeper
- **WHEN** the taker's value equals the keeper's GoalKeeper
- **THEN** the free kick is scored with probability 6%, matching the real-world ~5–6% conversion

#### Scenario: Specialist taker
- **WHEN** the taker has Shot 75 and Pass 70 (value 73) and the keeper has GoalKeeper 40
- **THEN** the probability is 0.06 + 0.10 × 33/99 ≈ 0.093

#### Scenario: Bounds
- **WHEN** the taker value exceeds the keeper's GoalKeeper by 99, or trails it by 99
- **THEN** the probability is 15% and 2% respectively

#### Scenario: Named free-kick taker substituted off
- **WHEN** the named free-kick taker was substituted off in minute 60 and a free kick comes in minute 70
- **THEN** the automatic rule picks the taker

### Requirement: Set-piece event recording
A set-piece outcome SHALL be recorded as the usual goal or missed-chance event, carrying a set-piece kind (0 open play, 1 penalty, 2 direct free kick) and naming the defending goalkeeper as the other team's player. Every goal count (score, top scorers, awards, player stats, match ratings) MUST include set-piece goals.

#### Scenario: Penalty scored
- **WHEN** a home penalty is converted
- **THEN** a goal event with set-piece kind 1 is recorded for the taker, the home score goes up by one, and the away goalkeeper is listed as the other team's player

#### Scenario: Free kick missed
- **WHEN** a direct free kick is not converted
- **THEN** a missed-chance event with set-piece kind 2 is recorded and the score is unchanged

#### Scenario: Historic events
- **WHEN** an event stored before this change is read
- **THEN** its set-piece kind is 0 (open play)

#### Scenario: Match report shows the kind
- **WHEN** a match report or live replay lists a set-piece goal
- **THEN** it is marked as a penalty or a free kick next to the scorer's name in the viewer's language

### Requirement: Discipline
Each minute SHALL have a chance of a card for a random outfield player of either side: 3%, or 3.6% in a Rival match and 4.2% in a Clásico (league and cup only, see rivalries). 6% of first cards are straight reds, and a second yellow becomes a red. A sent-off player MUST leave the pitch unreplaced. Goalkeepers are never booked.

#### Scenario: Straight red in a league match
- **WHEN** a player gets a straight red in a league or cup match
- **THEN** he is removed for the rest of the match and banned for 2 matches

#### Scenario: Second yellow
- **WHEN** a player already booked is booked again in a league or cup match
- **THEN** he is sent off and banned for 1 match

#### Scenario: Red card in a friendly
- **WHEN** a player is sent off in a friendly
- **THEN** he leaves the match but carries no suspension

#### Scenario: Clásico
- **WHEN** a league match between Clásico rivals is simulated
- **THEN** each minute has a 4.2% chance of a card, about 3.8 cards per match instead of 2.7

### Requirement: Injuries
Each minute SHALL have a 0.5% chance of injuring a random outfield player, removing him for the rest of the match and sidelining him for 1–5 matches. If the team has not used its 3 substitutions, the best available bench player MUST replace him as a logged substitution. A player already substituted off this match is not available. Goalkeepers are never injured.

#### Scenario: Substitutions left
- **WHEN** a player is injured and his team has used 2 of 3 substitutions
- **THEN** the best available bench player comes on in his place

#### Scenario: No substitutions left
- **WHEN** a player is injured and all 3 substitutions are used
- **THEN** the team plays on a man down

#### Scenario: Best option already came off
- **WHEN** a player is injured in minute 80 and the best player for his position was substituted off in minute 60
- **THEN** the next best eligible bench player comes on instead

### Requirement: Forfeit when a side runs out of players
Before kickoff and after every minute the simulator MUST check whether exactly one side has 7 or fewer players on the pitch; if so the match ends there, recorded as a 3–0 win for the other side with a `MATCH_ABANDONED` event.

#### Scenario: Fourth red card
- **WHEN** a side drops to 7 players
- **THEN** no further minutes are simulated and the result is 3–0 to the opponent, whatever the score was

#### Scenario: Both sides collapse together
- **WHEN** both sides are at 7 or fewer at the same check
- **THEN** no forfeit is awarded and the match plays on

### Requirement: Post-match processing
After a league match the engine SHALL record the result and standings, update finances, charge both sides' matchday wages, and store the match's attendance and weather in the report. The weather and the attendance MUST be drawn once, before the simulation, and used for the home advantage, the gate revenue and the report.

#### Scenario: League match finishes simulating
- **WHEN** a league match has been simulated
- **THEN** ticket revenue, sponsorship and wages are booked immediately
- **AND** availability updates, skill growth and the result notification wait for the live replay (see match-scheduling)

#### Scenario: Result save fails
- **WHEN** writing the result, events, instants or standings fails
- **THEN** none of them is stored, no budget changes, no replay starts, and the match stays unplayed for the next attempt

#### Scenario: Leftovers from an earlier failed attempt
- **WHEN** a match still has events or instants stored by an earlier failed attempt
- **THEN** the successful save replaces them, so the match keeps exactly one set

#### Scenario: Crowd drawn before kickoff
- **WHEN** a league match is simulated and 9,500 attendance is drawn before it
- **THEN** the home advantage uses 9,500, the gate revenue is for 9,500 spectators and the report shows 9,500

#### Scenario: Retried after a failed save
- **WHEN** a match's first simulation attempt fails to save
- **THEN** no gate is booked for it, and the next attempt draws its weather and attendance again

### Requirement: Stored match record for every competition
Every simulated match SHALL store the same record, whatever its competition (league match, cup tie or friendly): its events, its per-minute zone instants (one per simulated minute, added minutes included), its added time per half and its report flavor. The report flavor is the weather, attendance, referee and two assistant referees. Flavor generation is best-effort: if a name can't be drawn, the flavor is skipped and the match result stands.

#### Scenario: Cup tie simulated
- **WHEN** a cup tie finishes simulating with 2 first-half and 5 second-half added minutes
- **THEN** it has 97 stored instants (1 to 45, 45+1 and 45+2, 46 to 90, 90+1 to 90+5) and a stored weather, attendance, referee and two assistants

#### Scenario: Friendly simulated
- **WHEN** a friendly (club, national-team or World Cup group) finishes simulating
- **THEN** its stored weather and attendance are the same values that set its ticket revenue

#### Scenario: Forfeit
- **WHEN** a cup tie with 3 first-half added minutes stops in minute 61 because one side dropped to 7 players
- **THEN** it stores the instants for minutes 1 to 45, 45+1 to 45+3 and 46 to 61, a second-half added time of 0, and its flavor, the same as a league match does

#### Scenario: Referee names unavailable
- **WHEN** drawing a referee name fails after a cup tie was simulated
- **THEN** the tie's result, events and instants are still stored, and its flavor stays empty

### Requirement: Default result record
A fixture settled after repeated simulation failures SHALL be stored as played 0-0 with a single `MATCH_DEFAULT_RESULT` event (type 7) at minute 0 and no instants, lineups or report flavor. Any events or instants left by failed attempts MUST be removed.

#### Scenario: League match defaulted
- **WHEN** a league match is settled 0-0 by default
- **THEN** both sides get a draw in the standings (1 point, 0 goals) and are charged their matchday wages
- **AND** no ticket revenue or sponsorship is booked, and no skill growth happens
- **AND** the matchday counts against both sides' already-suspended or injured players, and each owner gets the result notification

#### Scenario: Cup tie defaulted
- **WHEN** a cup tie is settled 0-0 by default
- **THEN** it goes to the existing shoot-out (each side wins with probability 0.5), stored as `went_to_penalties = true`, and the winner gets the round prize

#### Scenario: World Cup group match defaulted
- **WHEN** a World Cup group friendly is settled 0-0 by default
- **THEN** both national teams get a draw in the group standings

#### Scenario: Viewer opens a defaulted match
- **WHEN** a viewer opens a match settled by default
- **THEN** the page shows the final 0-0 and the default-result event instead of a live replay

### Requirement: Stored lineup is the XI that kicked off
A match's stored lineup snapshot MUST hold the XI after the pre-kickoff removal and refill of unavailable players, together with the scheduled substitutions it was played with. So the Lineups tab, the ratings view and the live viewer name the player who actually started in each slot.

#### Scenario: Injured player in a human's saved lineup
- **WHEN** a human's saved lineup names an injured midfielder and the refill puts the best available midfielder in his slot
- **THEN** the match's stored lineup lists the replacement in that slot and does not list the injured player

### Requirement: Scheduled substitutions
A scheduled substitution SHALL fire at its minute only if its outgoing player is still on the pitch and its team has used fewer than 3 substitutions, counting injury replacements. If its incoming player is not eligible, the best eligible bench player for that position comes on instead. Not eligible means unavailable for the match, sent off, injured this match, already on the pitch or already substituted off.

#### Scenario: Cap reached by injuries
- **WHEN** a team has made 3 injury replacements by minute 50 and has a substitution scheduled for minute 60
- **THEN** the scheduled substitution does not fire, so the team never makes more than 3 changes

#### Scenario: Incoming player got injured since the lineup was saved
- **WHEN** a human scheduled a striker to come on at minute 65 and that striker is now injured
- **THEN** at minute 65 the best eligible bench player for the outgoing player's position comes on instead, as a logged substitution

#### Scenario: No eligible bench player
- **WHEN** a scheduled substitution's incoming player is not eligible and nobody on the bench is
- **THEN** the substitution is skipped and does not count against the 3

#### Scenario: Outgoing player already off
- **WHEN** a scheduled substitution's outgoing player was sent off or injured earlier
- **THEN** it is skipped and does not count against the 3

### Requirement: Projected kickoff replacements in the lineup editor
Every lineup a team owner reads for an upcoming league match, cup tie or friendly, and his default lineups, SHALL list for each unavailable starter the player the pre-kickoff refill would put in his slot if the match kicked off now. It is computed with the same rules and slot order as the refill. The saved lineup itself is not changed.

#### Scenario: Injured starter in the default lineup
- **WHEN** an owner opens his default lineup and his starting left-back is injured
- **THEN** the left-back is shown as injured, along with the name of the best available player who would replace him

#### Scenario: Injured player recovers
- **WHEN** that left-back's injury runs out
- **THEN** he is back in the default lineup with no replacement shown, because the saved lineup was never edited

### Requirement: Home advantage
Every minute, after the zone blend, each of the home side's nine zone ratings (attack, midfield and defense; centre, left and right) SHALL be multiplied by `1 + 0.04 + 0.07 × fill`, where fill is the match's attendance divided by the home stadium's capacity, 0 when the capacity is 0. World Cup group friendlies and World Cup knockout ties MUST use a multiplier of 1.

#### Scenario: Full stadium
- **WHEN** a league match is played with attendance equal to the home stadium's capacity
- **THEN** every home zone rating is multiplied by 1.11

#### Scenario: Empty stadium
- **WHEN** a home match draws 0 spectators
- **THEN** every home zone rating is multiplied by 1.04

#### Scenario: Half-full stadium
- **WHEN** 6,000 people attend a 12,000-seat stadium
- **THEN** the multiplier is 1 + 0.04 + 0.07 × 0.5 = 1.075

#### Scenario: World Cup group match
- **WHEN** two national teams play a World Cup group friendly
- **THEN** neither side's ratings are multiplied

#### Scenario: Expected effect between even sides
- **WHEN** a large sample of league matches between sides with identical squads is simulated at a half-full stadium
- **THEN** the home side wins 38%–45% of them, the away side 30%–37%, and the home side scores 1.05–1.25 times as many goals

### Requirement: Team mentality
Every lineup SHALL carry a mentality: Defensive (−1), Balanced (0, the default) or Attacking (+1). Every minute, after the zone blend and the home advantage, an Attacking side's three attack ratings MUST be multiplied by 1.08 and its three defense ratings by 0.92; a Defensive side's the other way round. Midfield ratings and a Balanced side are unchanged.

#### Scenario: Attacking side
- **WHEN** an Attacking side has a central attack of 200 and a central defense of 180 after the blend
- **THEN** they are 216 and 165.6 for that minute

#### Scenario: Two Balanced sides
- **WHEN** both sides are Balanced
- **THEN** the match is simulated exactly as without mentality

#### Scenario: Lineup saved before this change
- **WHEN** a lineup saved before this change is used
- **THEN** it plays Balanced

### Requirement: Mentality effect sizes
Between sides with identical squads, an Attacking side facing a Balanced one SHALL score 8%–20% more goals and concede 8%–20% more than a Balanced side does against a Balanced side; a Defensive side facing a Balanced one SHALL concede 8%–20% fewer goals and score 8%–20% fewer.

#### Scenario: Calibration sample
- **WHEN** 20,000 matches of an Attacking side against a Balanced side with identical squads are simulated
- **THEN** the Attacking side's goals for and goals against are both 8%–20% above the Balanced-vs-Balanced average

### Requirement: Conditional match orders
A lineup that can schedule substitutions SHALL also hold up to 3 conditional orders, each with a from-minute from 1 to 89, a score condition (losing, losing by 2+, drawing, winning, winning by 2+, from the side's own point of view) and one action: set the mentality, or substitute a starter for a bench player. A saved order MUST pass the same player checks as a scheduled substitution.

#### Scenario: Valid orders
- **WHEN** a manager saves "from 60, if losing, go Attacking" and "from 75, if winning, striker A off for defender B"
- **THEN** both orders are stored with the lineup

#### Scenario: Too many orders
- **WHEN** a manager saves 4 orders
- **THEN** the lineup is rejected with a validation error

#### Scenario: Invalid substitution order
- **WHEN** an order's incoming player is also a starter
- **THEN** the lineup is rejected, the same as an invalid scheduled substitution

### Requirement: Order firing
In the substitution stage of each minute, after scheduled substitutions, every unfired order whose from-minute is at most the current clock minute SHALL be checked, in slot order, against the score at the start of that minute. An order whose condition holds fires once and is never checked again. Orders are also checked in added minutes. An order that never meets its condition by the last second-half added minute MUST NOT fire.

#### Scenario: Going behind late
- **WHEN** an order "from 60, if losing, go Attacking" exists and the side goes 0–1 down in minute 72
- **THEN** the side plays Attacking from minute 73 to the end

#### Scenario: Condition met before the from-minute
- **WHEN** a side is losing from minute 20 and its order is "from 60, if losing, go Attacking"
- **THEN** the order fires in minute 60

#### Scenario: Condition never met
- **WHEN** a side wins 2–0 and its only order is "from 60, if losing, go Attacking"
- **THEN** the order never fires and the side stays at its kickoff mentality

#### Scenario: Two orders on the same minute
- **WHEN** orders in slots 1 and 2 both become due and hold in the same minute
- **THEN** slot 1 fires first, then slot 2

#### Scenario: Going behind in added time
- **WHEN** an order "from 80, if losing, go Attacking" exists and the side goes behind at 90+1 of a match with 4 added minutes
- **THEN** the order fires at 90+2 and the side plays Attacking until 90+4

#### Scenario: From-minute in the first-half added time
- **WHEN** an order has from-minute 45 and its condition first holds at 45+2
- **THEN** it fires at 45+2

### Requirement: Substitution orders
A substitution order that fires SHALL be applied with the scheduled-substitution rules: it needs its outgoing player on the pitch and fewer than 3 substitutions used, counting injury replacements and scheduled substitutions, and an ineligible incoming player is replaced by the best eligible bench player for that position. A substitution order that cannot be applied MUST be spent without counting against the 3.

#### Scenario: Cap already reached
- **WHEN** a side has used 3 substitutions and a substitution order's condition holds
- **THEN** the order is spent and no substitution happens

#### Scenario: Outgoing player sent off
- **WHEN** a substitution order's outgoing player was sent off earlier
- **THEN** the order is spent without a substitution

### Requirement: Mentality change event
A fired mentality order that changes the side's mentality SHALL be recorded as a `MENTALITY_CHANGE` event (type 8) at that minute, naming the team and the new mentality. An order that sets the mentality the side already has is spent without an event.

#### Scenario: Switch to Defensive
- **WHEN** a Balanced side's order "from 75, if winning, go Defensive" fires in minute 75
- **THEN** a `MENTALITY_CHANGE` event for minute 75 with mentality Defensive is stored

#### Scenario: Already Attacking
- **WHEN** an Attacking side's order "go Attacking" fires
- **THEN** no event is stored

### Requirement: Lineup roles
Every lineup SHALL be able to name a captain, a penalty taker and a free-kick taker, each optional ("automatic" when empty). Each named player MUST be in that lineup's XI or bench; the two takers MUST NOT be goalkeepers. A named player who later leaves the squad is treated as automatic.

#### Scenario: Roles saved
- **WHEN** a manager names his number 10 as free-kick taker and his centre-back as captain
- **THEN** both are stored with the lineup and the penalty taker stays automatic

#### Scenario: Goalkeeper as taker
- **WHEN** a manager names his goalkeeper as penalty taker
- **THEN** the lineup is rejected with a validation error

#### Scenario: Taker sold
- **WHEN** the named penalty taker is sold after the lineup was saved
- **THEN** the lineup's penalty taker is automatic

### Requirement: Captain's calming effect
When a card is drawn for an outfield player whose side has its captain on the pitch, and that player is not the captain, the card SHALL be cancelled with probability 10% and nothing is recorded. A side without a captain on the pitch MUST be booked exactly as today.

#### Scenario: Captain on the pitch
- **WHEN** a card is drawn for a midfielder whose captain is playing
- **THEN** with probability 10% no card is shown, and otherwise the card is shown as usual

#### Scenario: Captain booked
- **WHEN** the card is drawn for the captain himself
- **THEN** it is never cancelled

#### Scenario: Captain sent off
- **WHEN** the captain was sent off earlier in the match
- **THEN** his team-mates' cards are no longer cancelled

### Requirement: Cup tie level after added time
A cup tie that is level after the last second-half added minute SHALL go to the existing penalty shoot-out. There is no extra time.

#### Scenario: Equaliser in added time
- **WHEN** a cup tie's away side equalises at 90+3 and no other goal follows
- **THEN** the tie goes to the shoot-out, stored as `went_to_penalties = true`

## Code map

- Pipeline and pre/post-processing: `backend/engine/simulation.go` (`runMatchSimulation`,
  `simulateMatch`, `excludeUnavailablePlayers`, `applyForfeitIfUnderstaffed`),
  `engine/cups.go` (`simulateCupTie`), `engine/friendlies.go` (`simulateFriendly`). The refill is
  `common.BackfillEmptyPositions` (`common/lineup.go`, positions in `positionFillOrder`).
- Kickoff snapshot: `runMatchSimulation` takes a `lineupSnapshotStore` callback, called right after
  `excludeUnavailablePlayers` with `kickoffLineup(...)` (the XI rebuilt from `*PlayersByPosition`);
  `simulateMatch`, `simulateCupTie` and `simulateFriendly` pass `persistResolvedLineupSnapshot`,
  `persistCupTieLineupSnapshot` and `persistFriendlyLineupSnapshot`. `MatchSimulation.HomeLineup`/
  `AwayLineup` hold that kickoff XI too.
- Stages: `engine/stamina_simulation.go`, `engine/initial_simulation.go` (`blendLine`,
  `ZoneSpillover`, `applyHomeAdvantage`), `engine/scorer_simulation.go`, `engine/discipline_simulation.go`,
  `engine/injury_simulation.go` (`bringOnInjuryReplacement`), `engine/substitution_simulation.go`
  (`applyScheduledSubstitutions`, `bestEligibleBenchPlayer`, `teamSide`; substituted-off sets
  `MatchSimulation.HomeSubstitutedOff`/`AwaySubstitutedOff`).
- Set pieces: `engine/scorer_simulation.go` (`classifyChance`, `processSetPiece`,
  `bestOnPitchOutfielder`, `penaltyConversionProbability`, `freeKickConversionProbability`);
  `common.Event.SetPiece` and `common.SET_PIECE_*`; persisted in the `set_piece` column of
  `match_events`, `cup_tie_events` and `friendly_events` (`resources/sql/init82_match_event_set_piece.sql`).
  Defending keeper on every chance: `processChance` and `processSetPiece` fill
  `Event.OtherTeamPlayers`/`OtherTeamPlayersNames`; every event table stores them
  (`other_team_players`, `other_team_player_names`; cup and friendly columns added by
  `resources/sql/init108_event_other_team_players.sql`), written by `common.replaceMatchEventsTx`
  and read by `GetMatchEvents`, `GetCupTieEvents` and `GetFriendlyEvents`. Tests:
  `engine/scorer_simulation_test.go` (`TestProcessChance_OpenPlayRecords*`),
  `common/open_play_keeper_integration_test.go`.
  Frontend: `utils/matchCommentary.tsx` (set-piece phrasing), `utils/matchGoals.ts` and
  `features/match/MatchScoreboard.tsx` (`(pen.)` / `(f.k.)` marks), `utils/matchEvents.ts`
  (a missed penalty always shows in `MatchEventsReport`).
- Home advantage: `engine/initial_simulation.go` (`HomeAdvantageBase`, `HomeAdvantageCrowdWeight`,
  `homeAdvantageMultiplier`, `homeAdvantageFor`, `applyHomeAdvantage`, called in
  `InitialSimulation.Apply` right after `calculateQua`); `common.MatchSimulation.HomeAdvantage`
  (0 counts as 1), passed in through `runMatchSimulation`'s `homeAdvantage` parameter. Neutral
  fixtures: `common.IsWorldCupGroupMatch` and `common.IsWorldCupKnockoutCup` (`common/worldcup.go`).
  Calibration: `engine/home_advantage_test.go` (`TestHomeAdvantageCalibration`, `HOME_ADV_SIM=1`).
- Team mentality: constants, validation and storage helpers in `common/mentality.go`
  (`MENTALITY_*`, `MentalityAttackFactor` / `MentalityDefenseFactor`, `MentalityChanceShift`,
  `MentalityFatigueShift`, `ValidateMentality`, `FatigueMaxPenaltyFor`, `Get/SetTeamMentality`,
  `Get/SetTeamFriendlyMentality`); `common.Lineup.Mentality`, stored in the `mentality` column of
  `lineup`, `cup_tie_lineup` and `friendly_lineup` and in `teams.mentality` /
  `teams.friendly_mentality` (`resources/sql/init95_lineup_mentality.sql`), written by
  `SetLineupForMatch` / `SetCupTieLineup` / `SetFriendlyLineup` (so the kickoff snapshot keeps it)
  and read by every lineup getter. `MatchSimulation.HomeMentality` / `AwayMentality` are set from
  the kickoff lineups in `runMatchSimulation` and read by `applyMentality`
  (`engine/initial_simulation.go`, after `applyHomeAdvantage`), `chanceProbability`
  (`engine/scorer_simulation.go`) and `applyFatigue` (`engine/stamina_simulation.go`, via
  `common.FatigueMultiplierWithPenalty`). Tests: `engine/mentality_test.go` (including
  `TestMentalityCalibration`, `MENTALITY_SIM=1`), `common/mentality_integration_test.go`,
  `handlers/lineupMentality_test.go`.
- Conditional orders: `common/lineuporders.go` (`LineupOrder`, `ORDER_CONDITION_*`,
  `ORDER_ACTION_*`, `MaxConditionalOrders`, `OrderConditionHolds`, `ValidateOrders`, storage per
  lineup kind, `GetOrdersForMatch` / `GetOrdersForCupTie` / `GetFriendlyLineupOrders`); tables
  `default_lineup_orders`, `lineup_orders`, `cup_tie_lineup_orders`, `friendly_lineup_orders` and
  the `mentality` column of the three event tables (`resources/sql/init96_lineup_orders.sql`).
  `simulateMatch` / `simulateCupTie` / `simulateFriendly` put the resolved orders on the incoming
  lineups; `runMatchSimulation` moves them to `MatchSimulation.HomeOrders` / `AwayOrders` (the
  kickoff lineups never carry them) and the snapshot callback stores them. Firing:
  `engine/substitution_simulation.go` (`applyConditionalOrders` after `applyScheduledSubstitutions`,
  `applySubstitution` shared by both, `teamSide.orders` / `ordersFired` / `mentality`);
  `common.MENTALITY_CHANGE` events carry `Event.Mentality`. Lineup GET handlers fill
  `Lineup.Orders` for the owner only (`withOwnerOrders` in `handlers/lineupHandlers.go`). Tests:
  `engine/conditional_orders_test.go`, `engine/conditional_orders_integration_test.go`,
  `common/lineuporders_integration_test.go`, `handlers/lineupOrders_test.go`.
- Lineup roles: `common/lineuproles.go` (`LineupRoles` embedded in `Lineup`, `ValidateRoles`,
  `rolesSelectSql` reading a sold player's role as NULL, `Get`/`SetDefaultLineupRoles`,
  `RedactTakers` for the public match views, `CaptainCardCancelProbability`); columns on `lineup`,
  `cup_tie_lineup`, `friendly_lineup` and table `default_lineup_roles`
  (`resources/sql/init97_lineup_roles.sql`). Taker choice: `processSetPiece` with
  `namedOnPitchOutfielder` / `sideRoles` (`engine/scorer_simulation.go`); captain:
  `captainCancelsCard` in `DisciplineSimulation.Apply` (`engine/discipline_simulation.go`). Tests:
  `engine/lineup_roles_test.go`, `engine/lineup_roles_integration_test.go`,
  `common/lineuproles_integration_test.go`, `handlers/lineupRoles_test.go`.
- Crowd before kickoff: `simulateMatch`, `simulateCupTie` and `simulateFriendly` call
  `engine.drawMatchdayConditions` (`engine/stadium.go`: weather, then `drawMatchdayCrowd` returning a
  `matchdayCrowd`) before `runMatchSimulation`, and after the save commits pass that crowd to
  `UpdateTeamFinancesAfterMatch` / `creditFriendlyTicketRevenue` and `updateMatchReportFlavor`.
  Covered by `engine/home_advantage_integration_test.go`.
- Post-match processing (atomic, only-once result save): `common.SaveLeagueMatchResultTx`,
  `common.SaveFriendlyResultTx` and `common.SaveCupTieResultTx` (`common/matchresult.go`) write the
  result, events, instants, standings (under `LockNamespaceStandings`) and the pending post-match
  record in one transaction guarded by `played = false` (`common.ErrAlreadyPlayed`); money, flavor
  and the broadcast follow the commit in `simulateMatch`, `simulateCupTie` and `simulateFriendly`.
  Default result: `engine.SettleDefaultResults` (`engine/defaultresults.go`) and
  `common.MATCH_DEFAULT_RESULT`.
- Stored match record for every competition: per-minute instants of a cup tie are read back by
  `common.GetCupTieInstants`, events by `common.GetFriendlyEvents` / `GetCupTieEvents`; report flavor through
  `common.UpdateMatchFlavor` / `UpdateCupTieFlavor` / `UpdateFriendlyFlavor` (`common/matchflavor.go`),
  all driven by `engine.updateMatchReportFlavor`; tables `cup_tie_instants` and the flavor columns of
  `cup_ties` and `friendly_matches` (`resources/sql/init83_match_viewer_parity.sql`).
- Availability: `common.GetUnavailablePlayerIds`/`GetUnavailablePlayerIdsForFriendly`
  (`common/database.go`); substitutions cap `common.MaxScheduledSubstitutions`.
- Rivalry level: `MatchSimulation.RivalryLevel`, passed to `runMatchSimulation` by `simulateMatch` /
  `simulateCupTie` (read at kickoff with `common.KickoffRivalryLevel`; friendlies pass none), sets the
  card rate through `cardIncidentProbability` (`ProbCardIncidentRival`, `ProbCardIncidentClasico` in
  `engine/discipline_simulation.go`) and is stored with the result (`MatchResult.RivalryLevel`,
  `matches.rivalry_level` / `cup_ties.rivalry_level`).
- Added time (see stoppage-time): `engine/stoppage_time.go` (the `Stoppage*` costs, `MinAddedMinutes`,
  `MaxAddedFirstHalf` / `MaxAddedSecondHalf`, `timeLost`, `addedMinutesFor`) and the minute loop in
  `engine/simulation.go` (`playMatchMinutes`, `playRegulation`, `playAddedTime`, `playMinute` - which
  stamps `Event.AddedMinute` on whatever a minute records - and `announceAddedTime` for the
  `common.ADDED_TIME` board). `MatchSimulation.AddedMinute` / `PlayedMinutes` /
  `FirstHalfAddedTime` / `SecondHalfAddedTime`, `MatchInstant.AddedMinute` (set in
  `InitialSimulation.Apply`), `Event.AddedTime`; scheduled substitutions skip added time in
  `applyScheduledSubstitutions`. Stored through `MatchResult.FirstHalfAddedTime` /
  `SecondHalfAddedTime` (`engine.newMatchResult`, `common.Save*ResultTx`) in the
  `added_minute` / `added_time` columns of the event and instant tables and the
  `first_half_added_time` / `second_half_added_time` columns of `matches`, `cup_ties` and
  `friendly_matches` (`resources/sql/init107_stoppage_time.sql`); readers order by
  `minute, added_minute, event_id`. Tests: `engine/stoppage_time_test.go`,
  `engine/conditional_orders_test.go` (added-time orders), `engine/stamina_simulation_test.go`
  (`TestApplyFatigue_AddedTimeIsCappedAtTheFullTimePenalty`),
  `engine/match_viewer_parity_integration_test.go` (`assertStoredAddedTime`),
  `common/stoppage_time_integration_test.go`. The calibration helpers (`playEvenMatch`,
  `measureMentality`) play the real loop, added time included.
