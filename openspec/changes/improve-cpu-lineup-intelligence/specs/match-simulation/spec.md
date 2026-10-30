# Spec Delta

## MODIFIED Requirements

### Requirement: Unavailable players never take the field
Before kickoff the simulator MUST remove from the saved lineup every outfield player serving an injury or suspension, then refill each thinned position back to its original headcount with the best remaining eligible squad player for that position. Positions are refilled in the formation's fixed slot order (goalkeeper, defence, midfield, attack). If the squad runs out, the position stays short.

#### Scenario: Suspended striker in the saved lineup
- **WHEN** a suspended striker is still in a team's saved lineup
- **THEN** he does not play and the best available squad player for that position takes his slot

#### Scenario: Squad crisis
- **WHEN** no eligible player is left for a position
- **THEN** that position is left short rather than fielding anyone twice or anyone unavailable

#### Scenario: Suspensions in friendlies
- **WHEN** a player serving a red-card suspension is in a friendly lineup
- **THEN** he may play, because suspensions only bar league and cup matches (injuries still apply)

#### Scenario: Two positions need the same reserve
- **WHEN** an injured defender and an injured midfielder are both best replaced by the same bench player
- **THEN** he fills the defence slot, because defence comes first in slot order, and the midfield slot takes the next best player

### Requirement: Injuries
Each minute SHALL have a 0.5% chance of injuring a random outfield player, removing him for the rest of the match and sidelining him for 1–5 matches. If the team has not used its 3 substitutions, the best available bench player MUST replace him as a logged substitution. A player already substituted off this match is not available. Goalkeepers are never injured.

#### Scenario: Substitutions left
- **WHEN** a player is injured and his team has used 2 of 3 substitutions
- **THEN** the best available bench player comes on in his place

#### Scenario: No substitutions left
- **WHEN** a player is injured and all 3 substitutions are used
- **THEN** the team plays on a man down

#### Scenario: Best option already came off
- **WHEN** a player is injured in minute 80 and the best player for his position was substituted off in minute 60
- **THEN** the next best eligible bench player comes on instead

## ADDED Requirements

### Requirement: Stored lineup is the XI that kicked off
A match's stored lineup snapshot MUST hold the XI after the pre-kickoff removal and refill of unavailable players, together with the scheduled substitutions it was played with. So the Lineups tab, the ratings view and the live viewer name the player who actually started in each slot.

#### Scenario: Injured player in a human's saved lineup
- **WHEN** a human's saved lineup names an injured midfielder and the refill puts the best available midfielder in his slot
- **THEN** the match's stored lineup lists the replacement in that slot and does not list the injured player

### Requirement: Scheduled substitutions
A scheduled substitution SHALL fire at its minute only if its outgoing player is still on the pitch and its team has used fewer than 3 substitutions, counting injury replacements. If its incoming player is not eligible, the best eligible bench player for that position comes on instead. Not eligible means unavailable for the match, sent off, injured this match, already on the pitch or already substituted off.

#### Scenario: Cap reached by injuries
- **WHEN** a team has made 3 injury replacements by minute 50 and has a substitution scheduled for minute 60
- **THEN** the scheduled substitution does not fire, so the team never makes more than 3 changes

#### Scenario: Incoming player got injured since the lineup was saved
- **WHEN** a human scheduled a striker to come on at minute 65 and that striker is now injured
- **THEN** at minute 65 the best eligible bench player for the outgoing player's position comes on instead, as a logged substitution

#### Scenario: No eligible bench player
- **WHEN** a scheduled substitution's incoming player is not eligible and nobody on the bench is
- **THEN** the substitution is skipped and does not count against the 3

#### Scenario: Outgoing player already off
- **WHEN** a scheduled substitution's outgoing player was sent off or injured earlier
- **THEN** it is skipped and does not count against the 3

### Requirement: Projected kickoff replacements in the lineup editor
Every lineup a team owner reads for an upcoming league match, cup tie or friendly, and his default lineups, SHALL list for each unavailable starter the player the pre-kickoff refill would put in his slot if the match kicked off now. It is computed with the same rules and slot order as the refill. The saved lineup itself is not changed.

#### Scenario: Injured starter in the default lineup
- **WHEN** an owner opens his default lineup and his starting left-back is injured
- **THEN** the left-back is shown as injured, along with the name of the best available player who would replace him

#### Scenario: Injured player recovers
- **WHEN** that left-back's injury runs out
- **THEN** he is back in the default lineup with no replacement shown, because the saved lineup was never edited
