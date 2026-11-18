-- Stoppage time (openspec/specs/stoppage-time): each half of a match gets 1-5 added minutes. A
-- minute in added time is its clock minute (45 or 90) plus an added minute (1..N); a regulation
-- minute has added minute 0, so every row stored before this change reads as before.
--   * added_minute on the three event and three instant tables, part of the instants' primary key
--     (a first half with added time stores several instants at minute 45);
--   * added_time on the event tables: the number of minutes an ADDED_TIME event (type 9) announces;
--   * first_half_added_time / second_half_added_time on each fixture.
-- Re-runnable.
ALTER TABLE match_events ADD COLUMN IF NOT EXISTS added_minute smallint NOT NULL DEFAULT 0;
ALTER TABLE match_events ADD COLUMN IF NOT EXISTS added_time smallint;
ALTER TABLE cup_tie_events ADD COLUMN IF NOT EXISTS added_minute smallint NOT NULL DEFAULT 0;
ALTER TABLE cup_tie_events ADD COLUMN IF NOT EXISTS added_time smallint;
ALTER TABLE friendly_events ADD COLUMN IF NOT EXISTS added_minute smallint NOT NULL DEFAULT 0;
ALTER TABLE friendly_events ADD COLUMN IF NOT EXISTS added_time smallint;

ALTER TABLE match_instants ADD COLUMN IF NOT EXISTS added_minute smallint NOT NULL DEFAULT 0;
ALTER TABLE cup_tie_instants ADD COLUMN IF NOT EXISTS added_minute smallint NOT NULL DEFAULT 0;
ALTER TABLE friendly_instants ADD COLUMN IF NOT EXISTS added_minute smallint NOT NULL DEFAULT 0;

-- Widen each instants table's primary key to (id, minute, added_minute), once: the block only acts
-- while the current primary key does not include added_minute yet.
DO $$
DECLARE
    t record;
    pk_name text;
BEGIN
    FOR t IN SELECT * FROM (VALUES ('match_instants', 'match_id'), ('cup_tie_instants', 'tie_id'),
                                   ('friendly_instants', 'match_id')) AS v(tbl, id_col) LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_index i
            JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY (i.indkey)
            WHERE i.indrelid = t.tbl::regclass AND i.indisprimary AND a.attname = 'added_minute'
        ) THEN
            SELECT c.conname INTO pk_name FROM pg_constraint c
            WHERE c.conrelid = t.tbl::regclass AND c.contype = 'p';
            IF pk_name IS NOT NULL THEN
                EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', t.tbl, pk_name);
            END IF;
            EXECUTE format('ALTER TABLE %I ADD PRIMARY KEY (%I, minute, added_minute)', t.tbl, t.id_col);
        END IF;
    END LOOP;
END $$;

ALTER TABLE matches ADD COLUMN IF NOT EXISTS first_half_added_time smallint NOT NULL DEFAULT 0;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS second_half_added_time smallint NOT NULL DEFAULT 0;
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS first_half_added_time smallint NOT NULL DEFAULT 0;
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS second_half_added_time smallint NOT NULL DEFAULT 0;
ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS first_half_added_time smallint NOT NULL DEFAULT 0;
ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS second_half_added_time smallint NOT NULL DEFAULT 0;
