-- Player watchlist (see openspec/specs/player-watchlist and common/watchlist.go): the players a
-- club follows, at most 30 (common.MaxWatchlistSize). Only the following club ever reads its rows.
-- A deleted club or player takes its rows with it. Re-runnable.
CREATE TABLE IF NOT EXISTS player_watchlist (
    team_id    bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    player_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, player_id)
);

CREATE INDEX IF NOT EXISTS player_watchlist_player_idx ON player_watchlist (player_id);
