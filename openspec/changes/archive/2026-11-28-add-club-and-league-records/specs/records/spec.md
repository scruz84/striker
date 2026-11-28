# Spec Delta

## ADDED Requirements

### Requirement: Club match records
A club SHALL show its biggest win and defeat (widest margin, then most goals scored or conceded, then earliest), its highest-scoring match and its record home attendance (then earliest), over its league matches and cup ties. A record nobody holds is left out.

#### Scenario: Margins tie
- **WHEN** a club won 3-0, 4-1 and, in the cup, 5-2
- **THEN** its biggest win is the 5-2

### Requirement: Club streaks
A club SHALL show its longest runs of league wins, of league matches without defeat and of league defeats, in kickoff order, with their first and last dates. The first longest run wins a tie, and a run reaching the club's latest league match is marked ongoing.

#### Scenario: W W D L L
- **WHEN** a club's league results are win, win, draw, defeat, defeat
- **THEN** its streaks read 2 wins, 3 unbeaten and 2 defeats, the defeats ongoing

### Requirement: Club season and transfer records
A club SHALL show its best finish (highest tier, then best position, then earliest season), its most points in a season, its record signing and its record sale (biggest fee of a paying transfer, retired players included), and its all-time top 5 scorers and top 5 by appearances in league, cup, playoff and regional cup matches.

#### Scenario: Title beats runner-up
- **WHEN** a club finished 2nd with 51 points in season 1 and 1st with 48 in season 2, in the same tier
- **THEN** its best finish is season 2 and its most points season 1

### Requirement: League records
A league's history page SHALL list its champions newest first with their points, the clubs with most titles (then the latest title, then team id), its Golden Boots, the most points in a season, its biggest win, its highest-scoring match and its all-time top 5 scorers in its league matches.

#### Scenario: Two champions with one title each
- **WHEN** club A won season 1 and club B season 2
- **THEN** most titles lists B before A

### Requirement: Records never show early and are cached
Records SHALL leave out every fixture still inside its live-broadcast window, and MAY be served from a cache up to 10 minutes old.

#### Scenario: Record set on air
- **WHEN** a club wins 9-0 in a match that kicked off an hour ago
- **THEN** its biggest win is still the previous record
