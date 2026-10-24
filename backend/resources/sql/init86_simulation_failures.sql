-- Bounded, recoverable match simulation (see openspec harden-stuck-match-recovery).
--
-- simulation_attempts counts how many times a claim has tried to simulate a fixture; a fixture
-- that reaches common.MaxSimulationAttempts without being played is settled 0-0 instead of being
-- retried forever (engine.SettleDefaultResults). last_simulation_error keeps the reason the latest
-- attempt failed, and last_simulation_attempt_at drives the retry backoff
-- (common.SimulationRetryBackoff). A cup tie has no claim of its own (it is simulated under an
-- advisory lock), so it only uses the attempt columns.
--
-- Rows already stuck when the columns are first added have been failing unnoticed for an unknown
-- time, so they get one more attempt on the fixed code (2 of 3 already used) and are then settled
-- 0-0. That seeding sits inside the column-existence guard, so it only ever runs on the start-up
-- that adds the columns: a later long outage must never be mistaken for an old stuck row. A cup
-- tie cannot be detected this way (a failing tie was never claimed), so it starts counting from
-- zero.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = current_schema() AND table_name = 'matches'
                     AND column_name = 'simulation_attempts') THEN
        ALTER TABLE matches ADD COLUMN simulation_attempts int NOT NULL DEFAULT 0;
        ALTER TABLE matches ADD COLUMN last_simulation_error text;
        ALTER TABLE matches ADD COLUMN last_simulation_attempt_at timestamptz;
        ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS simulation_attempts int NOT NULL DEFAULT 0;
        ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS last_simulation_error text;
        ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS last_simulation_attempt_at timestamptz;

        UPDATE matches SET simulation_attempts = 2
        WHERE played = false AND in_simulation = true
          AND (in_simulation_since IS NULL OR in_simulation_since < now() - interval '1 hour');
        UPDATE friendly_matches SET simulation_attempts = 2
        WHERE played = false AND in_simulation = true
          AND (in_simulation_since IS NULL OR in_simulation_since < now() - interval '1 hour');
    END IF;
END $$;

ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS simulation_attempts int NOT NULL DEFAULT 0;
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS last_simulation_error text;
ALTER TABLE cup_ties ADD COLUMN IF NOT EXISTS last_simulation_attempt_at timestamptz;

-- Post-match effects (availability countdown and new injuries/suspensions, skill growth, result
-- notification) that wait for the live replay to finish. Written in the same transaction as the
-- result, claimed with DELETE ... RETURNING by whoever applies them: the replay's completion, or
-- the 5-minute recovery job once due_at (kickoff + common.LiveBroadcastWindow) has passed - so an
-- engine restart during the replay no longer loses them.
CREATE TABLE IF NOT EXISTS pending_post_match_effects (
    kind     text        NOT NULL,   -- 'league' | 'cup' | 'friendly'
    match_id bigint      NOT NULL,
    due_at   timestamptz NOT NULL,
    payload  jsonb       NOT NULL,
    PRIMARY KEY (kind, match_id)
);
CREATE INDEX IF NOT EXISTS pending_post_match_effects_due ON pending_post_match_effects (due_at);
