# Spec Delta

## Purpose

The league (division) details page at `/division/:id`. It shows a league's current season: the
standings table, matchday fixtures with navigation across every matchday, top scorers and recent
transfers, laid out for desktop and for phones.

## ADDED Requirements

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
Each card SHALL list that matchday's matches grouped by kickoff date, home above away, linking to `/match/:id` and highlighting the user's own match. A matchday with no matches SHALL show "Sin partidos". A score SHALL show only for a match that is played, past kickoff and outside its live-broadcast window. A played match still in that window SHALL show a LIVE chip instead.

#### Scenario: Earlier matchday
- **WHEN** a user browses to matchday 3, played two weeks ago
- **THEN** every match shows its final score

#### Scenario: Match still broadcasting
- **WHEN** a matchday's match kicked off 40 minutes ago and is already simulated
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
- **THEN** only the division details, top scorers and recent transfers endpoints are requested, and no `/weeks/` request is made

#### Scenario: Page load while matchday 12 is broadcasting
- **WHEN** matchday 12 was simulated but is still inside its live-broadcast window, so W reads as 11
- **THEN** the cards show matchdays 11 and 12, matchday 12 comes from the division details response, and only matchday 11 is requested from `/weeks/11`

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
