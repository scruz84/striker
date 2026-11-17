# Spec Delta

## MODIFIED Requirements

### Requirement: Periodic jobs once per slot
A job meant to run once per period SHALL first claim its slot (fire time truncated to the period, UTC) in `engine_job_runs`; only the winner runs. A database error counts as not claimed.

#### Scenario: Weekly free agent injection
- **WHEN** Sunday 00:00 UTC fires on two replicas
- **THEN** one batch of free agents is injected, not two

#### Scenario: Daily instant archive
- **WHEN** the daily instant archive fires on two replicas with the archive on
- **THEN** one replica runs the archive pass and the other does nothing

| Job | Slot |
| --- | --- |
| `free_agent_progression` | 24 h |
| `free_agent_injection` | 24 h (fires weekly) |
| `birthdays` | 24 h |
| `account_lifecycle_daily` | 24 h (inactivity warnings, club releases, account erasures) |
| `offer_response` | 12 h |
| `ai_activity` | 5 min |
| `instant_archive` | 24 h (deletes old seasons' instants while the archive is on) |

