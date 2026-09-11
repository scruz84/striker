-- Adds the tier a league sits at in its country's promotion/relegation pyramid (1 = top
-- division). Runs automatically after init.sql on a fresh database (Postgres' docker
-- entrypoint runs every *.sql file in this directory in name order on first init, and
-- "init2_..." sorts right after "init.sql").
--
-- On an ALREADY-INITIALIZED database, docker-entrypoint-initdb.d scripts do NOT re-run -
-- apply this file manually once, e.g.:
--   docker compose exec -T postgres psql -U <user> -d <db> < resources/sql/init2_league_tier.sql

ALTER TABLE leagues ADD COLUMN IF NOT EXISTS tier int NOT NULL DEFAULT 1;
