-- Short retention for every fixture's per-minute instants (openspec/specs/match-data-retention
-- "Instants are kept for a number of days"). The instants are about 94% of what a match stores and
-- are only read by the live replay (common.LiveBroadcastWindow) and by a finished match's zone duel
-- and rating scrubber, so the daily engine.InstantArchiveJob deletes the instants of every played
-- league match, cup tie and friendly whose kickoff is older than instants_keep_days, whether the
-- season archive is on or not, and flags it instants_archived like the season archive does.
-- Results, events, ratings and lineups are never touched.
--
-- 7 days by default; 0 keeps instants for good (only the season archive, if on, then deletes
-- them). The partial indexes on kickoff only hold played fixtures not archived yet, so once the
-- first pass has run they only hold the last days' fixtures. friendly_matches already has one
-- (friendly_matches_instants_to_archive, init118).
--
-- CREATE INDEX blocks writes to matches and cup_ties while it builds. To avoid that on a live
-- world, create the same two indexes beforehand from psql with CREATE INDEX CONCURRENTLY (same
-- names); this file then finds them and does nothing.
--
-- Re-runnable: ADD COLUMN / CREATE INDEX IF NOT EXISTS.

ALTER TABLE season ADD COLUMN IF NOT EXISTS instants_keep_days smallint NOT NULL DEFAULT 7
    CHECK (instants_keep_days BETWEEN 0 AND 365);

CREATE INDEX IF NOT EXISTS matches_instants_to_expire ON matches (match_date) WHERE played AND NOT instants_archived;
CREATE INDEX IF NOT EXISTS cup_ties_instants_to_expire ON cup_ties (match_date) WHERE played AND NOT instants_archived;
