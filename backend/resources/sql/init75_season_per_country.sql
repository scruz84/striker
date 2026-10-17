-- Moves the season counter off the old global singleton `season` row onto `countries` (one
-- counter per country) so each country can advance its own season independently, instead of
-- engine.AdvanceSeasonIfComplete waiting for every country's matches to finish at once - see
-- engine.AdvanceSeasonsIfComplete. `season` keeps only the config that's genuinely global
-- (transfer_window_open, match_schedule_*, promotion/relegation slot config, etc.).
ALTER TABLE countries ADD COLUMN IF NOT EXISTS current_season int NOT NULL DEFAULT 1;

-- One-time seed from the old global counter, then drop it - guarded so this only ever runs once:
-- this file re-applies on every --svc/--engine startup (see common.applyPendingMigrations), and
-- once season.current_season is gone there's nothing left to seed from, so every later run finds
-- the IF false and skips straight past. COALESCE falls back to 1 (matching the column's own
-- DEFAULT above) rather than leaving every country NULL - and so violating its NOT NULL
-- constraint - on a database whose `season` table has no row yet (EnsureSchema alone never
-- inserts one; only handlers.InitializeGame's createSeason does).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'season' AND column_name = 'current_season'
    ) THEN
        UPDATE countries SET current_season = COALESCE((SELECT current_season FROM season), 1);
        ALTER TABLE season DROP COLUMN current_season;
    END IF;
END $$;

-- The new, deliberately-decoupled cadence for regional/international cups and the World Cup
-- (see engine/cups.go) - these no longer wait for every country to roll over together, so they
-- need their own counter and their own "when did we last check" timestamp rather than piggy-
-- backing on any one country's current_season.
ALTER TABLE season ADD COLUMN IF NOT EXISTS international_cup_season int NOT NULL DEFAULT 1;
ALTER TABLE season ADD COLUMN IF NOT EXISTS last_international_cup_check timestamptz;
