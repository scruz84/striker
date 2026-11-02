# Spec Delta

## ADDED Requirements

### Requirement: Captain's armband
From kickoff on, the lineups pitch and the ratings list of the live and finished views SHALL mark each side's captain with a "C" armband. Named set-piece takers MUST NOT be shown to other managers.

#### Scenario: Finished match
- **WHEN** a viewer opens a finished match in which the home captain started
- **THEN** his chip on the lineups pitch shows a "C" armband

#### Scenario: No captain
- **WHEN** a side played without a named captain
- **THEN** no armband is shown for it
