-- Every time a player permanently joins a new club's senior squad - initial squad generation, a
-- transfer, a negotiated offer, or signing as a free agent (see common.RecordPlayerClubHistory
-- and its callers) - so a player's page can show the clubs they've played for, ordered by date.
-- Deliberately excludes loans (a temporary spell, not really "joining" a new club) and academy
-- promotion (the player was already on this same team's books, just in the cantera pool).
CREATE TABLE IF NOT EXISTS player_club_history (
    history_id bigserial PRIMARY KEY,
    player_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    team_id    bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    joined_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_player_club_history_player_id ON player_club_history(player_id);
