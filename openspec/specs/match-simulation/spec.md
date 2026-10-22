# match-simulation Specification

## Purpose

How a single match (league, cup tie or friendly) is turned into a result: which players take the
field, how team strength is rated each minute, and how chances, goals, cards, injuries, fatigue and
forfeits are decided. All three competitions share the same engine.

## Requirements

### Requirement: Per-minute simulation pipeline
The simulator SHALL run, for every minute from 1 to 90, the stages in this order: substitutions, stamina (fatigue), team qualification, scorer, discipline, injury.

#### Scenario: A minute is simulated
- **WHEN** minute N is simulated
- **THEN** due scheduled substitutions are applied first
- **AND** fatigue-adjusted player copies are rebuilt before qualification and scoring read them
- **AND** cards and injuries are rolled after the scoring stage

### Requirement: Unavailable players never take the field
Before kickoff the simulator MUST remove from the saved lineup every outfield player serving an injury or suspension, then refill each thinned position back to its original headcount with the best remaining eligible squad player for that position. If the squad runs out, the position stays short.

#### Scenario: Suspended striker in the saved lineup
- **WHEN** a suspended striker is still in a team's saved lineup
- **THEN** he does not play and the best available squad player for that position takes his slot

#### Scenario: Squad crisis
- **WHEN** no eligible player is left for a position
- **THEN** that position is left short rather than fielding anyone twice or anyone unavailable

#### Scenario: Suspensions in friendlies
- **WHEN** a player serving a red-card suspension is in a friendly lineup
- **THEN** he may play, because suspensions only bar league and cup matches (injuries still apply)

### Requirement: In-match fatigue
Each minute, every player's seven core skills SHALL be scaled by `1 - 0.30 × (minutes played / 90) × ((99 - Stamina) / 99)`, and every later stage MUST read these effective skills.

#### Scenario: Stamina 99
- **WHEN** a Stamina-99 player has played 90 minutes
- **THEN** his effective skills equal his raw skills

#### Scenario: Stamina 0 at full time
- **WHEN** a Stamina-0 player has played 90 minutes
- **THEN** each of his core skills is reduced by 30%

#### Scenario: Not yet on the pitch
- **WHEN** a player has played 0 minutes this match
- **THEN** he is unaffected whatever his Stamina

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
Each minute SHALL produce an attacking chance with probability 14% plus up to 10 points scaled by midfield dominance. The attacking side MUST be chosen by a midfield-weighted share bounded to 32%–68%.

#### Scenario: Even midfields
- **WHEN** both midfields are equal
- **THEN** a chance occurs with 14% probability and each side gets it half of the time

#### Scenario: Total midfield dominance
- **WHEN** one side holds almost all the midfield strength
- **THEN** the chance probability approaches 24% and that side gets at most 68% of chances

### Requirement: Chance resolution
A chance SHALL be central 50%, left 25% or right 25%, taken by a striker 70%, midfielder 20% or defender 10% (falling back to any remaining line), and converted with a logistic probability bounded to 5%–65% and centered on 28% for an even matchup.

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

### Requirement: Discipline
Each minute SHALL have a 3% chance of a card for a random outfield player of either side; 6% of first cards are straight reds, and a second yellow becomes a red. A sent-off player MUST leave the pitch unreplaced. Goalkeepers are never booked.

#### Scenario: Straight red in a league match
- **WHEN** a player gets a straight red in a league or cup match
- **THEN** he is removed for the rest of the match and banned for 2 matches

#### Scenario: Second yellow
- **WHEN** a player already booked is booked again in a league or cup match
- **THEN** he is sent off and banned for 1 match

#### Scenario: Red card in a friendly
- **WHEN** a player is sent off in a friendly
- **THEN** he leaves the match but carries no suspension

### Requirement: Injuries
Each minute SHALL have a 0.5% chance of injuring a random outfield player, removing him for the rest of the match and sidelining him for 1–5 matches. If the team has not used its 3 substitutions, the best available bench player MUST replace him as a logged substitution. Goalkeepers are never injured.

#### Scenario: Substitutions left
- **WHEN** a player is injured and his team has used 2 of 3 substitutions
- **THEN** the best available bench player comes on in his place

#### Scenario: No substitutions left
- **WHEN** a player is injured and all 3 substitutions are used
- **THEN** the team plays on a man down

### Requirement: Forfeit when a side runs out of players
Before kickoff and after every minute the simulator MUST check whether exactly one side has 7 or fewer players on the pitch; if so the match ends there, recorded as a 3–0 win for the other side with a `MATCH_ABANDONED` event.

#### Scenario: Fourth red card
- **WHEN** a side drops to 7 players
- **THEN** no further minutes are simulated and the result is 3–0 to the opponent, whatever the score was

#### Scenario: Both sides collapse together
- **WHEN** both sides are at 7 or fewer at the same check
- **THEN** no forfeit is awarded and the match plays on

### Requirement: Post-match processing
After a league match the engine SHALL record the result and standings, update finances, charge both sides' matchday wages, and store the match's attendance and weather in the report. The weather MUST be drawn once and used both for attendance and the report.

#### Scenario: League match finishes simulating
- **WHEN** a league match has been simulated
- **THEN** ticket revenue, sponsorship and wages are booked immediately
- **AND** availability updates, skill growth and the result notification wait for the live replay (see match-scheduling)

## Code map

- Pipeline and pre/post-processing: `backend/engine/simulation.go` (`runMatchSimulation`,
  `simulateMatch`, `excludeUnavailablePlayers`, `backfillEmptyPositions`,
  `applyForfeitIfUnderstaffed`), `engine/cups.go` (`simulateCupTie`), `engine/friendlies.go`
  (`simulateFriendly`).
- Stages: `engine/stamina_simulation.go`, `engine/initial_simulation.go` (`blendLine`,
  `ZoneSpillover`), `engine/scorer_simulation.go`, `engine/discipline_simulation.go`,
  `engine/injury_simulation.go` (`bringOnInjuryReplacement`).
- Availability: `common.GetUnavailablePlayerIds`/`GetUnavailablePlayerIdsForFriendly`
  (`common/database.go`); substitutions cap `common.MaxScheduledSubstitutions`.
