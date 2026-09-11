-- Tracks whether a team has already used its one allowed rename (see common.RenameTeam) -
-- defaults false so every existing team still gets to rename once.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS name_changed boolean NOT NULL DEFAULT false;
