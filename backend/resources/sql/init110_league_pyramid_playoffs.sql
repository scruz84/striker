-- League pyramid with promotion playoffs (openspec/changes/restructure-league-pyramid-playoffs,
-- specs league-pyramid and match-scheduling).
--
-- * cups.kind tells a domestic/international cup ('cup') from a country's end-of-season promotion
--   playoff ('playoff'). A playoff is one round of cup_ties, so it reuses the cup-tie simulation,
--   lineups and live broadcast; cups_playoff_once makes sure a country gets one per season even
--   with several engine replicas (engine.startPlayoff inserts with ON CONFLICT DO NOTHING).
-- * leagues.archived_season marks a league a restructure retired (see engine.applyRestructure): it
--   keeps its past seasons and stays reachable by id, but no listing used for play returns it.
-- * season_history.via_playoff records that a promotion or relegation came through the playoff.
-- * world_config moves a world still on the old default 1/4/16 x 10 pyramid to the new default
--   1/2/4/8/16 x 10 (common.DefaultWorldTiers). A custom saved structure is left alone.
-- * countries.league_kickoff_minute is the local minute of day tier 1 kicks off (default 14:00);
--   every lower tier kicks off 15 minutes later (common.EffectiveLeagueSchedule). The per-league
--   overrides that adding a country used to write (the same 14:00, converted to the server clock)
--   are removed once, when the column is created, so every league follows the new rule.
--
-- Re-runnable: every step is IF NOT EXISTS or guarded.

ALTER TABLE cups ADD COLUMN IF NOT EXISTS kind varchar(16) NOT NULL DEFAULT 'cup';
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'cups_kind_check') THEN
        ALTER TABLE cups ADD CONSTRAINT cups_kind_check CHECK (kind IN ('cup', 'playoff'));
    END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS cups_playoff_once ON cups (country, season) WHERE kind = 'playoff';

ALTER TABLE leagues ADD COLUMN IF NOT EXISTS archived_season int;
CREATE INDEX IF NOT EXISTS leagues_active_idx ON leagues (country, tier) WHERE archived_season IS NULL;

ALTER TABLE season_history ADD COLUMN IF NOT EXISTS via_playoff boolean NOT NULL DEFAULT false;

UPDATE world_config
SET tiers = '[{"tier":1,"name":"I","groups":1,"teamsPerLeague":10},
              {"tier":2,"name":"II","groups":2,"teamsPerLeague":10},
              {"tier":3,"name":"III","groups":4,"teamsPerLeague":10},
              {"tier":4,"name":"IV","groups":8,"teamsPerLeague":10},
              {"tier":5,"name":"V","groups":16,"teamsPerLeague":10}]'::jsonb
WHERE tiers = '[{"tier":1,"name":"I","groups":1,"teamsPerLeague":10},
                {"tier":2,"name":"II","groups":4,"teamsPerLeague":10},
                {"tier":3,"name":"III","groups":16,"teamsPerLeague":10}]'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'countries' AND column_name = 'league_kickoff_minute'
    ) THEN
        ALTER TABLE countries ADD COLUMN league_kickoff_minute smallint NOT NULL DEFAULT 840;
        DELETE FROM league_schedule;
    END IF;
END $$;
