# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Matchday fixtures content
Each card SHALL list that matchday's matches grouped by kickoff date, home above away, linking to `/match/:id` and highlighting the user's own match. A matchday with no matches SHALL show "Sin partidos". A stored score SHALL show only for a match that is played, past kickoff and outside its live-broadcast window. A played match still in that window SHALL show a LIVE chip instead, or its broadcast score as set out in "Live score in a matchday card".

#### Scenario: Earlier matchday
- **WHEN** a user browses to matchday 3, played two weeks ago
- **THEN** every match shows its final score

#### Scenario: Match still broadcasting
- **WHEN** a matchday's match kicked off 40 minutes ago and is already simulated, and the division live scores are unavailable
- **THEN** that match shows the LIVE chip and no score, on every matchday the user can browse to

### Requirement: Default pair needs no extra request
The page SHALL NOT request the matchday endpoint for a matchday whose matches the division details response already carries (its previous and next matchday). Those matches SHALL come from that response.

#### Scenario: Page load mid-season
- **WHEN** a user opens a league page with W = 12, no match inside its live-broadcast window, and no `week` parameter
- **THEN** only the division details, top scorers and recent transfers endpoints are requested, and no `/weeks/` or `/live` request is made

#### Scenario: Page load while matchday 12 is broadcasting
- **WHEN** matchday 12 was simulated but is still inside its live-broadcast window, so W reads as 11
- **THEN** the cards show matchdays 11 and 12, matchday 12 comes from the division details response, only matchday 11 is requested from `/weeks/11`, and the division live scores are requested from `/live`
