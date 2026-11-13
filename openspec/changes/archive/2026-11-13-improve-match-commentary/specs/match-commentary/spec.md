# Spec Delta

## Purpose

How the match chronicle turns events into commentary text: phrasing pools, lines that react to
the match situation, extra colour sentences, the no-repeat rule, the opening, half-time and
full-time lines, and the spoiler rules that apply to commentary text.

## ADDED Requirements

### Requirement: Phrasing pool sizes
Every commentary pool SHALL exist in Spanish and English with at least the minimum number of lines listed in the scenarios. Each line MUST use only the placeholders its pool allows (`{{player}}`, `{{team}}`, `{{opponent}}`, `{{keeper}}` for goals and misses, `{{inPlayer}}`/`{{outPlayer}}` for substitutions, `{{home}}`, `{{away}}`, `{{score}}`, `{{goals}}` where stated).

#### Scenario: Event pools
- **WHEN** the locale files are checked
- **THEN** each language has at least 12 lines for an open-play goal and for an open-play miss; 8 for a yellow card and for a substitution; 6 for a penalty goal, a penalty miss, a free-kick goal, a free-kick miss, a red card and an injury; 4 for a second yellow and for each mentality; 3 for an abandoned match; 2 for a default result

#### Scenario: Situation and colour pools
- **WHEN** the locale files are checked
- **THEN** each language has at least 4 lines for each goal situation and for a late miss, 10 colour lines for misses, 8 colour lines for yellow cards, 6 kickoff lines, and 3 lines for each half-time and full-time case

#### Scenario: No keeper stored
- **WHEN** an open-play miss line names `{{keeper}}` and the event has no keeper (a match from before the keeper was stored)
- **THEN** the line reads "the goalkeeper" / "el portero" in its place

#### Scenario: Unknown placeholder
- **WHEN** a line uses a placeholder its pool does not allow
- **THEN** the locale check fails

### Requirement: Goal situation sentence
Every goal SHALL be narrated as its own line followed by one situation sentence, from the first situation in this order that applies: hat-trick, late winner, late equaliser, comeback, brace, opener, equaliser, go-ahead, rout, extended lead, pulled one back, consolation. A goal is late at minute 85 or later, second-half added time included. Only events up to and including that goal count.

#### Scenario: Late equaliser
- **WHEN** the away side makes it 2-2 in minute 88
- **THEN** the goal line is followed by a late-equaliser sentence, not a plain equaliser one

#### Scenario: Hat-trick wins over everything
- **WHEN** a player scores their third goal of the match to make it 3-3 in minute 90+2
- **THEN** the sentence is a hat-trick one, and `{{goals}}` reads 3

#### Scenario: Comeback
- **WHEN** the home side was 0-2 down and its goal makes it 3-2 in minute 70
- **THEN** the sentence is a comeback one

#### Scenario: Score-based cases
- **WHEN** a goal is the first of the match, or leaves the scorers 4 or more ahead, 2-3 ahead, 1 behind, or 2 or more behind
- **THEN** the sentence is opener, rout, extended lead, pulled one back or consolation, in that order

#### Scenario: Old match
- **WHEN** a match played before this change is opened
- **THEN** its goals get situation sentences too, worked out from its stored events

### Requirement: Second yellow line
A red card SHALL be narrated from the second-yellow pool when the same player already had a yellow card earlier in the match, and from the red-card pool otherwise.

#### Scenario: Booked twice
- **WHEN** a player booked in minute 30 is sent off in minute 61
- **THEN** the minute-61 line comes from the second-yellow pool

#### Scenario: Straight red
- **WHEN** a player with no earlier booking is sent off
- **THEN** the line comes from the red-card pool

### Requirement: Late miss sentence
A missed chance at minute 85 or later SHALL be followed by a late-miss sentence when the side that missed was level or one goal behind at that moment. Other misses MUST NOT get it.

#### Scenario: Missed equaliser
- **WHEN** the home side, 1-2 down, misses a chance in minute 89
- **THEN** the miss line is followed by a late-miss sentence

#### Scenario: Comfortable lead
- **WHEN** a side leading 3-0 misses in minute 88
- **THEN** no late-miss sentence is added

### Requirement: Colour sentences
A missed chance with no late-miss sentence SHALL get one colour sentence 30% of the time, and a yellow card 25% of the time. The choice MUST be a fixed function of the match and the event, so the same event gets the same answer every time.

#### Scenario: Same match opened twice
- **WHEN** a viewer opens the same finished match twice
- **THEN** exactly the same misses and bookings carry a colour sentence both times

#### Scenario: Share over many events
- **WHEN** the rule is applied to 10,000 different misses
- **THEN** between 27% and 33% of them get a colour sentence

### Requirement: No repeated lines within a match
Within one match, each pool SHALL use every one of its lines once before it uses any line again. Which line an event gets MUST depend only on the match (its two teams and kickoff time) and the events before it in match order, including events that are hidden or collapsed.

#### Scenario: Twelve goals
- **WHEN** a match has 12 open-play goals and the open-play goal pool has 12 lines
- **THEN** all 12 goals read differently

#### Scenario: Rematch
- **WHEN** the same two teams meet again on another date with the same events
- **THEN** the chronicle does not read line for line like the first match

#### Scenario: Collapsed misses
- **WHEN** a viewer switches between "key events only" and the full commentary
- **THEN** every line that is shown in both modes reads the same in both

### Requirement: Commentary stays the same from live to finished
A line shown during the live broadcast SHALL read the same later in the live view and in the finished report. Red cards and injuries, which are hidden while live, MUST NOT change which line any other event gets.

#### Scenario: Line kept after full time
- **WHEN** a viewer reads a minute-20 goal line live and opens the finished report later
- **THEN** that goal and its situation sentence read the same

#### Scenario: Hidden injury
- **WHEN** a match has an injury in minute 30 and a substitution in minute 31
- **THEN** the substitution line is the same live and finished, and it never says the player coming off is injured

### Requirement: Kickoff line
The chronicle SHALL open, after the venue and referee lines when there are any, with one kickoff line from a pool of at least 6, which may name `{{home}}` and `{{away}}`. Which line MUST depend only on the match (its two teams and kickoff time).

#### Scenario: Live and finished
- **WHEN** a viewer reads a match live and again after full time
- **THEN** the kickoff line is the same both times

### Requirement: Full-time line by result
When the match is over, the chronicle SHALL close with a line for its result: goalless draw, scoring draw, home win, away win, or a win by 3 or more, naming the winner and `{{score}}`. An abandoned match or a default result MUST keep the neutral "the referee blows the final whistle" line.

#### Scenario: Big away win
- **WHEN** a match ends 0-4
- **THEN** the closing line comes from the win-by-3-or-more pool and names the away side

#### Scenario: Default result
- **WHEN** a match was awarded as a default result
- **THEN** the closing line is the neutral full-time line
