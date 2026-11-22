-- Indexes for the heaviest statements pg_stat_statements showed on a live world.
--
-- 1. Every foreign key to players(player_id) that had no index on its own column. Deleting one
--    player (a retirement, an academy cleanup) makes Postgres run, per referencing column, the
--    ON DELETE CASCADE / SET NULL statement "... WHERE $1 = <column>". Without an index each one
--    read the whole table - lineup_positions and friendly_lineup_positions hold millions of rows -
--    and a single DELETE FROM players took over a second. Columns that may be NULL (lineup roles,
--    pending orders, positions) only index their non-NULL rows: the statements above, and
--    "player_id = ANY($1)" lookups such as the national-team caps count, never match a NULL.
--
-- 2. player_transfers by buying team and date, for "players this team acquired since X"
--    (common.getPlayerIdsAcquiredSinceStmtSql), which otherwise walked the created_at index of
--    every transfer since X looking for one team's.
--
-- 3. Free agents best-skilled first, for common.GetBestFreeAgentsStmtSql: an AI-managed team's
--    pass reads the first few entries instead of sorting every unowned player. The expression and
--    the WHERE must stay exactly those of GetFreeAgentsStmtSql for Postgres to use it.
--
-- CREATE INDEX blocks writes to its table while it builds, which on the biggest tables takes some
-- seconds at startup. To avoid that on a live world, create the same indexes beforehand from psql
-- with CREATE INDEX CONCURRENTLY (same names); this file then finds them and does nothing.
--
-- Re-runnable: CREATE INDEX IF NOT EXISTS.

-- 1. Foreign keys to players
CREATE INDEX IF NOT EXISTS cup_tie_lineup_captain_id_idx ON cup_tie_lineup (captain_id) WHERE captain_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS cup_tie_lineup_free_kick_taker_id_idx ON cup_tie_lineup (free_kick_taker_id) WHERE free_kick_taker_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS cup_tie_lineup_penalty_taker_id_idx ON cup_tie_lineup (penalty_taker_id) WHERE penalty_taker_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS cup_tie_lineup_orders_player_in_id_idx ON cup_tie_lineup_orders (player_in_id) WHERE player_in_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS cup_tie_lineup_orders_player_out_id_idx ON cup_tie_lineup_orders (player_out_id) WHERE player_out_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS cup_tie_lineup_positions_player_id_idx ON cup_tie_lineup_positions (player_id) WHERE player_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS cup_tie_lineup_substitutions_player_in_id_idx ON cup_tie_lineup_substitutions (player_in_id);
CREATE INDEX IF NOT EXISTS cup_tie_lineup_substitutions_player_out_id_idx ON cup_tie_lineup_substitutions (player_out_id);
CREATE INDEX IF NOT EXISTS default_friendly_lineup_positions_player_id_idx ON default_friendly_lineup_positions (player_id);
CREATE INDEX IF NOT EXISTS default_friendly_lineup_substitutions_player_in_id_idx ON default_friendly_lineup_substitutions (player_in_id);
CREATE INDEX IF NOT EXISTS default_friendly_lineup_substitutions_player_out_id_idx ON default_friendly_lineup_substitutions (player_out_id);
CREATE INDEX IF NOT EXISTS default_lineup_orders_player_in_id_idx ON default_lineup_orders (player_in_id) WHERE player_in_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS default_lineup_orders_player_out_id_idx ON default_lineup_orders (player_out_id) WHERE player_out_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS default_lineup_positions_player_id_idx ON default_lineup_positions (player_id);
CREATE INDEX IF NOT EXISTS default_lineup_roles_captain_id_idx ON default_lineup_roles (captain_id) WHERE captain_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS default_lineup_roles_free_kick_taker_id_idx ON default_lineup_roles (free_kick_taker_id) WHERE free_kick_taker_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS default_lineup_roles_penalty_taker_id_idx ON default_lineup_roles (penalty_taker_id) WHERE penalty_taker_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS default_lineup_substitutions_player_in_id_idx ON default_lineup_substitutions (player_in_id);
CREATE INDEX IF NOT EXISTS default_lineup_substitutions_player_out_id_idx ON default_lineup_substitutions (player_out_id);
CREATE INDEX IF NOT EXISTS friendly_lineup_captain_id_idx ON friendly_lineup (captain_id) WHERE captain_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS friendly_lineup_free_kick_taker_id_idx ON friendly_lineup (free_kick_taker_id) WHERE free_kick_taker_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS friendly_lineup_penalty_taker_id_idx ON friendly_lineup (penalty_taker_id) WHERE penalty_taker_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS friendly_lineup_orders_player_in_id_idx ON friendly_lineup_orders (player_in_id) WHERE player_in_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS friendly_lineup_orders_player_out_id_idx ON friendly_lineup_orders (player_out_id) WHERE player_out_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS friendly_lineup_positions_player_id_idx ON friendly_lineup_positions (player_id) WHERE player_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS friendly_lineup_substitutions_player_in_id_idx ON friendly_lineup_substitutions (player_in_id);
CREATE INDEX IF NOT EXISTS friendly_lineup_substitutions_player_out_id_idx ON friendly_lineup_substitutions (player_out_id);
CREATE INDEX IF NOT EXISTS lineup_captain_id_idx ON lineup (captain_id) WHERE captain_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS lineup_free_kick_taker_id_idx ON lineup (free_kick_taker_id) WHERE free_kick_taker_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS lineup_penalty_taker_id_idx ON lineup (penalty_taker_id) WHERE penalty_taker_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS lineup_orders_player_in_id_idx ON lineup_orders (player_in_id) WHERE player_in_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS lineup_orders_player_out_id_idx ON lineup_orders (player_out_id) WHERE player_out_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS lineup_positions_player_id_idx ON lineup_positions (player_id) WHERE player_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS lineup_substitutions_player_in_id_idx ON lineup_substitutions (player_in_id);
CREATE INDEX IF NOT EXISTS lineup_substitutions_player_out_id_idx ON lineup_substitutions (player_out_id);
CREATE INDEX IF NOT EXISTS player_offer_rejections_player_id_idx ON player_offer_rejections (player_id);

-- 2. Transfers by buying team
CREATE INDEX IF NOT EXISTS player_transfers_to_team_created_idx ON player_transfers (to_team_id, created_at);

-- 3. Best free agents first
CREATE INDEX IF NOT EXISTS players_free_agents_by_skill_idx ON players
    ((goal_keeper + defense + "lateral" + pass + midfield + dribbling + shot) DESC) WHERE team_id IS NULL;
