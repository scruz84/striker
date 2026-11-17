-- Optional archive of old seasons' per-minute instants (openspec/specs/match-data-retention).
-- An admin turns it on and picks how many finished seasons keep their instants
-- (common.SetInstantArchiveSettings). While it is on, the daily engine.InstantArchiveJob deletes
-- the instants of every played fixture older than that and marks the fixture archived, so the
-- next pass never looks at it again. Results, events, ratings and lineups are never touched.
--
-- Off by default and keeps 1 finished season once on. The flags default to false, which is a
-- metadata-only change in Postgres 11+ (no table rewrite). Each partial index only holds the
-- played fixtures not archived yet: once the archive runs, that is only the kept seasons.
--
-- Re-runnable: ADD COLUMN / CREATE INDEX IF NOT EXISTS.

ALTER TABLE season ADD COLUMN IF NOT EXISTS archive_instants_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE season ADD COLUMN IF NOT EXISTS archive_keep_seasons smallint NOT NULL DEFAULT 1
    CHECK (archive_keep_seasons BETWEEN 0 AND 10);

ALTER TABLE matches ADD COLUMN IF NOT EXISTS instants_archived boolean NOT NULL DEFAULT false;
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS instants_archived boolean NOT NULL DEFAULT false;
ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS instants_archived boolean NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS matches_instants_to_archive ON matches (season) WHERE played AND NOT instants_archived;
CREATE INDEX IF NOT EXISTS cup_ties_instants_to_archive ON cup_ties (cup_id) WHERE played AND NOT instants_archived;
CREATE INDEX IF NOT EXISTS friendly_matches_instants_to_archive ON friendly_matches (match_date) WHERE played AND NOT instants_archived;
