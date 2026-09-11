-- Explicit in-game age, separate from birth_day (which is set to the wall-clock creation
-- time for every player - effectively decorative, since this game's seasons complete in
-- hours of real time, far too fast for wall-clock aging to mean anything). Age is what
-- actually drives player progression - see engine/progression.go.
--
-- Runs automatically after init5_transfers.sql on a fresh database. On an already-initialized
-- one, apply it manually once - see backend/README.md.

ALTER TABLE players ADD COLUMN IF NOT EXISTS age int NOT NULL DEFAULT 22;
