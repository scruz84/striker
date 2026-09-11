-- Accumulates each player's minutes played at each tactical position across the current
-- season's competitive matches (league + cup - deliberately NOT friendlies, which are meant to
-- have no effect beyond the match itself, same as they have no ticket revenue/classification
-- effect - see init26_friendlies.sql). Consumed and cleared once per season by
-- engine.ProgressPlayers, which gates skill growth on having actually played, restricted to the
-- skills relevant to whichever position a player spent the most minutes at, scaled by how many
-- minutes they played in total.
CREATE TABLE IF NOT EXISTS player_position_minutes (
    player_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    position  smallint NOT NULL,
    minutes   int NOT NULL DEFAULT 0,
    PRIMARY KEY (player_id, position)
);
