-- User account lifecycle (OpenSpec user-account-lifecycle): activity tracking, terms acceptance,
-- the "your club was released" notice and the pending-reset marker for released clubs.
-- Re-runnable: every statement is guarded.

ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen_at timestamptz;
-- Grace period on upgrade: nobody is treated as idle since before this migration ran.
UPDATE users SET last_seen_at = now() WHERE last_seen_at IS NULL;
ALTER TABLE users ALTER COLUMN last_seen_at SET DEFAULT now();
ALTER TABLE users ALTER COLUMN last_seen_at SET NOT NULL;
-- 0 none, 1 = 14-day club warning sent, 2 = 7-day club warning sent, 3 = 14-day erasure warning
-- sent. Any recorded activity sets it back to 0.
ALTER TABLE users ADD COLUMN IF NOT EXISTS inactivity_warning_stage smallint NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS terms_version int;            -- NULL = never accepted
ALTER TABLE users ADD COLUMN IF NOT EXISTS terms_accepted_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS released_team_name varchar(200); -- shown once at onboarding

-- Set when a club loses its owner; the engine resets the club later (common.ResetReleasedTeam).
ALTER TABLE teams ADD COLUMN IF NOT EXISTS reset_pending_since timestamptz;
-- Why the club was released (inactivity, self_deletion, admin_deletion, erasure) - only for the
-- striker.teams.reset metric's `reason` attribute, read when the reset actually runs.
ALTER TABLE teams ADD COLUMN IF NOT EXISTS reset_reason varchar(20);
-- A loaned-out player of a reset club becomes a free agent when his loan ends.
ALTER TABLE player_loans ADD COLUMN IF NOT EXISTS release_on_return boolean NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS users_last_seen_idx ON users(last_seen_at);
CREATE INDEX IF NOT EXISTS teams_reset_pending_idx ON teams(reset_pending_since)
  WHERE reset_pending_since IS NOT NULL;
