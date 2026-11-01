# Spec Delta

## MODIFIED Requirements

### Requirement: Knockout stage
Once every group match is played and the last one's live broadcast has finished (its kickoff at least 2 hours ago), the top two of each group SHALL be shuffled into a knockout cup (region `world`) run by the ordinary cup machinery, with each national side's lineup picked automatically per round. When that cup completes, its champion MUST be recorded on the World Cup.

#### Scenario: Group stage complete
- **WHEN** the last group match is played and its kickoff was at least 2 hours ago
- **THEN** the next cup job creates the knockout cup from the 2 best of every group
- **AND** only one knockout cup is created even when several engine replicas run that job at once

#### Scenario: Last group match still being broadcast
- **WHEN** every group match is played but the last one kicked off 40 minutes ago
- **THEN** no knockout cup is created, no knockout call-up is notified and the World Cup stays in its group stage
- **AND** the first cup job after that kickoff plus 2 hours creates the knockout cup

#### Scenario: Final played
- **WHEN** the knockout cup completes
- **THEN** the World Cup is marked completed with that champion
