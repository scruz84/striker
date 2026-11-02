-- Team mentality (see openspec/specs/match-simulation "Team mentality" and common.Lineup.Mentality):
-- Defensive (-1), Balanced (0, the default) or Attacking (+1), carried by every lineup kind - the
-- per-match overrides (also the kickoff snapshots the viewer reads) and the two standing defaults
-- on teams. Every existing lineup reads Balanced, so it plays exactly as before. Re-runnable.
ALTER TABLE lineup ADD COLUMN IF NOT EXISTS mentality smallint NOT NULL DEFAULT 0 CHECK (mentality BETWEEN -1 AND 1);
ALTER TABLE cup_tie_lineup ADD COLUMN IF NOT EXISTS mentality smallint NOT NULL DEFAULT 0 CHECK (mentality BETWEEN -1 AND 1);
ALTER TABLE friendly_lineup ADD COLUMN IF NOT EXISTS mentality smallint NOT NULL DEFAULT 0 CHECK (mentality BETWEEN -1 AND 1);
ALTER TABLE teams ADD COLUMN IF NOT EXISTS mentality smallint NOT NULL DEFAULT 0 CHECK (mentality BETWEEN -1 AND 1);
ALTER TABLE teams ADD COLUMN IF NOT EXISTS friendly_mentality smallint NOT NULL DEFAULT 0 CHECK (friendly_mentality BETWEEN -1 AND 1);
