-- Tracks which club a free agent most recently played for, so signing them within their first
-- season of free agency can charge a transfer premium to that club instead of it being a
-- completely free pickup - see common.SignFreeAgent and common.FreeAgentPremiumWageMultiplier.
-- NULL means either the player has never been released (still on their original team) or was
-- unassigned by common.AssignTeam reassigning a CPU squad rather than a real release - neither
-- case owes anyone a premium.
ALTER TABLE players ADD COLUMN IF NOT EXISTS last_team_id bigint REFERENCES teams(team_id);
