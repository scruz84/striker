# Spec Delta

## MODIFIED Requirements

### Requirement: Periodic simulation jobs
The engine SHALL run each competition's simulation from its own cron job, each polling its own table for fixtures that are due: league matches, cup ties and friendlies every minute.

#### Scenario: Per-minute jobs
- **WHEN** the engine process is running
- **THEN** `SimulateMatches`, `SimulateCupTies` and `SimulateFriendlies` each run once a minute

#### Scenario: Slower housekeeping jobs
- **WHEN** the engine process is running
- **THEN** the season-rollover check, the regional cup check, the CPU daily activity, national-team friendly pairing, national-team call-ups and stale live-broadcast recovery each run every 5 minutes
- **AND** stuck-simulation recovery runs every 15 minutes
- **AND** player birthdays run daily at 01:00 UTC, free-agent progression daily at 02:00 UTC, the World Cup job daily at 03:00 UTC, pending-offer responses at 00:00 and 12:00 UTC, and free-agent injection weekly on Sunday at 00:00 UTC
