-- Explicit in-game age, separate from birth_day (which was, at the time this column was added,
-- just set to the wall-clock creation time for every player and never touched again - effectively
-- decorative). Age is what actually drives player progression - see engine/progression.go.
--
-- birth_day has since taken on a real role: engine.ProcessBirthdays now rolls it forward every
-- engine.BirthdayCycleDays and bumps this column by one each time a player's birthday comes
-- around, so age changes on that cycle rather than once per season - see that function.
--
-- Runs automatically after init5_transfers.sql on a fresh database. On an already-initialized
-- one, apply it manually once - see backend/README.md.

ALTER TABLE players ADD COLUMN IF NOT EXISTS age int NOT NULL DEFAULT 22;
