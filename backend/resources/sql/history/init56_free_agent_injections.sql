-- Admin-configurable count of brand-new players injected straight into the free agent pool each
-- real calendar week, to keep the transfer market's free agents tab from running dry in a
-- long-running world where the only other sources are a contract expiring or an early release -
-- see common.InjectFreeAgents/engine.InjectFreeAgentsJob. Zero turns injection off entirely.
ALTER TABLE season ADD COLUMN IF NOT EXISTS free_agent_injections_per_week smallint NOT NULL DEFAULT 5;
