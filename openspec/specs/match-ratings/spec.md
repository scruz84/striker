# match-ratings Specification

## Purpose
Defines the 1-10 rating every player who featured in a match receives: who is rated, how the
number is built from his zone contribution, his events and the result, when it is computed and
stored, and how matches simulated before stored ratings existed are rated.

## Requirements

### Requirement: Everyone who featured is rated
Every player who was on the pitch for at least one simulated minute, or who appears in a goal, card or injury event of the match, SHALL receive exactly one rating for that match. This includes starters, players who came on, players who came off, and players sent off or injured. A substitution MUST NOT move events, minutes or the rating from one player to another.

#### Scenario: Substitution in minute 60
- **WHEN** a starter scores in minute 20 and is replaced in minute 60
- **THEN** the starter is rated with 60 minutes and his goal, and the substitute is rated separately with 30 minutes and no goal

#### Scenario: Sent off
- **WHEN** a defender is sent off in minute 35
- **THEN** he is rated with 35 minutes, his red card, and his zone contribution from those 35 minutes

### Requirement: Rating formula
A rating SHALL be 6.0 plus the zone component, the event deltas (goal +1.4, yellow −0.4, red −1.5, injury −0.2) and the result component: ±0.3 × minutes / 90 for a win or loss (minutes capped at 90), 0 for a draw. Each component is rounded half away from zero to one decimal before summing, and the sum MUST be clamped to [1.0, 10.0]. Form and any per-player jitter MUST NOT affect it.

#### Scenario: Striker with two goals in a win
- **WHEN** a striker plays 90 minutes of a win, scores twice and has a zone component of +0.5
- **THEN** his rating is 6.0 + 0.5 + 2.8 + 0.3 = 9.6

#### Scenario: Late substitute in a defeat
- **WHEN** a midfielder plays the last 30 minutes of a defeat with a zone component of −0.2 and no events
- **THEN** his rating is 6.0 − 0.2 − 0.1 = 5.7

#### Scenario: Clamp
- **WHEN** a player's components add up to 10.8
- **THEN** his rating is 10.0

### Requirement: Short appearances are unrated
A player whose minutes on the pitch are fewer than 10 SHALL be unrated ("s.c.", sin calificar) for that match: his record keeps minutes, contribution and breakdown, but he has no rating number. When ratings are shown as of an earlier minute, the threshold is the smaller of 10 and the minutes elapsed so far.

#### Scenario: Late substitute
- **WHEN** a player comes on in minute 85 and plays 6 minutes
- **THEN** his record shows 6 minutes and "s.c." instead of a rating

#### Scenario: Early in the match
- **WHEN** ratings are shown as of minute 5
- **THEN** every player on the pitch since kickoff has 5 minutes and is rated

#### Scenario: Exactly 10 minutes
- **WHEN** a player plays exactly 10 minutes
- **THEN** he is rated

### Requirement: Zone contribution per minute
Every simulated minute, the engine SHALL record, for each player on the pitch, his position, his own value in the zone duels his position feeds, and the rival's per-player value in the mirrored duels. Both are computed from the same fatigue-adjusted skills and the same position weights the minute's zone strengths use. A player who feeds two duels gets the mean of each figure.

#### Scenario: Central defender
- **WHEN** a central defender with effective Defense 58 faces two central strikers whose effective Shot totals 90
- **THEN** his contribution that minute is 58 against 45

#### Scenario: Full-back feeds two duels
- **WHEN** a left full-back's flank-defense value is 50 against the rival right attack's 40, and his flank-midfield value is 54 against the rival right midfield's 48
- **THEN** his contribution that minute is 52 against 44

#### Scenario: Goalkeeper
- **WHEN** a goalkeeper with effective GoalKeeper 62 faces a rival whose attacking players' per-player values average 50
- **THEN** his contribution that minute is 62 against 50

#### Scenario: Empty rival zone
- **WHEN** the rival has no player left in the mirrored duel after a red card
- **THEN** the rival value is 0 and that minute's edge is +1

### Requirement: Zone component
The zone component SHALL be 3.0 × the mean edge over the minutes whose contribution record lists the player, clamped to [−1.5, +1.5]. Each minute's edge is (own − rival) / (own + rival), and 0 when both are 0. A player's minutes on the pitch are the number of those minutes, and a player with none has a zone component of 0.

#### Scenario: Dominant full match
- **WHEN** a player's edge is 0.15 in every one of his 90 minutes
- **THEN** his zone component is 3.0 × 0.15 = 0.45, rounded half away from zero to +0.5

#### Scenario: Clamped mismatch
- **WHEN** a player's mean edge is 0.6
- **THEN** his zone component is +1.5

### Requirement: Ratings stored at simulation time
A match's ratings and its per-minute contributions SHALL be computed when the match is simulated and saved in the same transaction as its result, for league matches, cup ties and friendlies. Reading a match MUST return the stored values, so a rating never changes after the match because a player's skills or Form changed.

#### Scenario: Player improves after the match
- **WHEN** a player's Defense rises by 3 the week after a match
- **THEN** that match still shows the rating and contribution it showed at full time

#### Scenario: Failed save
- **WHEN** the result transaction of a match fails
- **THEN** no rating or contribution of that match is stored

### Requirement: Rating breakdown
For every rating, the stored record SHALL expose the player's team, main position (where he played the most minutes), minutes played, the minute he came on and went off (when either happened), his mean own and rival contribution, and the zone, events and result components.

#### Scenario: Substitute's record
- **WHEN** a winger comes on in minute 70 of a 1-0 win and plays 20 minutes with a mean contribution of 47 against 51
- **THEN** his record shows minutes 20, came on 70, contribution 47 vs 51, a negative zone component, and a result component of +0.1

### Requirement: Ratings of matches simulated before stored ratings
A played match with no stored ratings SHALL be rated on read from its starting lineups and events. Everyone who started or came on is rated. Events are credited to the player named in them. Minutes come from the substitution, red-card and injury minutes. The zone component is 0 because these matches have no per-minute contributions.

#### Scenario: Old match with a substitution
- **WHEN** a league match simulated before this change had a starter who scored and was replaced in minute 60
- **THEN** both the starter and the substitute are rated, the goal counts for the starter, and the substitute's result component uses 30 minutes

#### Scenario: Old match without lineups
- **WHEN** an old match has no stored lineup for either side
- **THEN** it has no ratings, as before

## Code map

- Formula: `backend/common/matchratings.go` (`ComputePlayerRatings`, `PlayerRating`, `roundTenth`,
  `RatingsById`, `MatchFinalMinute`, `MatchRatingMinMinutes`).
- Per-minute contributions: `backend/engine/initial_simulation.go` (`playerContributions`,
  `newSideDuelists`, `lateralDefenseValue` / `lateralMidfieldValue` / `lateralAttackValue`, shared
  with `calculateQua`); stored as `common.InstantContributions` on `MatchInstant`.
- Computed once per simulated fixture in `engine.newMatchResult` (`engine/postmatch.go`), which also
  puts the rated players' ratings on the post-match payload for Form (see player-development).
- Storage and read paths: `backend/common/matchresult.go` (`replaceMatchPlayerRatingsTx`,
  `GetMatchPlayerRatings`, `LoadPlayerRatings`, `addRatedPlayers`, `encodePlayerContributions` /
  `decodePlayerContributions`), called from `WeekMatchById`, `GetCupTieDetails` and
  `EnrichPlayedFriendly`; table `match_player_ratings` and the `player_contributions` columns from
  `resources/sql/init89_match_player_ratings.sql`.
- Frontend port: `frontend/src/utils/liveMatchRatings.ts` (`computePlayerRatings`, `startingLineup`),
  pinned to the Go formula by `backend/common/testdata/match_ratings_parity.json`
  (`TestComputePlayerRatings_ParityFixture`, `liveMatchRatings.test.ts`; regenerate with
  `STRIKER_WRITE_RATINGS_FIXTURE=1`).
