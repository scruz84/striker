# Spec Delta

## ADDED Requirements

### Requirement: Starting order of a new season
At rollover each league SHALL get a starting order 1..N stored as `classification.seed`. First come the teams relegated into the league, then those that stayed, then those promoted into it. Each group is ordered by final position last season, then points, goal difference and goals scored. A team with no previous season goes after all three groups.

#### Scenario: Promoted teams start at the bottom
- **WHEN** a 10-team division relegates its 9th and 10th, and team P1 (1st below) and P2 (2nd below) come up
- **THEN** the new season opens with last season's 1st to 8th in places 1-8, P1 9th and P2 10th

#### Scenario: Relegated teams start at the top
- **WHEN** teams R9 (9th) and R10 (10th) drop into the lower division
- **THEN** that division opens with R9 1st, R10 2nd, then the teams that stayed in their final order, and the two promoted teams' replacements are not involved

#### Scenario: Promoted from two groups
- **WHEN** two teams that both finished 1st in different lower groups come up with 70 and 64 points
- **THEN** the 70-point team starts second to last and the 64-point team last

### Requirement: Starting order breaks exact ties
Every standings ordering (division table, a team's position, final ranking at rollover, and the page's live revert) SHALL sort by points, goal difference, goals scored, wins, draws, then `seed` ascending with no seed last, then team id. On matchday 0 the table therefore reads in the starting order.

#### Scenario: Matchday 0
- **WHEN** a user opens a division whose new season has no match played
- **THEN** the table lists the teams in the starting order, the same on every reload

#### Scenario: Exact tie later
- **WHEN** two teams have identical points, goals, wins and draws after matchday 10
- **THEN** the one with the lower seed is listed first

### Requirement: Season under way gets a starting order
The migration that adds `seed` SHALL fill it for each league's current season when none of its rows has one, using the previous season's history with the same groups and ordering. It MUST be re-runnable without changing seeds already set.

#### Scenario: Deploy mid-preseason
- **WHEN** the change ships while season 3 is at matchday 0 with every team on 0 points
- **THEN** the division table shows season 2's order, with the teams promoted last season at the bottom

### Requirement: Final tables are kept
At rollover every team's `season_history` row SHALL store matches, points, wins, draws, losses, goals scored and goals received besides its final position, league and promotion or relegation. The migration MUST backfill these counters for every past season from its played league matches.

#### Scenario: Rollover
- **WHEN** a team finishes 4th with 31 points from 18 matches (9 W, 4 D, 5 L, 28-20)
- **THEN** its history row for that season holds position 4, 18, 31, 9, 4, 5, 28 and 20

#### Scenario: Backfill
- **WHEN** the migration runs on a world with two finished seasons
- **THEN** every history row with a league gets counters equal to its team's played league matches in that league and season

### Requirement: League seasons endpoint
`GET /v1/teams/division/:id/seasons` SHALL return `current` (the league country's current season) and `past` (every season with history rows for that league, newest first) to any signed-in user. A non-numeric id SHALL return 400.

#### Scenario: Third season
- **WHEN** a client requests the seasons of a league whose country is in season 3
- **THEN** the response is `{"current": 3, "past": [2, 1]}`

### Requirement: Past season table endpoint
`GET /v1/teams/division/:id/seasons/:season` SHALL return that league's final table for a past season, ordered by final position: per team its id, current name, position, the seven counters, `promoted` and `relegated`, plus the recorded league name and `totalWeeks` = 2 × (teams − 1). A season with no history rows, including the current one, SHALL return 404.

#### Scenario: Season 2 table
- **WHEN** a client requests season 2 of a 10-team league
- **THEN** the response has 10 positions ordered 1 to 10 and `totalWeeks` 18

#### Scenario: Current season
- **WHEN** a client requests the current season
- **THEN** the response is 404

### Requirement: Season picker
The standings SHALL carry a season picker listing the current season and every past one, shown only when there is a past season. Picking a past season SHALL set the `season` query parameter; picking the current season SHALL remove it. A `season` that is not a past season of the league SHALL show the current season.

#### Scenario: Pick season 2
- **WHEN** a user on `/division/7` picks "Temporada 2"
- **THEN** the URL becomes `/division/7?season=2` and the page shows season 2

#### Scenario: First season
- **WHEN** a league's country is in its first season
- **THEN** no season picker is shown

### Requirement: Past season view
A past season SHALL be read-only: its final table from the history endpoint, with zones from the recorded promotions and relegations and a trophy on position 1, and its matchdays with the navigator starting at its last pair (T−1, T) with no chips. Top scorers and recent transfers SHALL be hidden. The subtitle SHALL read "Temporada N · Clasificación final".

#### Scenario: Browse season 2
- **WHEN** a user opens `/division/7?season=2` for a 10-team league
- **THEN** the table shows season 2's final standings, the cards show matchdays 17 and 18 with final scores, and no scorers or transfers panel is shown

#### Scenario: Promoted in a past season
- **WHEN** the season 2 champion was promoted
- **THEN** its row carries the trophy and the promotion mark

### Requirement: History links to the past table
Each season of a team's history tab that has a recorded league SHALL link to `/division/:leagueId?season=N`, the table of the league the team played in that season.

#### Scenario: Open a past table from the team page
- **WHEN** a user taps season 2 in a team's history, a season the team spent in league 12
- **THEN** the page `/division/12?season=2` opens

## MODIFIED Requirements

### Requirement: Matchday matches endpoint
`GET /v1/teams/division/:id/weeks/:week` SHALL return the matches of that week in the league's current season, or in the season given by an optional `season` query parameter, in the same match shape as `previousWeekMatches`, to any signed-in user (it sits in the same authenticated route group as the division details and top-scorers endpoints, with no per-team ownership check). A non-numeric league id, a week below 1 or a non-numeric or below-1 `season` SHALL return 400. A week with no matches SHALL return 200 with an empty array. The response MAY be cached for at most 60 seconds.

#### Scenario: Valid week
- **WHEN** a client requests `/v1/teams/division/7/weeks/3`
- **THEN** the response is 200 with matchday 3's matches of league 7's current season

#### Scenario: Past season week
- **WHEN** a client requests `/v1/teams/division/7/weeks/3?season=2`
- **THEN** the response is 200 with matchday 3's matches of league 7 in season 2

#### Scenario: Invalid week
- **WHEN** a client requests `/v1/teams/division/7/weeks/0`, `/weeks/abc` or `/weeks/3?season=0`
- **THEN** the response is 400

#### Scenario: Week beyond the season
- **WHEN** a client requests week 31 of a league with 30 matchdays
- **THEN** the response is 200 with `[]`

### Requirement: Mobile tabs
On viewports narrower than the `md` breakpoint (900 px), the page SHALL show four full-width tabs in this order: "Tabla" (standings), "Jornadas" (matchday navigator and cards), "Goleadores" (top scorers only) and "Traspasos" (recent transfers only). In a past season only "Tabla" and "Jornadas" SHALL be shown. "Tabla" SHALL be selected on open. All labels MUST fit at 360 px wide without clipping or horizontal scroll, and each tab MUST be at least 44 px tall.

#### Scenario: Transfers on a phone
- **WHEN** a user on a 360 px phone opens the "Goleadores" tab
- **THEN** only the top-scorers panel is shown, and the recent-transfers panel is shown only under "Traspasos"

#### Scenario: English labels
- **WHEN** the language is English
- **THEN** the tabs read "Table", "Fixtures", "Scorers" and "Transfers"

#### Scenario: Past season on a phone
- **WHEN** a user on a 360 px phone opens `/division/7?season=2`
- **THEN** only the "Tabla" and "Jornadas" tabs are shown
