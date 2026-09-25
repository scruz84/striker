-- player_position_minutes (see resources/sql/history/init29_player_position_minutes.sql) fed a
-- season-end skill-drift step in engine.ProgressPlayers that no longer exists: skill growth is
-- now applied right after each match a player features in (see engine.updatePlayerSkillsAfterMatch,
-- called from engine.simulateMatch/simulateCupTie), using that match's own in-memory per-player
-- minutes rather than a running season-long accumulator. Nothing else ever read this table, so
-- it's dropped outright rather than left around unused - also removed from init.sql itself, so a
-- brand-new database no longer creates it at all (this migration still applies there too, as a
-- harmless no-op, since a fresh install runs every numbered migration alongside init.sql).
DROP TABLE IF EXISTS player_position_minutes;
