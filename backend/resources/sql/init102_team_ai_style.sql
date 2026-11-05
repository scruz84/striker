-- A CPU club's style (see common.AiClubStyle* and openspec/specs/cpu-team-management "Club
-- style"): 0 balanced, 1 youth, 2 frugal, 3 ambitious. NULL reads as balanced; real clubs get one
-- when created (handlers.createTeam) or from common.BackfillAiStyles at service startup. Re-runnable.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS ai_style smallint;
