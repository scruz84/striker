# Spec Delta

## Purpose

A team's standing lineups (the league/cup default and the friendly default) and their scheduled
substitutions, and which lineup and substitutions each kind of match falls back to when the team
has not saved a per-match override.

## ADDED Requirements

### Requirement: Independent standing substitutions
A team SHALL have two independent sets of standing scheduled substitutions, each of up to 3: one for its league/cup default lineup and one for its friendly default lineup. Saving either standing lineup MUST replace only its own set and MUST NOT read or change the other. A team starts with an empty friendly set; nothing is copied from the league/cup set.

#### Scenario: Saving the friendly default
- **WHEN** a manager with 2 league/cup substitutions saves his friendly default with 1 substitution (minute 60)
- **THEN** reading the friendly default returns that 1 substitution and reading the league/cup default still returns the original 2

#### Scenario: Saving the league/cup default
- **WHEN** the same manager then saves his league/cup default with 3 substitutions
- **THEN** the friendly default still returns its 1 substitution

#### Scenario: Clearing the friendly set
- **WHEN** a manager saves his friendly default with an empty or missing substitutions list
- **THEN** the friendly default has no substitutions and the league/cup default keeps its own

#### Scenario: After the migration
- **WHEN** an existing team with 3 league/cup substitutions opens its friendly default for the first time after the change ships
- **THEN** the friendly default shows no substitutions

### Requirement: Friendly default substitution validation
A substitution saved with the friendly default MUST pass the same checks as any other saved substitution: the outgoing player is one of the default's starters, the incoming player is a squad player not in the XI and not unavailable, no player is used twice, the minute is 1 to 90, at most 3. Availability follows friendly rules, so only an injury bars the incoming player. An invalid save is rejected with 400 and stores nothing.

#### Scenario: Suspended player comes on in a friendly
- **WHEN** the friendly default schedules a red-card-suspended, uninjured bench player to come on at minute 70
- **THEN** the save succeeds

#### Scenario: Injured incoming player
- **WHEN** the friendly default schedules an injured bench player to come on
- **THEN** the save is rejected with 400 and the previous friendly lineup and substitutions are unchanged

#### Scenario: Fourth substitution
- **WHEN** the friendly default is saved with 4 substitutions
- **THEN** the save is rejected with 400

### Requirement: Substitutions a match falls back to
A match SHALL use the substitutions paired with the lineup it plays. League match or cup tie: its override's own if it has one, else the league/cup set. Friendly: its override's own if it has one; else, when the friendly default lineup is used, the friendly set; else (computed best XI) none. A friendly MUST NOT use the league/cup set, nor a league match or cup tie the friendly set. CPU teams keep choosing theirs at kickoff.

#### Scenario: Friendly falls back to the friendly default
- **WHEN** a human team with a saved friendly default and 2 friendly substitutions plays a friendly it saved no override for
- **THEN** it starts the friendly default XI and plays those 2 substitutions, and the stored lineup snapshot of the friendly lists them

#### Scenario: Friendly with its own override
- **WHEN** the team saved an override with 1 substitution for that friendly
- **THEN** the friendly plays only that 1 substitution

#### Scenario: Friendly with no friendly default
- **WHEN** a team has no friendly default saved, 3 league/cup substitutions, and no override for the friendly
- **THEN** it starts its computed best XI with no scheduled substitutions

#### Scenario: League match ignores the friendly set
- **WHEN** a team with 1 league/cup substitution and 3 friendly substitutions plays a league match without an override
- **THEN** it plays the 1 league/cup substitution

### Requirement: Editors show each lineup's own substitutions
The lineup editor's Friendly tab SHALL show and save the friendly default's substitutions, and the League/Cup tab the league/cup ones. The per-friendly editor of a friendly without an override MUST pre-fill its drafts from the substitutions of the lineup it pre-fills from: the friendly default's own, or none when it shows the computed best XI.

#### Scenario: Opening a friendly without an override
- **WHEN** a manager with 2 league/cup substitutions and 1 friendly substitution opens an upcoming friendly he has not edited
- **THEN** the editor shows the friendly default XI with that 1 substitution in the planner

#### Scenario: Friendly tab hint
- **WHEN** a manager opens the Friendly tab of the lineup editor
- **THEN** the hint no longer says scheduled substitutions are shared with the league/cup default, and the substitutions planner is shown
