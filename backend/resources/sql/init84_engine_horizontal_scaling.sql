-- Support for running several --engine replicas at once (see common/enginelocks.go).
--
-- engine_job_runs records the last schedule slot each "once per period" engine job ran for. Every
-- replica fires the same cron trigger at the same instant; common.ClaimEngineJobSlot lets exactly
-- one of them claim the slot, so a weekly or daily job (free agent injection, free agent
-- progression, birthdays, offer responses, AI activity) runs once per slot, not once per replica.
CREATE TABLE IF NOT EXISTS engine_job_runs (
    job_name  text PRIMARY KEY,
    last_slot timestamptz NOT NULL
);

-- When a simulation pass claimed a league match or friendly (in_simulation = true). The stuck
-- simulation recovery only re-simulates rows claimed long enough ago that no live pass can still
-- be working on them (see common.StuckSimulationAge), and re-claims them atomically so two
-- replicas never re-simulate the same row. Rows claimed before this migration stay NULL and count
-- as stuck straight away, the same as before.
ALTER TABLE matches ADD COLUMN IF NOT EXISTS in_simulation_since timestamptz;
ALTER TABLE friendly_matches ADD COLUMN IF NOT EXISTS in_simulation_since timestamptz;
