-- Lets one cups row represent an international competition (issue #64) - entrants drawn from
-- several countries in the same region (see common.RegionForCountry) rather than one country's
-- own tier-1 teams. cup_ties/cup_tie_events and everything downstream (bracket generation,
-- simulation, lineups) already only ever reference team ids directly, with no country scoping,
-- so no other schema changes are needed for the bracket itself.
--
-- DROP COLUMN NOT NULL and ADD COLUMN IF NOT EXISTS are both safe to re-run. The constraint add
-- is guarded explicitly since Postgres has no "ADD CONSTRAINT IF NOT EXISTS" syntax - re-running
-- this against a database that already has it would otherwise fail with a duplicate-object error
-- on every subsequent startup (see common.EnsureSchema/applyPendingMigrations).
ALTER TABLE cups ALTER COLUMN country DROP NOT NULL;
ALTER TABLE cups ADD COLUMN IF NOT EXISTS region varchar(20);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'cups_country_xor_region') THEN
        ALTER TABLE cups ADD CONSTRAINT cups_country_xor_region CHECK ((country IS NULL) <> (region IS NULL));
    END IF;
END $$;
