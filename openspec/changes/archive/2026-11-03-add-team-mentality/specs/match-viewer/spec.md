# Spec Delta

## ADDED Requirements

### Requirement: Mentality in the lineups
From kickoff on, the lineups pitch of the live and finished views SHALL show each side's mentality at kickoff (Defensive, Balanced or Attacking) in the viewer's language. Before kickoff only a participating owner sees his own side's mentality, in his lineup editor.

#### Scenario: Finished match
- **WHEN** a viewer opens a finished league match in which the home side played Attacking
- **THEN** the lineups pitch labels the home side "Attacking" ("Ofensiva" in Spanish)

#### Scenario: Before kickoff
- **WHEN** a manager opens an upcoming match of two other clubs
- **THEN** no mentality is shown for either side
