# league-page Specification

## Purpose
The league (division) details page at `/division/:id`. It shows a league's current season: the
standings table, matchday fixtures with navigation across every matchday, top scorers and recent
transfers, laid out for desktop and for phones.

## Requirements

### Requirement: Default matchday pair
The fixtures section SHALL show two matchdays, L and L+1. By default L = clamp(W, 1, T−1), where W is the current week and T = 2 × (teams − 1) is the season's matchday count. W is the matches played by the top-ranked team, after any still-live result is reverted. The card for matchday W carries an "Anterior" chip and the card for matchday W+1 a "Próxima" chip, whichever pair is shown.

#### Scenario: Mid-season
- **WHEN** a 16-team league (T = 30) has W = 12 and the page opens
- **THEN** the cards show "Jornada 12" with the "Anterior" chip and "Jornada 13" with the "Próxima" chip

#### Scenario: Before the first matchday
- **WHEN** W = 0
- **THEN** the cards show "Jornada 1" with the "Próxima" chip and "Jornada 2" with no chip, and no empty card is shown

#### Scenario: Season finished
- **WHEN** W = T = 30
- **THEN** the cards show "Jornada 29" with no chip and "Jornada 30" with the "Anterior" chip

### Requirement: Matchday navigation
A navigator above the two cards SHALL let the user move the pair. "‹" sets L to L−1, "›" sets L to L+1, and a week picker listing matchdays 1 to T sets L to the chosen week, clamped to 1..T−1. "‹" is disabled at L = 1 and "›" at L = T−1. A "back to current" control appears only when L differs from the default and restores it.

#### Scenario: Step back
- **WHEN** the pair shows matchdays 12 and 13 and the user taps "‹"
- **THEN** the pair shows matchdays 11 and 12

#### Scenario: Jump to the last matchday
- **WHEN** T = 30 and the user picks matchday 30 in the week picker
- **THEN** the pair shows matchdays 29 and 30 and "›" is disabled

#### Scenario: Back to current
- **WHEN** W = 12, the user has moved to matchdays 3 and 4, and taps "back to current"
- **THEN** the pair shows matchdays 12 and 13 and the "back to current" control is hidden

### Requirement: Browsed matchday in the URL
The page SHALL keep a browsed pair in the `week` query parameter, holding L. Opening `/division/:id?week=N` SHALL show the pair (clamp(N, 1, T−1), +1). A missing or non-numeric `week` SHALL show the default pair. Returning to the default pair SHALL remove the parameter.

#### Scenario: Shared link
- **WHEN** a user opens `/division/7?week=5` in a 16-team league
- **THEN** the cards show matchdays 5 and 6

#### Scenario: Out-of-range week
- **WHEN** a user opens `/division/7?week=99` with T = 30
- **THEN** the cards show matchdays 29 and 30

### Requirement: Matchday fixtures content
Each card SHALL list that matchday's matches grouped by kickoff date, home above away, linking to `/match/:id` and highlighting the user's own match. A matchday with no matches SHALL show "Sin partidos". A stored score SHALL show only for a match that is played, past kickoff and outside its live-broadcast window. A played match still in that window SHALL show a LIVE chip instead, or its broadcast score as set out in "Live score in a matchday card".

#### Scenario: Earlier matchday
- **WHEN** a user browses to matchday 3, played two weeks ago
- **THEN** every match shows its final score

#### Scenario: Match still broadcasting
- **WHEN** a matchday's match kicked off 40 minutes ago and is already simulated, and the division live scores are unavailable
- **THEN** that match shows the LIVE chip and no score, on every matchday the user can browse to

### Requirement: Matchday matches endpoint
`GET /v1/teams/division/:id/weeks/:week` SHALL return the matches of that week in the league's current season, in the same match shape as `previousWeekMatches`, to any signed-in user (it sits in the same authenticated route group as the division details and top-scorers endpoints, with no per-team ownership check). A non-numeric league id or a week below 1 SHALL return 400. A week with no matches SHALL return 200 with an empty array. The response MAY be cached for at most 60 seconds.

#### Scenario: Valid week
- **WHEN** a client requests `/v1/teams/division/7/weeks/3`
- **THEN** the response is 200 with matchday 3's matches of league 7's current season

#### Scenario: Invalid week
- **WHEN** a client requests `/v1/teams/division/7/weeks/0` or `/weeks/abc`
- **THEN** the response is 400

#### Scenario: Week beyond the season
- **WHEN** a client requests week 31 of a league with 30 matchdays
- **THEN** the response is 200 with `[]`

### Requirement: Default pair needs no extra request
The page SHALL NOT request the matchday endpoint for a matchday whose matches the division details response already carries (its previous and next matchday). Those matches SHALL come from that response.

#### Scenario: Page load mid-season
- **WHEN** a user opens a league page with W = 12, no match inside its live-broadcast window, and no `week` parameter
- **THEN** only the division details, top scorers and recent transfers endpoints are requested, and no `/weeks/` or `/live` request is made

#### Scenario: Page load while matchday 12 is broadcasting
- **WHEN** matchday 12 was simulated but is still inside its live-broadcast window, so W reads as 11
- **THEN** the cards show matchdays 11 and 12, matchday 12 comes from the division details response, only matchday 11 is requested from `/weeks/11`, and the division live scores are requested from `/live`

### Requirement: Mobile tabs
On viewports narrower than the `md` breakpoint (900 px), the page SHALL show four full-width tabs in this order: "Tabla" (standings), "Jornadas" (matchday navigator and cards), "Goleadores" (top scorers only) and "Traspasos" (recent transfers only). "Tabla" SHALL be selected on open. All four labels MUST fit at 360 px wide without clipping or horizontal scroll, and each tab MUST be at least 44 px tall.

#### Scenario: Transfers on a phone
- **WHEN** a user on a 360 px phone opens the "Goleadores" tab
- **THEN** only the top-scorers panel is shown, and the recent-transfers panel is shown only under "Traspasos"

#### Scenario: English labels
- **WHEN** the language is English
- **THEN** the tabs read "Table", "Fixtures", "Scorers" and "Transfers"

### Requirement: Desktop layout
From the `md` breakpoint up, the page SHALL show the standings (8 of 12 columns) beside a stack of top scorers and recent transfers (4 of 12 columns), with the matchday navigator and the two matchday cards side by side below, and no tabs.

#### Scenario: Desktop browse
- **WHEN** a user on a 1280 px screen taps "›" in the navigator
- **THEN** only the two matchday cards change, and the standings, top scorers and recent transfers stay as they were

### Requirement: Recent transfers content
The recent-transfers panel of a division (division page and dashboard) SHALL list the 10 most recent player moves in which the buying or selling club belongs to that division, newest first: bought listings, accepted offers, exercised buy options, permanent moves recorded before structured history, free-agent signings that paid a premium to a previous club, and loans. Releases and premium-free free-agent signings MUST NOT appear.

#### Scenario: Free loan
- **WHEN** a club in division "2A" takes a player on a 1-season loan with no fee from a club in "1A"
- **THEN** the panels of both "2A" and "1A" show the move with a "Loan" chip, "<borrower> on loan from <lender>" and "Free" as the amount

#### Scenario: Loan with a fee
- **WHEN** the loan carried a loan fee of 150,000
- **THEN** the row shows the "Loan" chip and 150,000

#### Scenario: Permanent transfer
- **WHEN** a club in the division buys a listed player for 2,000,000
- **THEN** the row has no chip, reads "<buyer> from <seller>" and shows 2,000,000

#### Scenario: Buy option
- **WHEN** a borrower exercises a 900,000 buy option
- **THEN** a row for 900,000 appears with no "Loan" chip, newer than the original loan row

#### Scenario: Release
- **WHEN** a club in the division releases a player
- **THEN** nothing is added to the panel

### Requirement: Recent transfers endpoint
`GET /v1/team/division/:id/recent-transfers` SHALL read `player_transfers` and return, per row, the player, `amount` (the fee, ≥ 0), `kind` (a `PLAYER_TRANSFER_*` value), buyer and seller team ids and names, and `occurredAt`.

#### Scenario: Loan row
- **WHEN** the latest move in the division is a free loan
- **THEN** the first element has `kind` 3 and `amount` 0, with the borrower as buyer and the lender as seller

### Requirement: Live score in a matchday card
On the current season, a played match still inside its live-broadcast window SHALL show the score and state from the division live scores in place of the bare LIVE chip: "63'" (pulsing, error colour) while `live`, "Half-time" while `halftime`, and "FT" (neutral) once `finished`. A `pending` entry, a match with no entry, or a failed request SHALL keep the LIVE chip and no score.

#### Scenario: Match in play
- **WHEN** a matchday card shows a match at minute 63 of its broadcast at 1-0, whose stored result is 3-0
- **THEN** the row shows 1 and 0 beside the teams and a "63'" chip, never 3-0

#### Scenario: Broadcast over, window still open
- **WHEN** a match's broadcast reported full time at 2-2 and it kicked off 1 hour 50 minutes ago
- **THEN** the row shows 2 and 2 with an "FT" chip

#### Scenario: Simulated ahead of kickoff
- **WHEN** a match was simulated and its kickoff is still 5 minutes away
- **THEN** the row keeps the LIVE chip and shows no score, as it did before this change

### Requirement: Live scores only while something is live
The league page SHALL request the division live scores only on the current season, and only while the previous matchday carried by the division details response has a played match inside its live-broadcast window. It SHALL follow the division live scores polling rules.

#### Scenario: Quiet day
- **WHEN** no match of the league is inside its live-broadcast window
- **THEN** the page makes no live scores request

#### Scenario: Browsing an old matchday mid-broadcast
- **WHEN** matchday 12 is broadcasting and the user browses to matchdays 3 and 4
- **THEN** the page keeps polling, because the live table still uses the scores, and the cards of matchdays 3 and 4 show their stored final scores

### Requirement: Live table switch
On the current season, while at least one division live scores entry is `live`, `halftime` or `finished`, the standings SHALL show a "Live" switch above the table, on by default. With it on, the table SHALL be the live table. With it off, the table SHALL be today's table. The switch is not remembered between visits.

#### Scenario: Matchday under way
- **WHEN** a user opens the league page 30 minutes after matchday 12 kicked off
- **THEN** the "Live" switch is shown, on, and the table is the live table

#### Scenario: Switch off
- **WHEN** the user turns the switch off
- **THEN** the table shows the standings with every still-live result reverted, as before this change

#### Scenario: Nothing live
- **WHEN** no entry is `live`, `halftime` or `finished`, or the live scores request failed
- **THEN** no switch is shown and the table is the standings with still-live results reverted

#### Scenario: Past season
- **WHEN** a user views season 2 of the league
- **THEN** no switch is shown

### Requirement: Live table computation
The live table SHALL start from the standings with every still-live result reverted. For each reverted match whose entry is `live`, `halftime` or `finished`, it SHALL add one played match with the broadcast score: 3 points for a win, 1 for a draw, 0 for a loss, plus goals for and against. It SHALL then sort with the standings' own ordering and renumber.

#### Scenario: Live win moves a team up
- **WHEN** before the matchday A is 5th with 20 points and B 4th with 21, and A leads 1-0 at minute 70 while B draws 0-0
- **THEN** the live table has A on 23 points in 4th and B on 22 points in 5th

#### Scenario: Pending match adds nothing
- **WHEN** a match was simulated early and its entry is still `pending`
- **THEN** both teams' rows in the live table equal their pre-matchday rows

### Requirement: Live table markers
In the live table, a row whose position differs from its pre-matchday position SHALL show "▲N" (success colour) or "▼N" (error colour), N being the places gained or lost. A row whose team's entry is `live` or `halftime` SHALL show a pulsing live dot. The promotion and relegation zones SHALL follow live positions.

#### Scenario: Position markers
- **WHEN** A goes from 5th to 4th and B from 4th to 5th in the live table
- **THEN** A's row shows "▲1", B's row "▼1", and both rows show the live dot while their matches are in play

#### Scenario: Into the relegation zone
- **WHEN** a league has 3 relegation slots and a team drops from 15th to 16th of 18 in the live table
- **THEN** that row is drawn in the relegation zone with "▼1"

#### Scenario: Live table on a phone
- **WHEN** a user on a 360 px phone opens "Tabla" during a matchday
- **THEN** the switch and the markers fit without clipping or horizontal page scroll

### Requirement: Standings zones
On the current season, the standings table SHALL colour each row by the zone its position has in the league pyramid. Direct promotion uses the success colour, the promotion and relegation playoff the warning colour, and direct relegation the error colour. The table never marks more teams as promoted than actually go up. A legend SHALL list only the zones the league has. The live table SHALL apply the same zones to live positions.

#### Scenario: Tier-2 group of 10
- **WHEN** the user opens a tier-2 group of a 5-tier country
- **THEN** 1st is in the promotion colour, 2nd, 7th and 8th in the playoff colour, and 9th and 10th in the relegation colour

#### Scenario: Top division
- **WHEN** the user opens tier 1
- **THEN** no row is in the promotion colour, 7th and 8th are in the playoff colour, and the legend shows "Playoff" and "Relegation" only

#### Scenario: Bottom tier
- **WHEN** the user opens a tier-5 group
- **THEN** 1st is in the promotion colour, 2nd in the playoff colour, and no row is in the relegation colour

### Requirement: Playoff section
While the country's current season has playoff ties, the league page SHALL show a "Playoff" card listing the ties where at least one side belongs to this league. Each row shows both teams, the kickoff, and the score once its broadcast is over (with a penalties mark if needed). Each row links to the tie's match page. The card is absent before the ties exist and on past seasons.

#### Scenario: Ties created
- **WHEN** tier 1's league ends and its 7th and 8th have ties against two tier-2 runners-up
- **THEN** tier 1's page and both tier-2 groups' pages show a "Playoff" card listing the ties that involve them

#### Scenario: Before the league ends
- **WHEN** league matches remain
- **THEN** no "Playoff" card is shown

### Requirement: Playoff marks on past seasons
A past season's table SHALL mark teams that went up or down through the playoff with "(playoff)" next to the promotion or relegation mark. A team that won its playoff and stayed SHALL show no mark.

#### Scenario: Relegated through the playoff
- **WHEN** a team finished 8th in tier 1 and lost its playoff
- **THEN** its row in that past season shows the relegation colour and "(playoff)"

## Code map

- Backend: `backend/common/leaguetransfers.go` (`GetRecentLeagueTransfers`,
  `getRecentLeagueTransfersStmtSql`, `LeagueTransfer`), read from `player_transfers`
  (`backend/common/playertransfers.go`); `backend/resources/sql/init93_player_transfers_recent_index.sql`;
  `backend/handlers/statsHandlers.go` (`RecentLeagueTransfersHandler`).
- Frontend: `frontend/src/features/leagues/LeaguesPage.tsx`,
  `frontend/src/features/leagues/RecentLeagueTransfers.tsx` (also on the dashboard),
  `frontend/src/api/types.ts` (`LeagueTransfer`); strings under `recentTransfers.*` in
  `i18n/locales/{en,es}.json`.
- Live scores and live table: `frontend/src/features/division/DivisionDetailsPage.tsx`
  (`hideStillLiveResults`, the "Live" switch, `FixturesCard` broadcast scores, `ClassificationTable`'s
  `live` markers), `features/division/liveTable.ts` (`applyLiveScores`, `compareClassification`),
  `api/hooks.ts` (`useDivisionLiveScores`), `utils/liveScoreLabel.ts`; backend
  `backend/handlers/statsHandlers.go` (`LeagueLiveScoresHandler`) and `backend/common/livematches.go`
  (`GetLeagueLiveScores`) behind `GET /v1/teams/division/:id/live` (see the `division-live-scores`
  spec). Strings: `division.liveTable`, `division.positionUp`, `division.positionDown`,
  `liveScores.fullTime` in `i18n/locales/{en,es}.json`.
- Zones and playoff card: `backend/common/pyramid.go` (`PyramidZones`, `ZoneOf`,
  `GetLeagueZones`) filling `DivisionDetails.Zones`, and `backend/common/playoffs.go`
  (`GetLeaguePlayoffTies`, which hides a tie's result while it is broadcasting) filling
  `DivisionDetails.Playoff` (`common.GetDivisionDetails`); frontend `features/division/zones.ts`
  (`zoneFor`, `zonesLegend`), `features/division/PlayoffCard.tsx` and `ClassificationTable`'s
  legend and `markOf` ("(playoff)" from `LeagueSeasonPosition.viaPlayoff`) in
  `DivisionDetailsPage.tsx`. Strings: `division.legend.playoff`, `division.legend.candidate`,
  `division.playoff.*`, `division.viaPlayoff`.
- Tests: `backend/common/leaguetransfers_integration_test.go`,
  `frontend/src/features/leagues/RecentLeagueTransfers.test.tsx`,
  `frontend/src/features/division/liveTable.test.ts`,
  `frontend/src/features/division/DivisionDetailsPage.liveTable.test.tsx`,
  `frontend/src/features/division/DivisionDetailsPage.zones.test.tsx`,
  `frontend/src/features/division/PlayoffCard.test.tsx`.
