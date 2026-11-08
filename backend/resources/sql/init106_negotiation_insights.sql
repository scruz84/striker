-- Negotiation insights (openspec/specs/player-negotiation "Negotiation insight"): a club that has
-- negotiated with a player (renewal, free-agent signing, transfer offer, pre-contract offer or
-- clause payment he judged) sees his negotiation tier on his page. One row per club and player.
-- Re-runnable.
CREATE TABLE IF NOT EXISTS negotiation_insights (
    team_id    bigint NOT NULL REFERENCES teams (team_id) ON DELETE CASCADE,
    player_id  bigint NOT NULL REFERENCES players (player_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, player_id)
);
