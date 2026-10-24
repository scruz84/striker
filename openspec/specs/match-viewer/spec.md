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
In the live view, the viewer SHALL hide injury and red-card events and per-player match ratings, for every competition, until the replay completes. The finished view shows them all.

#### Scenario: Red card in a live cup tie
- **WHEN** a player is sent off in minute 30 of a cup tie whose replay is at minute 45
- **THEN** the live chronicle doesn't list the red card
- **AND** the finished view lists it once the replay completes

### Requirement: Scoreboard subtitle
The scoreboard subtitle SHALL read `<competition label> · <kickoff date and time> · <weather emoji>`. Any part that is unknown is left out. The labels are:
- League match: "Jornada N de T".
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

## Code map

- Viewer: `frontend/src/features/match/MatchViewer.tsx` (phase switch, live completion, scheduled / live /
  finished views, `lineupSection` and `belowScoreboard` slots) and the view model
  `features/match/matchViewModel.ts` (`MatchView`, `fromWeekMatch`, `fromCupTie`, `fromFriendly`,
  `positionOf`, `weeksInASeason`).
- Route adapters: `features/match/MatchDetailsPage.tsx` (league), `features/cup/CupTieDetailsPage.tsx`
  (cup tie, bye card and shootout line), `features/friendlies/FriendlyMatchPage.tsx` (friendly, national
  team and World Cup group match).
- Zone duel: `features/match/ZoneComparisonRows.tsx` (`DuelCell`), `features/match/zoneDuel.ts`
  (share, strength and tooltip formatting), `features/match/MatchMinuteZoneComparison.tsx`,
  `features/match/Lineups.tsx` (`goalkeeperRating`, the "Tácticas" tab).
- Subtitle: `utils/matchSubtitle.ts`; labels `match.competition.*` in `i18n/locales/{en,es}.json`.
- Backend payloads: `common.CupTieDetails` / `common.FriendlyMatch` (stadium, flavor, instants, World Cup
  group label); storage described in match-simulation's code map.
