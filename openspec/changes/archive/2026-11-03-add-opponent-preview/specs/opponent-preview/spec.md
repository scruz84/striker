# Spec Delta

## Purpose

Shows a club's owner, before one of his matches, how the opponent has been playing and where his
own saved lineup is projected to win or lose the duel on the pitch, using only public data.

## ADDED Requirements

### Requirement: Preview contents
The preview of an upcoming fixture SHALL show the opponent's last 5 league and cup results (most recent first), its current league position, the formation and XI it kicked off with in its latest league or cup match, and its 3 best average match ratings over its last 5 league and cup matches among players with at least 3 rated appearances there.

#### Scenario: Established opponent
- **WHEN** the opponent played 8 league matches, the latest in a 4-3-3
- **THEN** the preview shows its last 5 results, its position, a 4-3-3 with that match's starting XI and its top 3 rated players

#### Scenario: No history
- **WHEN** the opponent has not played a league or cup match yet
- **THEN** the preview shows its position (if any) and states that there is no recent data, with no likely XI and no projection

### Requirement: Projected zone duel
The preview SHALL rate the owner's currently saved lineup for that fixture against the opponent's likely XI with the same zone ratings the simulation uses at minute 1 (no fatigue), including any home advantage and mentality the simulation would apply, and show each zone's share and strength as the viewer's zone duel does.

#### Scenario: Stronger midfield
- **WHEN** the owner's saved midfield rates 240 against the opponent's 200
- **THEN** the midfield row shows a 55% / 45% share in the owner's favour

#### Scenario: Lineup changed
- **WHEN** the owner saves a lineup with an extra striker
- **THEN** the projection shown after the save uses the new lineup

### Requirement: Preview privacy
Only an owner whose club plays the fixture SHALL get its preview, and only before kickoff. The opponent's data MUST come only from public sources (results, kickoff lineups and ratings of matches whose replay has finished, standings); its saved lineup, substitutions, orders, mentality and its players' availability MUST NOT be used.

#### Scenario: Non-participant
- **WHEN** a manager whose club is not in the fixture requests its preview
- **THEN** the request is refused

#### Scenario: Opponent changed its lineup
- **WHEN** the opponent saved a completely new XI for this fixture
- **THEN** the preview still shows the XI it kicked off with in its latest match

#### Scenario: Latest match still being replayed
- **WHEN** the opponent's latest match is still in its live replay
- **THEN** the preview uses the match before it

### Requirement: Preview placement
The lineup editor SHALL show, above the pitch and collapsed on phones, the preview of the club's next fixture (league, cup or friendly) whose lineup lock (20 minutes before kickoff) has not started. The scheduled view of a match SHALL show it to a participating owner below his lineup editor, and MUST NOT show it once that match's lock has started.

#### Scenario: Lineup editor
- **WHEN** an owner opens his lineup editor and his next fixture is a cup tie
- **THEN** the preview of that cup tie is shown above the pitch

#### Scenario: Lineup locked
- **WHEN** a participating owner opens a match whose lineup lock has started
- **THEN** the match page shows no preview, only the "lineup locked" notice

#### Scenario: Next fixture already locked
- **WHEN** the club's next league match kicks off in 10 minutes and a friendly follows tomorrow
- **THEN** the lineup editor shows the preview of tomorrow's friendly
