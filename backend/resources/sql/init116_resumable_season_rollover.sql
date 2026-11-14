-- Resumable season rollover (openspec/changes/make-season-rollover-resumable, spec
-- engine-replicas "Resumable season rollover", issue #401). engine.advanceCountrySeasonIfComplete
-- records each completed step of a country's rollover here, so a run that stops halfway (a
-- restart, an error between steps) resumes at the next step instead of repeating aging, prizes,
-- the cup and promotion/relegation:
--   rollover_season - the season being closed (NULL when no rollover is in progress)
--   rollover_step   - the last step completed, 0 for none (see engine.rolloverStep*)
--   rollover_state  - the final standings and moves later steps need (engine.rolloverState)
--
-- Re-runnable.

ALTER TABLE countries ADD COLUMN IF NOT EXISTS rollover_season integer;
ALTER TABLE countries ADD COLUMN IF NOT EXISTS rollover_step smallint NOT NULL DEFAULT 0;
ALTER TABLE countries ADD COLUMN IF NOT EXISTS rollover_state jsonb;
