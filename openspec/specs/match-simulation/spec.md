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
A penalty SHALL be taken by the attacking side's on-pitch outfield player with the highest fatigue-adjusted Shot, ties going to the lowest player id. It is scored with probability `0.78 + 0.30 × (taker Shot − keeper GoalKeeper) / 99`, bounded to 60%–92%. It uses neither the open-play clear-cut roll nor the flank logic.

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
- **WHEN** the side's highest-Shot player has been sent off, injured off or substituted
- **THEN** the best remaining on-pitch outfield player takes the penalty

#### Scenario: No outfield player left
- **WHEN** the attacking side has no outfield player on the pitch
- **THEN** the penalty is discarded, the same as an open-play chance

### Requirement: Direct free-kick resolution
A direct free kick SHALL be taken by the attacking side's on-pitch outfield player with the highest `0.6 × Shot + 0.4 × Pass` (fatigue-adjusted), ties going to the lowest player id. It is scored with probability `0.06 + 0.10 × (taker value − keeper GoalKeeper) / 99`, bounded to 2%–15%.

#### Scenario: Even taker and keeper
- **WHEN** the taker's value equals the keeper's GoalKeeper
- **THEN** the free kick is scored with probability 6%, matching the real-world ~5–6% conversion

#### Scenario: Specialist taker
- **WHEN** the taker has Shot 75 and Pass 70 (value 73) and the keeper has GoalKeeper 40
- **THEN** the probability is 0.06 + 0.10 × 33/99 ≈ 0.093

#### Scenario: Bounds
- **WHEN** the taker value exceeds the keeper's GoalKeeper by 99, or trails it by 99
- **THEN** the probability is 15% and 2% respectively

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

#### Scenario: Result save fails
- **WHEN** writing the result, events, instants or standings fails
- **THEN** none of them is stored, no budget changes, no replay starts, and the match stays unplayed for the next attempt

#### Scenario: Leftovers from an earlier failed attempt
- **WHEN** a match still has events or instants stored by an earlier failed attempt
- **THEN** the successful save replaces them, so the match keeps exactly one set

### Requirement: Stored match record for every competition
Every simulated match SHALL store the same record, whatever its competition (league match, cup tie or friendly): its events, its per-minute zone instants (one per simulated minute) and its report flavor. The report flavor is the weather, attendance, referee and two assistant referees. Flavor generation is best-effort: if a name can't be drawn, the flavor is skipped and the match result stands.

#### Scenario: Cup tie simulated
- **WHEN** a cup tie finishes simulating after 90 minutes
- **THEN** it has 90 stored instants (minutes 1 to 90) and a stored weather, attendance, referee and two assistants

#### Scenario: Friendly simulated
- **WHEN** a friendly (club, national-team or World Cup group) finishes simulating
- **THEN** its stored weather and attendance are the same values that set its ticket revenue

#### Scenario: Forfeit
- **WHEN** a cup tie stops in minute 61 because one side dropped to 7 players
- **THEN** it stores the instants for minutes 1 to 61 and its flavor, the same as a league match does

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

## Code map

- Pipeline and pre/post-processing: `backend/engine/simulation.go` (`runMatchSimulation`,
  `simulateMatch`, `excludeUnavailablePlayers`, `backfillEmptyPositions`,
  `applyForfeitIfUnderstaffed`), `engine/cups.go` (`simulateCupTie`), `engine/friendlies.go`
  (`simulateFriendly`).
- Stages: `engine/stamina_simulation.go`, `engine/initial_simulation.go` (`blendLine`,
  `ZoneSpillover`), `engine/scorer_simulation.go`, `engine/discipline_simulation.go`,
  `engine/injury_simulation.go` (`bringOnInjuryReplacement`).
- Set pieces: `engine/scorer_simulation.go` (`classifyChance`, `processSetPiece`,
  `bestOnPitchOutfielder`, `penaltyConversionProbability`, `freeKickConversionProbability`);
  `common.Event.SetPiece` and `common.SET_PIECE_*`; persisted in the `set_piece` column of
  `match_events`, `cup_tie_events` and `friendly_events` (`resources/sql/init82_match_event_set_piece.sql`).
  Frontend: `utils/matchCommentary.tsx` (set-piece phrasing), `utils/matchGoals.ts` and
  `features/match/MatchScoreboard.tsx` (`(pen.)` / `(f.k.)` marks), `utils/matchEvents.ts`
  (a missed penalty always shows in `MatchEventsReport`).
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
