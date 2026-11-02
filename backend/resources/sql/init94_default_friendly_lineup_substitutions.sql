-- A team's standing scheduled substitutions for its friendly default lineup, independent of the
-- league/cup set in default_lineup_substitutions. Same shape as that table; no data is copied from
-- it, so every team starts with none. Re-runnable.
CREATE TABLE IF NOT EXISTS default_friendly_lineup_substitutions (
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    player_out_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    minute        smallint NOT NULL CHECK (minute BETWEEN 1 AND 90),
    PRIMARY KEY (team_id, slot)
);
