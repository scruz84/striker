-- Match-report flavor (Hattrick-style match report): weather, referee/assistants, and the
-- crowd size - generated once at simulation time (see common.UpdateMatchFlavor) so the report
-- shows the same numbers/names on every later view rather than re-rolling them. NULL for a
-- match that hasn't been simulated yet, or one simulated before this column existed.
ALTER TABLE matches ADD COLUMN IF NOT EXISTS weather VARCHAR(20);
ALTER TABLE matches ADD COLUMN IF NOT EXISTS attendance INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS referee_name VARCHAR(120);
ALTER TABLE matches ADD COLUMN IF NOT EXISTS assistant1_name VARCHAR(120);
ALTER TABLE matches ADD COLUMN IF NOT EXISTS assistant2_name VARCHAR(120);
