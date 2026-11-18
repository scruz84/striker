-- Adds a team-level "socios" (club members) count: the paying core of a club's home crowd, which
-- replaces "any stadium you build fills up" as what drives matchday attendance - see
-- engine/stadium.go's CalculateAttendance (attendance = min(capacity, socios x price x form x
-- rival x weather x noise)) and engine/socios.go, which drifts it toward a fanbase-and-position
-- target at every season rollover. DEFAULT 500 is common.DefaultSocios, the crowd a default
-- 1000-seat ground drew at the default ticket price before this existed.
--
-- Migrations are re-applied at every startup (see common.applyPendingMigrations), so the one-off
-- backfill of existing teams to their fanbase-derived target (2.2 x fanbase^1.5, the same curve
-- engine.sociosTarget aims at) only runs when the column is first created - never again, which
-- would otherwise reset every team's socios on each restart.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'teams' AND column_name = 'socios'
    ) THEN
        ALTER TABLE teams ADD COLUMN socios integer NOT NULL DEFAULT 500;
        UPDATE teams SET socios = round(2.2 * power(fanbase, 1.5));
    END IF;
END
$$;
