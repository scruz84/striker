-- Per-country league structures and dissolved teams (openspec/specs/league-pyramid).
--
-- countries.structure is a country's own structure, in world_config.tiers' format. NULL means the
-- country follows the global world_config (see common.EffectiveStructure). A restructure reshapes
-- the country into it at its next rollover.
--
-- teams.dissolved_season is the season a CPU team was dissolved in by a restructure with fewer
-- places than teams (see common.DissolveTeam). A dissolved team keeps its row, past matches and
-- history but never plays again: it leaves its league (league_id = -1) and every club listing skips
-- it. The partial index serves those listings.
--
-- Re-runnable: ADD COLUMN / CREATE INDEX IF NOT EXISTS.

ALTER TABLE countries ADD COLUMN IF NOT EXISTS structure jsonb;
ALTER TABLE teams ADD COLUMN IF NOT EXISTS dissolved_season int;

CREATE INDEX IF NOT EXISTS teams_active_clubs ON teams (country) WHERE dissolved_season IS NULL AND NOT is_national_team;
