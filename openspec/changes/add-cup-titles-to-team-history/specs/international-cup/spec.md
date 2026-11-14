# Spec Delta

## ADDED Requirements

### Requirement: Regional cup title in the champion's history
When a regional cup edition completes, its champion SHALL get the title in its history (see team-history "International cup title"), in the league season its country is playing at that moment. The edition number MUST never be used as a league season.

#### Scenario: Champion of edition 3
- **WHEN** a Spanish team wins European Cup edition 3 while Spain plays season 7
- **THEN** the team's season 7 history line shows "European Cup winner (Ed. 3)"

### Requirement: Regional cup page opens a given edition
The International Cup page SHALL read `region` and `cup` from its URL. With a valid `region` it opens that tab; with `cup` it shows that edition of the tab's region instead of the latest one. Switching tabs MUST drop `cup` and show that region's latest edition.

#### Scenario: Link from the history tab
- **WHEN** the user opens `/international-cup?region=europe&cup=41`
- **THEN** the European Cup tab shows edition cup 41 and its bracket

#### Scenario: Back to the latest
- **WHEN** the user then picks the Copa America tab
- **THEN** the page shows the latest Copa America edition
