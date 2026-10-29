# Spec Delta

## MODIFIED Requirements

### Requirement: Live spoiler withholding
In the live view, the viewer SHALL hide injury and red-card events, for every competition, until the replay completes. Live per-player ratings are shown, but they MUST be computed without the withheld injury and red-card events. The finished view shows every event and the stored ratings.

#### Scenario: Red card in a live cup tie
- **WHEN** a player is sent off in minute 30 of a cup tie whose replay is at minute 45
- **THEN** the live chronicle doesn't list the red card
- **AND** the finished view lists it once the replay completes

#### Scenario: Live rating ignores the red card
- **WHEN** the same player's live rating is shown at minute 45
- **THEN** it doesn't include the −1.5 for the red card, and the finished view's rating does

## ADDED Requirements

### Requirement: Ratings list everyone who featured
The ratings view SHALL list, per team, every player who has a rating for the match: starters in lineup order, then substitutes in order of the minute they came on. Each row MUST show the player's position, his minutes (with "↑ N'" for a player who came on and "↓ N'" for one who went off or was sent off), his mean zone contribution against the rival's (for example "58 vs 45"), and the rating.

#### Scenario: Starter replaced at 60
- **WHEN** a finished match had the home left winger replaced in minute 60
- **THEN** the home list shows the winger with "↓ 60'" and his own rating, and the substitute after the starters with "↑ 60'" and his own rating

#### Scenario: Unrated substitute
- **WHEN** a substitute came on in minute 85
- **THEN** his row shows "↑ 85'", his contribution and "s.c." in place of the rating chip

#### Scenario: Match simulated before stored ratings
- **WHEN** the match has no stored contributions
- **THEN** the rows show position, minutes and rating, and no contribution figure

### Requirement: Rating breakdown on demand
Tapping or hovering a rating SHALL show its breakdown: base 6.0, zone component, goals, cards, injury and result, each to one decimal, with the components that are 0 left out.

#### Scenario: Scorer in a win
- **WHEN** a viewer taps the 9.6 of a striker who scored twice in a win with a zone component of +0.5
- **THEN** the breakdown reads "Base 6.0 · Zona +0.5 · Goles +2.8 · Resultado +0.3"

### Requirement: Ratings by minute
Ratings shown live or at a minute picked on the scrubber SHALL use the rating formula with only the contributions, events and score up to that minute, and minutes counted up to that minute. At the match's final minute, the finished view MUST show the stored ratings.

#### Scenario: Scrubbing back to half-time
- **WHEN** a viewer scrubs a finished 2-1 home win back to minute 45, when it was 0-1
- **THEN** each home player's result component is negative, scaled by his minutes so far, and only events up to minute 45 count

#### Scenario: Back to full time
- **WHEN** the viewer moves the scrubber back to the final minute
- **THEN** every rating equals the stored rating shown in the breakdown
