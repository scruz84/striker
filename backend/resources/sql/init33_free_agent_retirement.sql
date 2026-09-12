-- Tracks how many consecutive seasons a player has gone unsigned (team_id IS NULL) - see
-- engine.ProgressFreeAgents, the only writer, which increments this once per season and forces
-- a free agent's retirement once it reaches engine.FreeAgencyRetirementSeasons. Reset to 0 the
-- moment a player is signed (see common.SignFreeAgent), so it only ever measures the player's
-- *current*, unbroken spell of free agency.
ALTER TABLE players ADD COLUMN IF NOT EXISTS free_agent_seasons integer NOT NULL DEFAULT 0;
