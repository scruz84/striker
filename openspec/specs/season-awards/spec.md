# season-awards Specification

## Purpose

The individual honors each league hands out when its country's season rolls over (Golden Boot and
Player of the Season), how winners are chosen, notified and kept on the player's profile.

## Requirements

### Requirement: Golden Boot
At season rollover each league SHALL award the Golden Boot to its top scorer in its league matches for the whole season (including matches still being replayed live when the season closes), from the player match stats, ties to the lower player id. A scorer who retired in that same rollover can win it. It MUST be skipped when nobody scored.

#### Scenario: Top scorer exists
- **WHEN** a league's season ends and its top scorer has 14 goals
- **THEN** that player wins the Golden Boot

#### Scenario: No goals
- **WHEN** no goal was scored in the league all season
- **THEN** no Golden Boot is awarded

### Requirement: Player of the Season
Each league SHALL award Player of the Season to the player with the highest score from that season's league match stats: +3 per goal, −1 per yellow card, −3 per red card, ties to the lower player id. A player who retired in that same rollover can win it. It MUST be skipped when no score is positive.

#### Scenario: Goals and cards
- **WHEN** a player scored 5 goals and got 2 yellow cards
- **THEN** his score is 13

#### Scenario: Retired in the rollover
- **WHEN** the season's best scorer of 9 points retires before the awards are handed out
- **THEN** he is still Player of the Season

### Requirement: Award delivery and record
Each award SHALL be stored permanently in `player_season_awards` and notified to the winner's team owner as `NOTIFICATION_AWARD` (distinguished by message key) linking to the player's page. The player's profile MUST show his Golden Boot and MVP counts. The record MUST outlive the player: a winner who retires keeps his awards (see retired-players).

#### Scenario: Winner plays for an owned team
- **WHEN** a player wins the Golden Boot
- **THEN** his owner gets a `notifications.messages.goldenBoot` notification linking to `/player/{id}`
- **AND** his profile's Golden Boot counter increases by one

#### Scenario: Winner retires
- **WHEN** a Golden Boot winner retires two seasons later
- **THEN** his profile still counts that Golden Boot

### Requirement: Awards run alongside season-end money
Awards SHALL be computed per league at rollover, alongside prize money and season-ticket/merchandising payouts and before standings are reset. The team of the season (see best-xi) is picked with them.

#### Scenario: Rollover order
- **WHEN** a country's season rolls over
- **THEN** each league pays prize money and season income and hands out its awards and its team of the season before promotion/relegation is applied

## Code map

- `backend/engine/awards.go` (`awardSeasonHonors`), called from `applyPromotionRelegation`
  (`engine/season.go`).
- `common/seasonawards.go` (`GetPlayerOfTheSeason`, weights), `common.GetTopScorersFinal`
  (`common/topscorers.go`, over `common.GetLeagueLeaders` in `common/playerstatsqueries.go`),
  `common.RecordPlayerSeasonAward`, `common.GetPlayerSeasonAwards`.
- The team of the season: `engine/bestxi.go` (`awardSeasonBestXi`), see openspec/specs/best-xi.
- Backfill of earlier seasons: `resources/sql/history/init76_player_season_awards.sql`.
