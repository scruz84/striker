# Spec Delta

## ADDED Requirements

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
