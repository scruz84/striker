# Spec Delta

## ADDED Requirements

### Requirement: Stored match record for every competition
Every simulated match SHALL store the same record, whatever its competition (league match, cup tie or friendly): its events, its per-minute zone instants (one per simulated minute) and its report flavor. The report flavor is the weather, attendance, referee and two assistant referees. Flavor generation is best-effort: if a name can't be drawn, the flavor is skipped and the match result stands.

#### Scenario: Cup tie simulated
- **WHEN** a cup tie finishes simulating after 90 minutes
- **THEN** it has 90 stored instants (minutes 1 to 90) and a stored weather, attendance, referee and two assistants

#### Scenario: Friendly simulated
- **WHEN** a friendly (club, national-team or World Cup group) finishes simulating
- **THEN** its stored weather and attendance are the same values that set its ticket revenue

#### Scenario: Forfeit
- **WHEN** a cup tie stops in minute 61 because one side dropped to 7 players
- **THEN** it stores the instants for minutes 1 to 61 and its flavor, the same as a league match does

#### Scenario: Referee names unavailable
- **WHEN** drawing a referee name fails after a cup tie was simulated
- **THEN** the tie's result, events and instants are still stored, and its flavor stays empty
