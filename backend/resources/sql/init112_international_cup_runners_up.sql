-- Two entrants per country in the regional (international) club cups
-- (openspec/changes/add-international-cup-runners-up, spec international-cup).
--
-- * international_cup_entries.final_position is the entrant's place in its country's tier-1
--   season: 1 for the champion, 2 for the runner-up. Every row written before this is a champion.
-- * The primary key moves from (cup_id, country) to (cup_id, team_id), since a country now sends
--   two teams to one edition. Freshness is still max(season) per country, on the same index.
--
-- Re-runnable: the column is IF NOT EXISTS and the key is only swapped while it still contains
-- country.

ALTER TABLE international_cup_entries ADD COLUMN IF NOT EXISTS final_position smallint NOT NULL DEFAULT 1;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
        WHERE c.conrelid = 'international_cup_entries'::regclass AND c.contype = 'p' AND a.attname = 'country'
    ) THEN
        ALTER TABLE international_cup_entries DROP CONSTRAINT international_cup_entries_pkey;
        ALTER TABLE international_cup_entries ADD PRIMARY KEY (cup_id, team_id);
    END IF;
END $$;
