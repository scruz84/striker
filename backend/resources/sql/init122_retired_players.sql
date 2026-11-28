-- Retired players keep their record (openspec/specs/retired-players). A player who retires is
-- deleted from `players` (common.RetirePlayer, common.RetireFreeAgentAsTrainer); before this file
-- his Golden Boots, Player of the Season awards, transfers and club history went with him through
-- ON DELETE CASCADE, so nothing could ever look back on his career.
--
-- retired_players is the snapshot written in the same transaction as the delete: who he was, the
-- position he played best, the age he retired at, and the club and season he retired from. The
-- three history tables drop their foreign key to players, so their rows outlive him; a world reset
-- (handlers.teamPlayerDependentTables) clears them explicitly instead.
--
-- Re-runnable, and takes no blocking lock once applied: CREATE TABLE IF NOT EXISTS, and each
-- constraint dropped only while it exists.

CREATE TABLE IF NOT EXISTS retired_players (
    player_id      bigint PRIMARY KEY,
    first_name     varchar(200) NOT NULL DEFAULT '',
    last_name      varchar(200) NOT NULL DEFAULT '',
    birth_country  varchar(4),
    avatar         varchar(1000),
    best_position  smallint NOT NULL DEFAULT 0,
    age            smallint NOT NULL DEFAULT 0,
    -- The club he belonged to when he retired (a free agent's last club); NULL when he never had
    -- one. The name is kept too, so the profile reads right even after a club is renamed.
    last_team_id   bigint REFERENCES teams(team_id) ON DELETE SET NULL,
    last_team_name varchar(200),
    -- The current season of his last club's country when he retired; NULL without a club.
    retired_season int,
    retired_at     timestamptz NOT NULL DEFAULT now()
);

-- This file runs again on every startup, and ALTER TABLE takes an ACCESS EXCLUSIVE lock on its
-- table even when the constraint is already gone - held until the file ends, against transfers
-- writing those tables (see init120). Each constraint is therefore only dropped while pg_constraint,
-- read without a table lock, still lists it.
DO $$
DECLARE
    fk record;
BEGIN
    FOR fk IN SELECT * FROM (VALUES
        ('player_season_awards', 'player_season_awards_player_id_fkey'),
        ('player_transfers', 'player_transfers_player_id_fkey'),
        ('player_club_history', 'player_club_history_player_id_fkey')
    ) AS v(tbl, name) LOOP
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = fk.name AND conrelid = to_regclass(fk.tbl)) THEN
            EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', fk.tbl, fk.name);
        END IF;
    END LOOP;
END
$$;
