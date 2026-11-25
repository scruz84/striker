-- Short retention for friendlies' per-minute instants (openspec/specs/match-data-retention
-- "Friendly instants are kept for a number of days"). Friendlies are the most numerous fixtures a
-- world plays and nobody looks at their minute-by-minute detail once they are a few days old, so
-- the daily engine.InstantArchiveJob deletes the instants of every played club friendly whose
-- kickoff is older than friendly_instants_keep_days, whether the season archive is on or not, and
-- flags it instants_archived like the season archive does. National-team friendlies and World Cup
-- group matches (a national team at home) are few and left to the season archive, like league
-- matches and cups. Results, events, ratings and lineups are never touched.
--
-- 7 days by default; 0 keeps friendlies' instants for good (only the season archive, if on, then
-- deletes them). The existing friendly_matches_instants_to_archive partial index (init118) already
-- covers the kickoff range scan.
--
-- Re-runnable: ADD COLUMN IF NOT EXISTS.

ALTER TABLE season ADD COLUMN IF NOT EXISTS friendly_instants_keep_days smallint NOT NULL DEFAULT 7
    CHECK (friendly_instants_keep_days BETWEEN 0 AND 365);
