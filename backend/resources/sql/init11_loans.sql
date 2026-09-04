-- Player loans: a temporary, free transfer to another team that auto-returns the player to its
-- original owner at a set season. loan_listings is the open market (like transfer_listings);
-- accepting one creates a player_loans row tracking where the player has to go back to and
-- when - see common/loans.go and engine.ReturnExpiredLoans.

CREATE TABLE IF NOT EXISTS loan_listings (
    listing_id       bigserial PRIMARY KEY,
    player_id        bigint NOT NULL UNIQUE REFERENCES players(player_id) ON DELETE CASCADE,
    lending_team_id  bigint NOT NULL REFERENCES teams(team_id),
    duration_seasons int NOT NULL,
    listed_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS loan_listings_idx1 ON loan_listings(lending_team_id);

CREATE TABLE IF NOT EXISTS player_loans (
    loan_id       bigserial PRIMARY KEY,
    player_id     bigint NOT NULL UNIQUE REFERENCES players(player_id) ON DELETE CASCADE,
    from_team_id  bigint NOT NULL REFERENCES teams(team_id),
    to_team_id    bigint NOT NULL REFERENCES teams(team_id),
    return_season int NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS player_loans_idx1 ON player_loans(return_season);
