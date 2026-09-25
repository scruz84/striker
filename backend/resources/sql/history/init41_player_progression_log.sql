-- One row per player per season progression run (see engine.ProgressPlayers) - captures the
-- before/after age and seven core skills at the exact moment they change, since the drift itself
-- is randomized (see engine.driftSkill) and can't be recomputed after the fact. Backs the player
-- details page's "latest evolution" section and the squad-wide "latest evolutions" view linked
-- from its notification. ON DELETE CASCADE because RetirePlayer hard-deletes the player row
-- (see common.RetirePlayer) - once a player is gone there's no profile page left to show this on.
CREATE TABLE IF NOT EXISTS player_progression_log (
    id bigserial PRIMARY KEY,
    player_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    season integer NOT NULL,
    age_before integer NOT NULL,
    age_after integer NOT NULL,
    goal_keeper_before integer NOT NULL,
    goal_keeper_after integer NOT NULL,
    defense_before integer NOT NULL,
    defense_after integer NOT NULL,
    lateral_before integer NOT NULL,
    lateral_after integer NOT NULL,
    pass_before integer NOT NULL,
    pass_after integer NOT NULL,
    midfield_before integer NOT NULL,
    midfield_after integer NOT NULL,
    dribbling_before integer NOT NULL,
    dribbling_after integer NOT NULL,
    shot_before integer NOT NULL,
    shot_after integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- GetLatestPlayerEvolution (one player's own page) and GetLatestPlayerEvolutionsForTeam (the
-- squad-wide view) both want "most recent row(s) for a set of players" - id DESC is equivalent to
-- created_at DESC here (rows are only ever inserted, never reordered) and lets the index double
-- as a covering order for the "most recent per player" query pattern.
CREATE INDEX IF NOT EXISTS idx_player_progression_log_player_id ON player_progression_log(player_id, id DESC);
