# match-viewer Specification

## Purpose

The single match page that presents any match (league match, cup tie or friendly, World Cup
fixtures included) before kickoff, live and after full time. The goal is that every competition
shows a viewer the same information in the same layout.

## Requirements

### Requirement: One viewer for every competition
League matches (`/match/:id`), cup ties (`/cup/ties/:id`, which includes international-cup and World Cup knockout ties) and friendlies (`/friendlies/:id`, which includes national-team and World Cup group matches) SHALL all render through the same match viewer. The viewer MUST show the same sections in the same order for every competition. A section may be hidden only when the match has no data for it.

#### Scenario: Cup tie looks like a league match
- **WHEN** a user opens a played cup tie
- **THEN** they see the scoreboard, then the Crónica / Notas / Tácticas tabs, then the head-to-head card, exactly as for a played league match

#### Scenario: Existing links keep working
- **WHEN** a notification, cup bracket, team page or World Cup page links to `/match/:id`, `/cup/ties/:id` or `/friendlies/:id`
- **THEN** the link opens the shared viewer for that match

### Requirement: Viewer phases
The viewer SHALL show one of three views:
- **Scheduled**, before kickoff: the scoreboard marked as scheduled, plus the lineup section.
- **Live**, from kickoff until the live replay reports completion: the live scoreboard, lineups pitch, chronicle, zone duel, minute-by-minute ratings and head-to-head card.
- **Finished**: the scoreboard and the three tabs.

#### Scenario: Opening a match during its broadcast
- **WHEN** a viewer opens a friendly 20 minutes after kickoff while its replay is still running
- **THEN** the live view is shown, caught up to minute 20, and it switches to the finished view only when the replay completes

#### Scenario: Played flag set before kickoff
- **WHEN** a match is already marked played 10 minutes before its advertised kickoff
- **THEN** the viewer still shows the scheduled view until kickoff

### Requirement: Live spoiler withholding
In the live view, the viewer SHALL hide injury and red-card events, for every competition, until the replay completes. Live per-player ratings are shown, but they MUST be computed without the withheld injury and red-card events. The finished view shows every event and the stored ratings.

#### Scenario: Red card in a live cup tie
- **WHEN** a player is sent off in minute 30 of a cup tie whose replay is at minute 45
- **THEN** the live chronicle doesn't list the red card
- **AND** the finished view lists it once the replay completes

#### Scenario: Live rating ignores the red card
- **WHEN** the same player's live rating is shown at minute 45
- **THEN** it doesn't include the −1.5 for the red card, and the finished view's rating does

### Requirement: Scoreboard subtitle
The scoreboard subtitle SHALL read `<competition label> · <kickoff date and time> · <weather emoji>`. Any part that is unknown is left out. The labels are:
- League match: "<division name> · Jornada N de T" (just "Jornada N" while the division details are still loading).
- Cup tie: "<cup name> · <round name>".
- World Cup group match: "Mundial · Grupo X · Jornada N".
- Any other friendly: "Amistoso".

#### Scenario: Cup quarter-final in the rain
- **WHEN** a played quarter-final of "Copa del Rey" had rain
- **THEN** the subtitle reads "Copa del Rey · Cuartos de final · 21/10/2026 20:00 · 🌧️"

#### Scenario: Old friendly without stored weather
- **WHEN** a friendly played before this change is opened
- **THEN** the subtitle reads "Amistoso · <date>" with no weather emoji

#### Scenario: World Cup group fixture
- **WHEN** the second-matchday match of World Cup group B is opened
- **THEN** the subtitle reads "Mundial · Grupo B · Jornada 2 · <date>"

#### Scenario: Live league match names its division
- **WHEN** a viewer opens a live matchday-7 match of division "Segunda.2", which has 20 teams (38 matchdays), on a sunny day
- **THEN** the subtitle reads "Segunda.2 · Jornada 7 de 38 · <date> · ☀️", the same in the scheduled, live and finished views

### Requirement: Venue and report flavor
The viewer SHALL show the home team's stadium name and the stadium-entry banner whenever the home team has a stadium. For any competition where weather, attendance and referees are stored, the chronicle MUST open with those lines, both live and after full time.

#### Scenario: Friendly at a club stadium
- **WHEN** a club friendly is played at the home club's stadium "Estadio Norte" in front of 8,412 people
- **THEN** the scoreboard shows "Estadio Norte"
- **AND** the chronicle mentions the crowd of 8,412, the weather and the three referees

#### Scenario: National team without a stadium
- **WHEN** the home side has no stadium
- **THEN** no stadium caption and no stadium-entry banner are shown

### Requirement: Live scorers
The live scoreboard SHALL list each side's goalscorers so far for every competition, with the same penalty and free-kick marks the finished scoreboard uses.

#### Scenario: Goal in a live cup tie
- **WHEN** the home side scores a penalty in minute 12 of a cup tie being broadcast
- **THEN** the live scoreboard lists the scorer with "12'" and the penalty mark under the home side

### Requirement: Zone duel shows share and strength
Each cell of the zone duel SHALL show, for each side, its share of that duel as a whole percentage and, beneath it, that side's raw strength rounded to the nearest whole number. Tapping or hovering a cell MUST show the matchup caption and both raw values to one decimal.

#### Scenario: Central defensive duel
- **WHEN** the home side's central defense plus goalkeeper totals 42.6 and the away side's central attack totals 28.4
- **THEN** the home half shows "60%" in bold above "43", and the away half shows "40%" above "28"
- **AND** the tooltip reads "vs ataque central del rival: 42.6 - 28.4"

#### Scenario: Both sides at zero
- **WHEN** both raw values of a duel are 0
- **THEN** each half shows "50%" above "0"

#### Scenario: Narrow phone
- **WHEN** the viewer is 360 px wide
- **THEN** every percentage and strength figure fits inside its half-cell with no horizontal scroll and no text smaller than 12 px

### Requirement: Goalkeeper counted live
In the zone duel's defense and attack columns, the viewer SHALL add each side's starting goalkeeper's GoalKeeper skill to that side's defensive value, both live and after full time. The same minute of the same match MUST show the same figures in either view.

#### Scenario: Full time reached
- **WHEN** a live match reaches minute 90 and the page switches to the finished view
- **THEN** the zone duel at minute 90 shows the same percentages and strengths it showed live

### Requirement: Zone duel availability
The finished view's Tácticas tab SHALL include the minute-scrubbable zone duel for any match, of any competition, that has stored per-minute instants. A match without stored instants MUST show only the lineups pitch in that tab.

#### Scenario: Cup tie played after this change
- **WHEN** a cup tie simulated after this change is opened after full time
- **THEN** its Tácticas tab shows the zone duel with a slider from minute 0 to the final minute

#### Scenario: Cup tie played before this change
- **WHEN** a cup tie simulated before this change is opened
- **THEN** its Tácticas tab shows only the lineups pitch

### Requirement: Lineup section visibility
The lineup editor SHALL be shown only to a logged-in owner whose own team plays the match, and only in the scheduled view. Once the lineup lock window starts, that owner sees the "lineup locked" notice instead. Nobody else sees either, and neither is shown in the live or finished view.

#### Scenario: Non-participant opens a friendly
- **WHEN** an owner whose team is not playing opens an upcoming friendly
- **THEN** no lineup editor and no locked notice are shown

#### Scenario: Participant during the live match
- **WHEN** a participating owner opens their cup tie after kickoff
- **THEN** the live view is shown with no lineup editor and no locked notice

### Requirement: Cup-only extras
For a cup tie that was level after 90 minutes, the viewer SHALL show a penalty-shootout line under the scoreboard naming the winner. For a bye, the viewer MUST show the bye card with the team that went through, in place of the scoreboard and tabs.

#### Scenario: Tie decided on penalties
- **WHEN** a cup tie ends 1-1 and the away side wins the shootout
- **THEN** the finished view shows "1 - 1" and, beneath it, the line naming the away side as the shootout winner

#### Scenario: Bye
- **WHEN** a user opens a tie with only one team
- **THEN** the bye card names that team, and no scoreboard or tabs are shown

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

## Code map

- Viewer: `frontend/src/features/match/MatchViewer.tsx` (phase switch, live completion, scheduled / live /
  finished views, `lineupSection`, `previewSection` (the opponent preview, see opponent-preview) and
  `belowScoreboard` slots) and the view model
  `features/match/matchViewModel.ts` (`MatchView`, `fromWeekMatch` - its league label is prefixed with the
  division name -, `fromCupTie`, `fromFriendly`, `positionOf`, `weeksInASeason`); each adapter's `liveUrl`
  comes from `liveMatchPath` in `utils/matchSummary.ts`.
- Route adapters: `features/match/MatchDetailsPage.tsx` (league), `features/cup/CupTieDetailsPage.tsx`
  (cup tie, bye card and shootout line), `features/friendlies/FriendlyMatchPage.tsx` (friendly, national
  team and World Cup group match).
- Zone duel: `features/match/ZoneComparisonRows.tsx` (`DuelCell`), `features/match/zoneDuel.ts`
  (share, strength and tooltip formatting), `features/match/MatchMinuteZoneComparison.tsx`,
  `features/match/Lineups.tsx` (`goalkeeperRating`, the "Tácticas" tab).
- Ratings: `features/match/MatchRatingsTab.tsx` (the "Notas" tab), `features/match/MatchMinuteRatings.tsx`
  (scrubber, live and finished), `features/match/TeamRatingsColumn.tsx` (rows, "s.c.", `RatingBreakdown`),
  `utils/liveMatchRatings.ts` (`computePlayerRatings`, `startingLineup`); the formula is in the match-ratings
  spec.
- Mentality: `features/match/MentalityLabel.tsx`, shown next to each side's name in `Lineups.tsx`
  (finished) and the live lineups of `MatchViewer.tsx`, from `Lineup.mentality` of the kickoff
  lineups; labels `match.mentality.*`. The lineup editor's selector is in
  `features/lineup/MatchLineupForm.tsx` (labels `lineup.mentality.*`).
- Subtitle: `utils/matchSubtitle.ts`; labels `match.competition.*` in `i18n/locales/{en,es}.json`.
- Backend payloads: `common.CupTieDetails` / `common.FriendlyMatch` (stadium, flavor, instants, World Cup
  group label); storage described in match-simulation's code map.
