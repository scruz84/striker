-- How a goal / missed chance came about: 0 open play, 1 penalty, 2 direct free kick (see
-- common.SET_PIECE_*). A set piece reuses the usual GOAL / NO_GOAL event, so everything that counts
-- goals keeps working. The default makes every event stored before this migration read as open
-- play, so no backfill is needed; re-running it is a no-op.
ALTER TABLE match_events ADD COLUMN IF NOT EXISTS set_piece smallint NOT NULL DEFAULT 0;
ALTER TABLE cup_tie_events ADD COLUMN IF NOT EXISTS set_piece smallint NOT NULL DEFAULT 0;
ALTER TABLE friendly_events ADD COLUMN IF NOT EXISTS set_piece smallint NOT NULL DEFAULT 0;
