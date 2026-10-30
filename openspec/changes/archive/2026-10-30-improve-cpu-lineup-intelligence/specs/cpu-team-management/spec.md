# Spec Delta

## MODIFIED Requirements

### Requirement: Lineup and formation
Every CPU team SHALL re-pick its best XI from the current squad whenever it acts and again at the kickoff of each of its matches, choosing among the six formations ranked by best-XI strength: a Director takes the strongest (rank 0), a Manager the second, a Caretaker the third or fourth at random. A Manager or Director MUST pick only from players available for that match; a Caretaker picks from the whole squad.

#### Scenario: New signing
- **WHEN** a CPU team signs a better striker
- **THEN** its next lineup refresh can include him

#### Scenario: Injured starter, Manager
- **WHEN** a Manager's best striker is injured at kickoff of a league match
- **THEN** its formation and XI are chosen from the squad without him, so a different formation can win

#### Scenario: Injured starter, Caretaker
- **WHEN** a Caretaker's best striker is injured at kickoff
- **THEN** its XI still names him and the match-day backfill replaces him with the best available player for that position

#### Scenario: Suspended player in a friendly
- **WHEN** a Manager's player serving a suspension is picked for a friendly
- **THEN** he may play, because suspensions only bar league and cup matches

## ADDED Requirements

### Requirement: CPU kickoff lineup obeys a human's lineup rules
The XI, formation and scheduled substitutions a CPU team picks at kickoff MUST pass the same validation a human's saved lineup and substitutions pass, except that a Caretaker's XI may name unavailable players. A pick that fails validation MUST be dropped in favour of the team's stored lineup for that match.

#### Scenario: Valid pick
- **WHEN** a Director's kickoff XI and 3 substitutions are validated
- **THEN** they match the formation's position counts, name only squad players, never repeat a player, and every substitution takes off a starter for a fit bench player at a minute from 1 to 90

#### Scenario: Invalid pick
- **WHEN** a CPU kickoff pick fails validation
- **THEN** the match is played with the team's stored lineup and the failure is logged

### Requirement: CPU planned substitutions
At kickoff of a league match or cup tie a CPU team SHALL schedule substitutions by its AI level, scoring each starter/bench pair at a slot by fatigue gain: the bench player's average effective value over the remaining minutes minus the starter's, times those minutes (see scenarios). A Caretaker schedules at most 1, a Manager at most 2, a Director at most 3. No player is used in two pairs.

#### Scenario: Gain formula
- **WHEN** a starter worth V with Stamina Ss would be replaced at minute m by a bench player worth W with Stamina Sb
- **THEN** the starter's average factor is 1 − 0.30 × ((m + 90) / 180) × (99 − Ss) / 99, the bench player's is 1 − 0.30 × ((90 − m) / 180) × (99 − Sb) / 99
- **AND** gain = (90 − m) × (W × bench factor − V × starter factor), with W and V each player's value at the starter's slot

#### Scenario: Caretaker
- **WHEN** a Caretaker's lowest-Stamina outfield starter has Stamina below 50 and the best bench player for his slot is worth at least 80% of him
- **THEN** it schedules that one change at minute 70; otherwise it schedules none

#### Scenario: Manager
- **WHEN** a Manager evaluates every outfield pair at minute 70
- **THEN** it schedules the top 2 pairs by gain whose gain is at least 5% of (90 − 70) × V, so one substitution always stays free for injuries

#### Scenario: Director
- **WHEN** a Director evaluates every pair at minutes 55, 60, …, 85
- **THEN** each pair takes its best minute, and the top 3 pairs by gain whose gain is at least 2% of (90 − m) × V are scheduled
- **AND** the third of them is moved to minute 75 if it was due earlier, so an early injury can still be covered

#### Scenario: Nothing worth changing
- **WHEN** no pair reaches its level's threshold
- **THEN** the team schedules no substitutions

### Requirement: CPU friendly rotation
At kickoff of a friendly a CPU team SHALL start an XI and schedule 3 substitutions at minute 60 so its reserves play, by AI level. Regulars are the XI it would field in a league match that day. Bench players for the substitutions are picked by highest value at their best open slot, and each replaces the starter in that slot. Fewer than 3 are scheduled if the squad has no eligible reserves left.

#### Scenario: Caretaker friendly
- **WHEN** a Caretaker plays a friendly
- **THEN** it starts its regulars and at minute 60 brings on its 3 best bench players

#### Scenario: Manager friendly
- **WHEN** a Manager plays a friendly
- **THEN** its 5 outfield regulars with the highest value at their slot do not start or come on, it starts the best XI from the rest in its league formation, and its 3 substitutions use only players who did not start

#### Scenario: Director friendly
- **WHEN** a Director plays a friendly
- **THEN** it starts the best XI from players who are not regulars, including a backup goalkeeper if it has one
- **AND** a position no reserve can fill takes the lowest-valued regular for it, and key players start only if needed to field 11
- **AND** its 3 substitutions at minute 60 prefer players aged 21 or under, then the highest value

#### Scenario: Thin squad
- **WHEN** a CPU team has only 13 fit players for a friendly
- **THEN** it fields 11 and schedules at most 2 substitutions
