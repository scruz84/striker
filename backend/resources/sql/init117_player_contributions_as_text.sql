-- player_contributions on the three instants tables (added by init89_match_player_ratings.sql as
-- jsonb) becomes text. Go writes and reads the column whole (common.encodePlayerContributions /
-- decodePlayerContributions marshal and unmarshal the JSON) and no query uses JSON operators on
-- it, so jsonb's binary form buys nothing. For this shape, an object of many small numbers, the
-- binary form is also much larger than the JSON text: about 1.3 KB against 0.75 KB per minute.
-- Instants make up about 94% of the data a season stores, and as text a match's instants take
-- about 30% less space.
--
-- Converting rewrites each table under an ACCESS EXCLUSIVE lock. On a database with many played
-- matches this can take a while, and matches cannot be simulated or viewed until it finishes, so
-- deploy it in a maintenance window. Existing values keep their content (jsonb's text output,
-- which the Go decoder reads as before).
--
-- Re-runnable: each column is altered only while it is still jsonb. EnsureSchema holds its
-- advisory lock while it runs, so two replicas starting together never convert a table twice.

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['match_instants', 'cup_tie_instants', 'friendly_instants'] LOOP
        IF EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = current_schema() AND table_name = t
                     AND column_name = 'player_contributions' AND data_type = 'jsonb') THEN
            EXECUTE format('ALTER TABLE %I ALTER COLUMN player_contributions TYPE text USING player_contributions::text', t);
        END IF;
    END LOOP;
END $$;
